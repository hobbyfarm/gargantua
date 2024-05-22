package vmservice

import (
	"context"
	"time"

	hfv1 "github.com/hobbyfarm/gargantua/v3/pkg/apis/hobbyfarm.io/v1"
	hfInformers "github.com/hobbyfarm/gargantua/v3/pkg/client/informers/externalversions"
	controllers "github.com/hobbyfarm/gargantua/v3/pkg/microservices/controller"
	"github.com/hobbyfarm/gargantua/v3/pkg/util"
	"github.com/hobbyfarm/gargantua/v3/protos/general"
	terraformpb "github.com/hobbyfarm/gargantua/v3/protos/terraform"

	"github.com/golang/glog"
	hferrors "github.com/hobbyfarm/gargantua/v3/pkg/errors"
	vmProto "github.com/hobbyfarm/gargantua/v3/protos/vm"
	vmclaimProto "github.com/hobbyfarm/gargantua/v3/protos/vmclaim"
	vmsetProto "github.com/hobbyfarm/gargantua/v3/protos/vmset"
	"k8s.io/client-go/kubernetes"
)

const (
	vmSetFinalizer = "finalizer.hobbyfarm.io/vmset"
)

type VMController struct {
	controllers.DelayingWorkqueueController
	controllers.Reconciler
	internalVmServer *GrpcVMServer
	terraformClient  terraformpb.TerraformSvcClient
	vmClaimClient    vmclaimProto.VMClaimSvcClient
	vmSetClient      vmsetProto.VMSetSvcClient
}

func NewVMController(
	kubeClient *kubernetes.Clientset,
	internalVmServer *GrpcVMServer,
	hfInformerFactory hfInformers.SharedInformerFactory,
	terraformClient terraformpb.TerraformSvcClient,
	vmClaimClient vmclaimProto.VMClaimSvcClient,
	vmSetClient vmsetProto.VMSetSvcClient,
	ctx context.Context,
) (*VMController, error) {
	vmInformer := hfInformerFactory.Hobbyfarm().V1().VirtualMachines().Informer()
	delayingWorkqueueController := *controllers.NewDelayingWorkqueueController(
		ctx,
		vmInformer,
		kubeClient,
		"vm-controller",
		time.Minute*30,
		nil,
	)

	vmController := &VMController{
		DelayingWorkqueueController: delayingWorkqueueController,
		internalVmServer:            internalVmServer,
		terraformClient:             terraformClient,
		vmClaimClient:               vmClaimClient,
		vmSetClient:                 vmSetClient,
	}
	vmController.SetReconciler(vmController)

	return vmController, nil
}

func (v *VMController) Reconcile(objName string) error {
	glog.V(8).Infof("reconciling vm %s inside vm controller", objName)
	// fetch vm
	vm, err := v.internalVmServer.GetVM(v.Context, &general.GetRequest{Id: objName})
	if err != nil {
		if hferrors.IsGrpcNotFound(err) {
			glog.Infof("vm %s not found on queue.. ignoring", objName)
			return nil
		} else {
			glog.Errorf("error while retrieving vm %s from queue with err %v", objName, err)
			return err
		}
	}

	// trigger reconcile on vmClaims only when associated VM is running
	// this should avoid triggering unwanted reconciles of VMClaims until the VM's are running
	if vm.GetVmClaimId() != "" && vm.GetStatus().GetStatus() == string(hfv1.VmStatusRunning) {
		v.vmClaimClient.AddToWorkqueue(v.Context, &general.ResourceId{Id: vm.GetVmClaimId()})
	}
	if vm.GetVmSetId() != "" && vm.GetDeletionTimestamp() != nil {
		glog.V(4).Infof("requeuing vmset %s to account for tainted vm %s", vm.GetVmSetId(), vm.GetId())
		updatedVmFinalizers := util.RemoveFinalizer(vm.GetFinalizers(), vmSetFinalizer)
		_, err := v.internalVmServer.UpdateVM(v.Context, &vmProto.UpdateVMRequest{Id: vm.GetId(), Finalizers: &general.StringArray{
			Values: updatedVmFinalizers,
		}})
		if err != nil {
			glog.Errorf("error removing vm finalizer on vm %s", vm.GetId())
			return err
		}
		v.vmSetClient.AddToWorkqueue(v.Context, &general.ResourceId{Id: vm.GetVmSetId()})
	}
	return nil
}
