package vmsetservice

import (
	"context"
	"fmt"
	"math/rand"
	"strings"
	"time"

	hfv1 "github.com/hobbyfarm/gargantua/v3/pkg/apis/hobbyfarm.io/v1"
	hfInformers "github.com/hobbyfarm/gargantua/v3/pkg/client/informers/externalversions"
	hflabels "github.com/hobbyfarm/gargantua/v3/pkg/labels"
	"github.com/hobbyfarm/gargantua/v3/pkg/util"
	"google.golang.org/protobuf/types/known/wrapperspb"

	"github.com/golang/glog"
	"k8s.io/apimachinery/pkg/labels"

	hferrors "github.com/hobbyfarm/gargantua/v3/pkg/errors"
	controllers "github.com/hobbyfarm/gargantua/v3/pkg/microservices/controller"
	environmentpb "github.com/hobbyfarm/gargantua/v3/protos/environment"
	generalpb "github.com/hobbyfarm/gargantua/v3/protos/general"
	vmpb "github.com/hobbyfarm/gargantua/v3/protos/vm"
	vmsetpb "github.com/hobbyfarm/gargantua/v3/protos/vmset"
	vmtemplatepb "github.com/hobbyfarm/gargantua/v3/protos/vmtemplate"
	"k8s.io/client-go/kubernetes"
)

const (
	vmSetFinalizer = "finalizer.hobbyfarm.io/vmset"
)

type VMSetController struct {
	controllers.DelayingWorkqueueController
	controllers.Reconciler
	internalVmSetServer *GrpcVMSetServer
	environmentClient   environmentpb.EnvironmentSvcClient
	vmClient            vmpb.VMSvcClient
	vmTemplateClient    vmtemplatepb.VMTemplateSvcClient
}

func NewVMSetController(
	kubeClient *kubernetes.Clientset,
	internalVmSetServer *GrpcVMSetServer,
	hfInformerFactory hfInformers.SharedInformerFactory,
	environmentClient environmentpb.EnvironmentSvcClient,
	vmClient vmpb.VMSvcClient,
	vmTemplateClient vmtemplatepb.VMTemplateSvcClient,
	ctx context.Context,
) (*VMSetController, error) {
	vmSetInformer := hfInformerFactory.Hobbyfarm().V1().VirtualMachineSets().Informer()
	delayingWorkqueueController := *controllers.NewDelayingWorkqueueController(
		ctx,
		vmSetInformer,
		kubeClient,
		"vmset-controller",
		time.Minute*30,
		internalVmSetServer.vmSetWorkqueue,
	)

	vmSetController := &VMSetController{
		DelayingWorkqueueController: delayingWorkqueueController,
		internalVmSetServer:         internalVmSetServer,
		environmentClient:           environmentClient,
		vmClient:                    vmClient,
		vmTemplateClient:            vmTemplateClient,
	}
	vmSetController.SetReconciler(vmSetController)
	vmSetController.SetWorkScheduler(vmSetController)

	return vmSetController, nil
}

func (v *VMSetController) Reconcile(objName string) error {
	glog.V(8).Infof("reconciling vmclaim %s inside vm claim controller", objName)
	// fetch vmClaim
	vmSet, err := v.internalVmSetServer.GetVMSet(v.Context, &generalpb.GetRequest{Id: objName})
	if err != nil {
		if hferrors.IsGrpcNotFound(err) {
			glog.Infof("vmset %s not found on queue.. ignoring", objName)
			//v.vmSetWorkqueue.Forget(obj)
			return nil
		} else {
			return fmt.Errorf("error while retrieving vmset %s from queue with err %v", objName, err)
		}
	}

	err = v.reconcileVirtualMachineSet(vmSet)

	//v.vmSetWorkqueue.Forget(obj)
	glog.V(4).Infof("vm set processed by vm set controller %v", objName)
	return err
}

func (v *VMSetController) enqueueVMSet(vmSetId string) {
	glog.V(8).Infof("Enqueueing vm set %s", vmSetId)
	//v.vmSetWorkqueue.AddRateLimited(key)
	v.GetWorkqueue().Add(vmSetId)
}

func (v *VMSetController) reconcileVirtualMachineSet(vmset *vmsetpb.VMSet) error {
	vmLabels := labels.Set{
		"vmset": vmset.GetId(),
	}
	currentVMList, err := v.vmClient.ListVM(v.Context, &generalpb.ListOptions{
		LabelSelector: vmLabels.AsSelector().String(),
		LoadFromCache: true,
	})

	if err != nil {
		glog.Errorf("error listing vms in vmset controller")
		return err
	}

	currentVMs := currentVMList.GetVms()

	if len(currentVMs) < int(vmset.GetCount()) { // if desired count is greater than the current provisioned
		// 1. let's check the environment to see if there is available capacity
		// 2. if available capacity is available let's create new VM's
		glog.V(4).Infof("vmset %s needs %d vm's but current vm count is %d", vmset.GetId(), vmset.GetCount(), len(currentVMs))
		env, err := v.environmentClient.GetEnvironment(v.Context, &generalpb.GetRequest{
			Id:            vmset.GetEnvironment(),
			LoadFromCache: true,
		})
		var provision bool
		provision = true
		if provisionMethod, ok := env.GetAnnotations()["hobbyfarm.io/provisioner"]; ok && provisionMethod != "" {
			provision = false
		}
		if err != nil {
			if hferrors.IsGrpcNotFound(err) {
				glog.Errorf("environment invalid")
			}
			return err
		}

		vmt, err := v.vmTemplateClient.GetVMTemplate(v.Context, &generalpb.GetRequest{
			Id:            vmset.GetVmTemplate(),
			LoadFromCache: true,
		})

		if err != nil {
			return fmt.Errorf("error while retrieving virtual machine template %s %v", vmset.GetVmTemplate(), err)
		}
		needed := int(vmset.GetCount()) - len(currentVMs)

		glog.V(5).Infof("provisioning %d vms", needed)
		for i := 0; i < needed; i++ {
			vmName := strings.Join([]string{vmset.GetBaseName(), fmt.Sprintf("%08x", rand.Uint32())}, "-")
			config := util.GetVMConfig(env, vmt)
			sshUser := config["ssh_username"]
			protocol, exists := config["protocol"]
			if !exists {
				protocol = "ssh"
			}
			restrictedBind := vmset.GetRestrictedBind()

			vmLabels := map[string]string{
				"dynamic":                       "false",
				"vmset":                         vmset.GetId(),
				hflabels.VirtualMachineTemplate: vmt.GetId(),
				hflabels.EnvironmentLabel:       env.GetId(),
				"bound":                         "false",
				"ready":                         "false",
				hflabels.ScheduledEventLabel:    vmset.GetLabels()[hflabels.ScheduledEventLabel],
				"restrictedbind":                fmt.Sprintf("%t", restrictedBind),
			}
			if restrictedBind {
				vmLabels["restrictedbindvalue"] = vmset.GetRestrictedBindValue()
			}
			if provisionMethod, ok := env.GetAnnotations()["hobbyfarm.io/provisioner"]; ok && provisionMethod != "" {
				vmLabels["hobbyfarm.io/provisioner"] = provisionMethod
			}

			_, err := v.vmClient.CreateVM(v.Context, &vmpb.CreateVMRequest{
				Id:           vmName,
				VmTemplateId: vmt.GetId(),
				SshUsername:  sshUser,
				Protocol:     protocol,
				SecretName:   "",
				User:         "",
				Provision:    provision,
				VmSetId:      vmset.GetId(),
				VmSetUid:     vmset.GetUid(),
				Labels:       vmLabels,
				Finalizers:   []string{vmSetFinalizer},
			})

			if err != nil {
				glog.Error(err)
			}

			_, err = v.vmClient.UpdateVMStatus(v.Context, &vmpb.UpdateVMStatusRequest{
				Id:            vmName,
				Status:        string(hfv1.VmStatusRFP),
				Allocated:     wrapperspb.Bool(false),
				Tainted:       wrapperspb.Bool(false),
				WsEndpoint:    env.GetWsEndpoint(),
				PublicIp:      wrapperspb.String(""),
				PrivateIp:     wrapperspb.String(""),
				EnvironmentId: env.GetId(),
				Hostname:      wrapperspb.String(""),
			})

			if err != nil {
				glog.Error(err)
			}
		}
	}

	// handle case of scaling down VMSets
	if len(currentVMs) > int(vmset.GetCount()) {
		// We first calculate how many VMs already have been deleted to avoid deleting more than we need
		currentlyDeleting := 0
		for _, x := range currentVMs {
			if x.GetDeletionTimestamp() != nil {
				currentlyDeleting++
			}
		}

		// We need to delete all over the spec.count minus the VMs that are already being deleted right now.
		needed_delete := len(currentVMs) - int(vmset.GetCount()) - currentlyDeleting
		glog.V(4).Infof("vmset %s needs to delete %d vm's and %d are already flagged as deleted", vmset.GetId(), needed_delete, currentlyDeleting)
		for _, cur_vm := range currentVMs {
			if needed_delete == 0 {
				break
			}

			if !cur_vm.GetStatus().GetAllocated() && cur_vm.GetDeletionTimestamp() == nil {
				_, err = v.vmClient.DeleteVM(v.Context, &generalpb.ResourceId{Id: cur_vm.GetId()})
				if err != nil {
					glog.Errorf("error deleting vm %s with error: %v", cur_vm.GetId(), err)
				} else {
					needed_delete--
				}
			}
		}
		if needed_delete > 0 {
			glog.V(4).Infof("vmset %d could not delete %d VMs due to some VMs being in use.", vmset.GetId(), needed_delete)
		}
	}

	vmList, err := v.vmClient.ListVM(v.Context, &generalpb.ListOptions{
		LabelSelector: vmLabels.AsSelector().String(),
		LoadFromCache: true,
	})

	if err != nil {
		glog.Errorf("error while retrieving vms owned by vmset %s", vmset.GetId())
	}

	vms := vmList.GetVms()

	provisionedCount := 0
	activeCount := 0
	for _, x := range vms {
		if x.GetDeletionTimestamp() == nil && !x.GetStatus().GetTainted() {
			activeCount++
		}
		provisionedCount++
	}

	if activeCount < int(vmset.GetCount()) {
		glog.V(4).Infof("requeing VMset as there are not enough VMs ready")
		v.enqueueVMSet(vmset.GetId())
	}

	err = v.updateVMSetCount(vmset.GetId(), activeCount, provisionedCount)

	return err
}

func (v *VMSetController) updateVMSetCount(vmSetName string, active int, prov int) error {
	_, err := v.internalVmSetServer.UpdateVMSetStatus(v.Context, &vmsetpb.UpdateVMSetStatusRequest{
		Available:   wrapperspb.UInt32(uint32(active)),
		Provisioned: wrapperspb.UInt32(uint32(prov)),
	})
	if err != nil {
		return fmt.Errorf("error updating Virtual Machine Set Status: %s, %v", vmSetName, err)
	}
	return nil
}
