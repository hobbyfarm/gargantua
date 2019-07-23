package vmclaimcontroller

import (
	"fmt"
	"github.com/golang/glog"
	hfv1 "github.com/hobbyfarm/gargantua/pkg/apis/hobbyfarm.io/v1"
	hfClientset "github.com/hobbyfarm/gargantua/pkg/client/clientset/versioned"
	hfInformers "github.com/hobbyfarm/gargantua/pkg/client/informers/externalversions"
	hfListers "github.com/hobbyfarm/gargantua/pkg/client/listers/hobbyfarm.io/v1"
	"github.com/hobbyfarm/gargantua/pkg/util"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/retry"
	"k8s.io/client-go/util/workqueue"
	"time"
)
type VMClaimController struct {
	hfClientSet *hfClientset.Clientset

	vmLister hfListers.VirtualMachineLister
	vmClaimLister hfListers.VirtualMachineClaimLister

	vmClaimWorkqueue workqueue.RateLimitingInterface

	vmClaimHasSynced cache.InformerSynced
	vmHasSynced cache.InformerSynced

}

func NewVMClaimController(hfClientSet *hfClientset.Clientset, hfInformerFactory hfInformers.SharedInformerFactory) (*VMClaimController, error) {
	vmClaimController := VMClaimController{}
	vmClaimController.hfClientSet = hfClientSet

	vmClaimController.vmLister = hfInformerFactory.Hobbyfarm().V1().VirtualMachines().Lister()
	vmClaimController.vmClaimLister = hfInformerFactory.Hobbyfarm().V1().VirtualMachineClaims().Lister()

	vmClaimController.vmClaimWorkqueue = workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "VMClaims")

	vmClaimInformer := hfInformerFactory.Hobbyfarm().V1().VirtualMachineClaims().Informer()

	vmClaimInformer.AddEventHandlerWithResyncPeriod(cache.ResourceEventHandlerFuncs{
		AddFunc: vmClaimController.enqueueVMClaim,
		UpdateFunc: func(old, new interface{}) {
			vmClaimController.enqueueVMClaim(new)
		},
		DeleteFunc: vmClaimController.enqueueVMClaim,
	}, time.Second * 30)

	vmClaimController.vmClaimHasSynced = hfInformerFactory.Hobbyfarm().V1().VirtualMachineClaims().Informer().HasSynced
	vmClaimController.vmHasSynced = hfInformerFactory.Hobbyfarm().V1().VirtualMachines().Informer().HasSynced

	return &vmClaimController, nil
}


func (v *VMClaimController) enqueueVMClaim(obj interface{}) {
	var key string
	var err error
	if key, err = cache.MetaNamespaceKeyFunc(obj); err != nil {
		//utilruntime.HandleError(err)
		return
	}
	glog.V(8).Infof("Enqueueing vm claim %v", key)
	v.vmClaimWorkqueue.AddRateLimited(key)
}

func (v *VMClaimController) Run(stopCh <-chan struct{}) error {
	defer v.vmClaimWorkqueue.ShutDown()

	glog.V(4).Infof("Starting vm claim controller")
	glog.Info("Waiting for informer caches to sync")
	if ok := cache.WaitForCacheSync(stopCh, v.vmClaimHasSynced, v.vmHasSynced); !ok {
		return fmt.Errorf("failed to wait for caches to sync")
	}
	glog.Info("Starting vm claim worker")

	go wait.Until(v.runVMClaimWorker, time.Second, stopCh)

	//if ok := cache.WaitForCacheSync(stopCh, )
	<-stopCh
	return nil
}

func (v *VMClaimController) runVMClaimWorker() {
	for v.processNextVMClaim() {

	}
}

func (v *VMClaimController) processNextVMClaim() bool {
	obj, shutdown := v.vmClaimWorkqueue.Get()

	glog.V(8).Infof("processing VM Claim")

	if shutdown {
		return false
	}

	err := func() error {
		defer v.vmClaimWorkqueue.Done(obj)
		glog.V(4).Infof("processing vm claim in env controller: %v", obj)
		_, objName, err := cache.SplitMetaNamespaceKey(obj.(string)) // this is actually not necessary because VM's are not namespaced yet...
		if err != nil {
			glog.Errorf("error while splitting meta namespace key %v", err)
			return nil
		}

		vmClaim, err := v.hfClientSet.HobbyfarmV1().VirtualMachineClaims().Get(objName, metav1.GetOptions{})
		if err != nil {
			glog.Errorf("error while retrieving virtual machine claim %s, likely deleted %v", objName, err)
			v.vmClaimWorkqueue.Forget(obj)
			return nil
		}

		if vmClaim.Status.Bound {
			v.vmClaimWorkqueue.Forget(obj)
			glog.V(8).Infof("vm claim %s already bound, forgetting", objName)
			return nil
		}

		needed := make(map[string]int)

		for _, template := range vmClaim.Spec.VirtualMachines {
			if template.VirtualMachineId == "" {
				if val, ok := needed[template.Template]; ok {
					needed[template.Template] = val + 1
				} else {
					needed[template.Template] = 1
				}
			}
		}

		if len(needed) == 0 {
			glog.V(8).Infof("vm claim %s does not need any vms, marking ready and bound", objName)
			v.updateVMClaimStatus(true, true, objName)
			return nil
		}

		envList, err := v.hfClientSet.HobbyfarmV1().Environments().List(metav1.ListOptions{})

		if err != nil {
			glog.Error(err)
		}

		var chosenEnvironmentId string

		for _, env := range envList.Items {
			acceptable := true
			for t, n := range needed {
				vms, err := v.vmLister.List(labels.Set{
					"bound": "false",
					"environment": env.Name,
					"ready": "true", // do we really want to be marking ready as a label
					"template": t,
				}.AsSelector())

				if err != nil {
					glog.Error(err)
					acceptable = false
					break
				}

				if len(vms) < n {
					acceptable = false
					break
				}
			}
			if acceptable {
				chosenEnvironmentId = env.Name // @todo: change to using the ID instead of name
				break
			}
		}

		if chosenEnvironmentId == "" {
			glog.Errorf("error while trying to find matching environment for vm claim %s", vmClaim.Name)
			v.vmClaimWorkqueue.AddRateLimited(obj)
			return nil
		}


		for name, vmStruct := range vmClaim.Spec.VirtualMachines {
			if vmStruct.VirtualMachineId == "" {
				vmId, err := v.assignNextFreeVM(vmClaim.Spec.Id, vmClaim.Spec.UserId, vmStruct.Template, chosenEnvironmentId)
				if err != nil {
					glog.Fatalf("error while assigning next free VM %v", err)
				}

				err = v.updateVMClaimWithVM(name, vmId, vmClaim.Spec.Id)
				if err != nil {
					glog.Fatalf("error while updating VM Claim with VM %v", err)
				}
			}
		}

		v.updateVMClaimStatus(true, true, vmClaim.Spec.Id)

		v.vmClaimWorkqueue.Forget(obj)
		glog.V(4).Infof("vmclaim processed and assigned by controller %v", objName)

		return nil

	}()

	if err != nil {
		return true
	}

	return true
}

func (v *VMClaimController) assignNextFreeVM(vmClaimId string, user string, template string, environmentId string) (string, error) {
	vms, err := v.vmLister.List(labels.Set{
		"bound": "false",
		"environment": environmentId,
		"ready": "true",
		"template": template,
	}.AsSelector())

	if err != nil {
		return  "", fmt.Errorf("error while listing all vms %v", err)
	}

	assigned := false
	vmId := ""
	for _, vm := range vms {
		if !vm.Status.Allocated && vm.Status.Status == hfv1.VmStatusRunning && !vm.Status.Tainted {
			// we can assign this vm
			assigned = true
			vmId = vm.Spec.Id

			retryErr := retry.RetryOnConflict(retry.DefaultRetry, func() error {
				result, getErr := v.hfClientSet.HobbyfarmV1().VirtualMachines().Get(vmId, metav1.GetOptions{})
				if getErr != nil {
					return fmt.Errorf("Error retrieving latest version of Virtual Machine %s: %v", vmId, getErr)
				}

				result.Status.Allocated = true
				result.Spec.VirtualMachineClaimId = vmClaimId
				result.Spec.UserId = user

				result.Labels["bound"] = "true"

				vm, updateErr := v.hfClientSet.HobbyfarmV1().VirtualMachines().Update(result)
				if updateErr != nil {
					return updateErr
				}
				glog.V(4).Infof("updated result for virtual machine")

				verifyErr := util.VerifyVM(v.vmLister, vm)

				if verifyErr != nil {
					return verifyErr
				}
				return nil
			})
			if retryErr != nil {
				return "", fmt.Errorf("Error updating Virtual Machine: %s, %v", vmId, retryErr)
			}
			break}
	}

	if assigned {
		return vmId, nil
	}

	return vmId, fmt.Errorf("unknown error while assigning next free vm")

}

func (v *VMClaimController) rollbackAssignmentOfVM(vmId string) (error) {

			retryErr := retry.RetryOnConflict(retry.DefaultRetry, func() error {
				result, getErr := v.hfClientSet.HobbyfarmV1().VirtualMachines().Get(vmId, metav1.GetOptions{})
				if getErr != nil {
					return fmt.Errorf("Error retrieving latest version of Virtual Machine %s: %v", vmId, getErr)
				}

				result.Status.Allocated = false
				result.Spec.VirtualMachineClaimId = ""
				result.Spec.UserId = ""

				_, updateErr := v.hfClientSet.HobbyfarmV1().VirtualMachines().Update(result)
				glog.V(4).Infof("updated result for virtual machine")


				return updateErr
			})
			if retryErr != nil {
				return fmt.Errorf("Error updating Virtual Machine: %s, %v", vmId, retryErr)
			}

	return fmt.Errorf("unknown error while rolling back assignment of vm")

}

func (v *VMClaimController) updateVMClaimWithVM(vmName string, vmId string, vmClaimId string) (error) {

	retryErr := retry.RetryOnConflict(retry.DefaultRetry, func() error {
		result, getErr := v.hfClientSet.HobbyfarmV1().VirtualMachineClaims().Get(vmClaimId, metav1.GetOptions{})
		if getErr != nil {
			return fmt.Errorf("Error retrieving latest version of Virtual Machine Claim %s: %v", vmClaimId, getErr)
		}

		vmClaimVM := result.Spec.VirtualMachines[vmName]

		vmClaimVM.VirtualMachineId = vmId

		result.Spec.VirtualMachines[vmName] = vmClaimVM

		vmc, updateErr := v.hfClientSet.HobbyfarmV1().VirtualMachineClaims().Update(result)
		glog.V(4).Infof("updated result for virtual machine claim")

		if updateErr != nil {
			return updateErr
		}

		verifyErr := util.VerifyVMClaim(v.vmClaimLister, vmc)
		if verifyErr != nil {
			return verifyErr
		}
		return updateErr
	})
	if retryErr != nil {
		return fmt.Errorf("Error updating Virtual Machine Claim: %s, %v", vmClaimId, retryErr)
	}
	return nil
}

func (v *VMClaimController) updateVMClaimStatus(bound bool, ready bool, vmClaimId string) (error) {

	retryErr := retry.RetryOnConflict(retry.DefaultRetry, func() error {
		result, getErr := v.hfClientSet.HobbyfarmV1().VirtualMachineClaims().Get(vmClaimId, metav1.GetOptions{})
		if getErr != nil {
			return fmt.Errorf("Error retrieving latest version of Virtual Machine Claim %s: %v", vmClaimId, getErr)
		}

		result.Status.Bound = bound
		result.Status.Ready = ready

		vmc, updateErr := v.hfClientSet.HobbyfarmV1().VirtualMachineClaims().Update(result)
		if updateErr != nil {
			return updateErr
		}
		glog.V(4).Infof("updated result for virtual machine claim")

		verifyErr := util.VerifyVMClaim(v.vmClaimLister, vmc)

		if verifyErr != nil {
			return verifyErr
		}
		return nil
	})
	if retryErr != nil {
		return fmt.Errorf("Error updating Virtual Machine Claim: %s, %v", vmClaimId, retryErr)
	}
	return nil
}