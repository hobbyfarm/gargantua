package vmclaimcontroller

import (
	"fmt"
	"github.com/golang/glog"
	hfClientset "github.com/hobbyfarm/gargantua/pkg/client/clientset/versioned"
	hfInformers "github.com/hobbyfarm/gargantua/pkg/client/informers/externalversions"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/retry"
	"k8s.io/client-go/util/workqueue"
	"time"
)
type VMClaimController struct {
	hfClientSet *hfClientset.Clientset
	vmClaimWorkqueue workqueue.RateLimitingInterface

	vmClaimHasSynced cache.InformerSynced

}

func NewVMClaimController(hfClientSet *hfClientset.Clientset, hfInformerFactory hfInformers.SharedInformerFactory) (*VMClaimController, error) {
	vmClaimController := VMClaimController{}
	vmClaimController.hfClientSet = hfClientSet
	vmClaimController.vmClaimWorkqueue = workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "VMClaims")
	//vmClaimController.hfInformerFactory = hfInformerFactory

	vmClaimInformer := hfInformerFactory.Hobbyfarm().V1().VirtualMachineClaims().Informer()

	vmClaimInformer.AddEventHandlerWithResyncPeriod(cache.ResourceEventHandlerFuncs{
		AddFunc: vmClaimController.enqueueVMClaim,
		UpdateFunc: func(old, new interface{}) {
			vmClaimController.enqueueVMClaim(new)
		},
		DeleteFunc: vmClaimController.enqueueVMClaim,
	}, time.Second * 30)

	vmClaimController.vmClaimHasSynced = hfInformerFactory.Hobbyfarm().V1().VirtualMachineClaims().Informer().HasSynced

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
	if ok := cache.WaitForCacheSync(stopCh, v.vmClaimHasSynced); !ok {
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

	glog.V(4).Infof("processing VM Claim")

	if shutdown {
		return false
	}

	//glog.V(4).Infof("vmc %v", obj)

	err := func() error {
		defer v.vmClaimWorkqueue.Done(obj)
		glog.V(4).Infof("processing vm claim in env controller: %v", obj)
		_, objName, err := cache.SplitMetaNamespaceKey(obj.(string)) // this is actually not necessary because VM's are not namespaced yet...
		if err != nil {
			glog.Errorf("error while splitting meta namespace key %v", err)
			//e.vmWorkqueue.AddRateLimited(obj)
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
			glog.V(4).Infof("vm claim %s already bound", objName)
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
			glog.V(4).Infof("vm claim does not need any vms, marking ready and bound")
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
				if a, ok := env.Status.AvailableCount[t]; !ok {
					acceptable = false // environment doesn't have available vm's of this type
					break
				} else {
					if a < n {
						acceptable = false // environment doesn't have enough vm's of this type
						break
					}
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

		glog.V(4).Infof("chose environment %s for vm claim %s", chosenEnvironmentId, vmClaim.Name)

		// now we can assign the vm's from chosenEnvironmentId

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
	return false
}

func (v *VMClaimController) assignNextFreeVM(vmClaimId string, user string, template string, environmentId string) (string, error) {
	vms, err := v.hfClientSet.HobbyfarmV1().VirtualMachines().List(metav1.ListOptions{})

	if err != nil {
		return  "", fmt.Errorf("error while listing all vms %v", err)
	}

	assigned := false
	vmId := ""
	for _, vm := range vms.Items {
		if !vm.Status.Allocated && vm.Status.EnvironmentId == environmentId  && vm.Spec.VirtualMachineTemplateId == template {
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

				_, updateErr := v.hfClientSet.HobbyfarmV1().VirtualMachines().Update(result)
				glog.V(4).Infof("updated result for virtual machine")


				return updateErr
			})
			if retryErr != nil {
				return "", fmt.Errorf("Error updating Virtual Machine: %s, %v", vmId, retryErr)
			}
			break
		}
	}

	if assigned {
		return vmId, nil
	}

	return vmId, fmt.Errorf("unknown error while assigning next free vm")

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

		_, updateErr := v.hfClientSet.HobbyfarmV1().VirtualMachineClaims().Update(result)
		glog.V(4).Infof("updated result for virtual machine claim")

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

		_, updateErr := v.hfClientSet.HobbyfarmV1().VirtualMachineClaims().Update(result)
		glog.V(4).Infof("updated result for virtual machine claim")

		return updateErr
	})
	if retryErr != nil {
		return fmt.Errorf("Error updating Virtual Machine Claim: %s, %v", vmClaimId, retryErr)
	}
	return nil
}