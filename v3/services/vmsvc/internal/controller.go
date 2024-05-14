package vmservice

import (
	"context"
	"time"

	hfv1 "github.com/hobbyfarm/gargantua/v3/pkg/apis/hobbyfarm.io/v1"
	hfInformers "github.com/hobbyfarm/gargantua/v3/pkg/client/informers/externalversions"
	controllers "github.com/hobbyfarm/gargantua/v3/pkg/microservices/controller"
	"github.com/hobbyfarm/gargantua/v3/protos/general"

	"github.com/golang/glog"
	hferrors "github.com/hobbyfarm/gargantua/v3/pkg/errors"
	vmclaimProto "github.com/hobbyfarm/gargantua/v3/protos/vmclaim"
	"k8s.io/client-go/kubernetes"
)

type VMController struct {
	controllers.DelayingWorkqueueController
	controllers.Reconciler
	internalVmServer *GrpcVMServer
	vmClaimClient    vmclaimProto.VMClaimSvcClient
}

func NewVMController(
	kubeClient *kubernetes.Clientset,
	internalVmServer *GrpcVMServer,
	hfInformerFactory hfInformers.SharedInformerFactory,
	vmClaimClient vmclaimProto.VMClaimSvcClient,
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
		vmClaimClient:               vmClaimClient,
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
	if vm.VmClaimId != "" && vm.Status.Status == string(hfv1.VmStatusRunning) {
		v.vmClaimClient.AddToWorkqueue(v.Context, &general.ResourceId{Id: vm.VmClaimId})
	}
	return nil
}
