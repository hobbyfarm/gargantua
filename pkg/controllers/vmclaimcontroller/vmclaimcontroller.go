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
	"math/rand"
	"strings"
	"time"
)

const (
	StaticBindAttemptThreshold  int = 3
	DynamicBindAttemptThreshold int = 2
)

type VMClaimController struct {
	hfClientSet *hfClientset.Clientset

	vmLister      hfListers.VirtualMachineLister
	vmClaimLister hfListers.VirtualMachineClaimLister

	vmClaimWorkqueue workqueue.Interface

	vmWorkqueue workqueue.Interface

	vmClaimHasSynced cache.InformerSynced
	vmHasSynced      cache.InformerSynced
}

func NewVMClaimController(hfClientSet *hfClientset.Clientset, hfInformerFactory hfInformers.SharedInformerFactory) (*VMClaimController, error) {
	vmClaimController := VMClaimController{}
	vmClaimController.hfClientSet = hfClientSet

	vmClaimController.vmLister = hfInformerFactory.Hobbyfarm().V1().VirtualMachines().Lister()
	vmClaimController.vmClaimLister = hfInformerFactory.Hobbyfarm().V1().VirtualMachineClaims().Lister()

	vmClaimController.vmClaimWorkqueue = workqueue.New()
	vmClaimController.vmWorkqueue = workqueue.New()

	vmClaimInformer := hfInformerFactory.Hobbyfarm().V1().VirtualMachineClaims().Informer()

	vmClaimInformer.AddEventHandlerWithResyncPeriod(cache.ResourceEventHandlerFuncs{
		AddFunc: vmClaimController.enqueueVMClaim,
		UpdateFunc: func(old, new interface{}) {
			vmClaimController.enqueueVMClaim(new)
		},
		DeleteFunc: vmClaimController.enqueueVMClaim,
	}, time.Minute*30)

	vmInformer := hfInformerFactory.Hobbyfarm().V1().VirtualMachines().Informer()

	vmInformer.AddEventHandlerWithResyncPeriod(cache.ResourceEventHandlerFuncs{
		AddFunc: vmClaimController.enqueueVM,
		UpdateFunc: func(old, new interface{}) {

		},
		DeleteFunc: vmClaimController.enqueueVM,
	}, time.Minute*30)

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
	v.vmClaimWorkqueue.Add(key)
}

func (v *VMClaimController) enqueueVM(obj interface{}) {
	var key string
	var err error
	if key, err = cache.MetaNamespaceKeyFunc(obj); err != nil {
		return
	}
	glog.V(8).Infof("enqueueing vm %v in vm claim controller to inform vmclaim if exists", key)
	v.vmWorkqueue.Add(key)
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
	go wait.Until(v.runVMWorker, time.Second, stopCh)
	//if ok := cache.WaitForCacheSync(stopCh, )
	<-stopCh
	return nil
}

func (v *VMClaimController) runVMClaimWorker() {
	for v.processNextVMClaim() {

	}
}

func (v *VMClaimController) runVMWorker() {
	for v.processNextVM() {

	}
}

func (v *VMClaimController) processNextVM() bool {
	obj, shutdown := v.vmWorkqueue.Get()
	glog.V(8).Infof("processing VM in vm claim controller for update")

	if shutdown {
		return false
	}

	err := func() error {
		defer v.vmWorkqueue.Done(obj)
		_, objName, err := cache.SplitMetaNamespaceKey(obj.(string))
		vm, err := v.hfClientSet.HobbyfarmV1().VirtualMachines().Get(objName, metav1.GetOptions{})

		if err != nil {
			// ideally should put logic here to determine if we need to retry and push this vm back onto the workqueue
			glog.Errorf("error while retrieving vm %s: %v", objName, err)
			return nil
		}

		if vm.Spec.VirtualMachineClaimId != "" {
			v.vmClaimWorkqueue.Add(vm.Spec.VirtualMachineClaimId)
		}
		return nil
	}()

	if err != nil {
		glog.Errorf("vm claim controller process next vm returned an error %v", err)
		return true
	}
	return true
}

func (v *VMClaimController) processNextVMClaim() bool {
	obj, shutdown := v.vmClaimWorkqueue.Get()

	glog.V(8).Infof("processing VM Claim")

	if shutdown {
		return false
	}

	err := func() error {
		defer v.vmClaimWorkqueue.Done(obj)
		glog.V(4).Infof("processing vm claim in vm claim controller: %v", obj)
		_, objName, err := cache.SplitMetaNamespaceKey(obj.(string)) // this is actually not necessary because VM's are not namespaced yet...
		if err != nil {
			glog.Errorf("error while splitting meta namespace key %v", err)
			return nil
		}

		vmClaim, err := v.hfClientSet.HobbyfarmV1().VirtualMachineClaims().Get(objName, metav1.GetOptions{})
		if err != nil {
			glog.Errorf("error while retrieving virtual machine claim %s, likely deleted %v", objName, err)
			//v.vmClaimWorkqueue.Forget(obj)
			return nil
		}

		if vmClaim.Status.Tainted {
			//v.vmClaimWorkqueue.Forget(obj)
			glog.V(8).Infof("vm claim %s tainted, forgetting", objName)
			return nil
		}

		if vmClaim.Status.Bound && vmClaim.Status.Ready {
			//v.vmClaimWorkqueue.Forget(obj)
			glog.V(8).Infof("vm claim %s already bound and ready, forgetting", objName)
			return nil
		}

		if vmClaim.Status.Bound && !vmClaim.Status.Ready {
			vmClaimIsReady := true
			for _, needed := range vmClaim.Spec.VirtualMachines {
				if needed.VirtualMachineId != "" {
					vm, err := v.hfClientSet.HobbyfarmV1().VirtualMachines().Get(needed.VirtualMachineId, metav1.GetOptions{})

					if err != nil {
						glog.Errorf("error while retrieving vm from k8s api: %v", err)
						//v.vmClaimWorkqueue.AddRateLimited(obj)
						v.vmClaimWorkqueue.Add(obj)
						return nil
					}

					if vm.Status.Status != hfv1.VmStatusRunning {
						vmClaimIsReady = false
						break
					}
				} else {
					glog.Errorf("found vm claim marked as bound but vm ID was not populated")
					vmClaimIsReady = false
					break
				}
			}

			if vmClaimIsReady {
				v.updateVMClaimStatus(true, true, vmClaim.Spec.Id)
				//v.vmClaimWorkqueue.Forget(obj)
				glog.V(8).Infof("vm claim %s is now bound and ready, forgetting", objName)
				return nil
			}
			glog.V(8).Infof("vm claim %s is not ready yet, requeuing", objName)
			//v.vmClaimWorkqueue.AddRateLimited(obj)
			v.vmClaimWorkqueue.Add(obj)
			return nil
		}

		if vmClaim.Status.BindMode == "dynamic" {

			// let's check to see if there is an active DynamicBindRequest

			dynamicBindRequest, err := v.hfClientSet.HobbyfarmV1().DynamicBindRequests().Get(vmClaim.Status.DynamicBindRequestId, metav1.GetOptions{})

			if err != nil {
				glog.Errorf("Error while attempting to retrieve the dynamic bind request. Perhaps this is a transient error, queuing again.")
				//v.vmClaimWorkqueue.AddRateLimited(obj)
				v.vmClaimWorkqueue.Add(obj)
				return nil
			}

			if !dynamicBindRequest.Status.Expired {
				if dynamicBindRequest.Status.Fulfilled { // we are ready to bind this vm claim
					for vmName, vmId := range dynamicBindRequest.Status.VirtualMachineIds {
						v.updateVMClaimWithVM(vmName, vmId, objName)
					}
					v.updateVMClaimStatus(true, false, objName)
				}
			} else {
				v.updateVMClaimBindMode("static", "", vmClaim.Spec.Id)
			}
			//v.vmClaimWorkqueue.AddRateLimited(obj)
			v.vmClaimWorkqueue.Add(obj)

			return nil
		} else {

			if vmClaim.Status.BindMode != "static" && vmClaim.Status.StaticBindAttempts == 0 {
				v.updateVMClaimBindMode("static", "", vmClaim.Spec.Id)
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
				//v.vmClaimWorkqueue.Forget(obj)
				return nil
			}

			envList, err := v.hfClientSet.HobbyfarmV1().Environments().List(metav1.ListOptions{})

			if err != nil {
				glog.Error(err)
			}

			var chosenEnvironmentId string

			environments := envList.Items

			rand.Seed(time.Now().UnixNano())

			rand.Shuffle(len(environments), func(i, j int) {
				environments[i], environments[j] = environments[j], environments[i]
			})

			for _, env := range environments {
				acceptable := true
				for t, n := range needed {
					vmLabels := labels.Set{
						"bound":       "false",
						"environment": env.Name,
						"ready":       "true", // do we really want to be marking ready as a label
						"template":    t,
					}
					if vmClaim.Spec.RestrictedBind {
						vmLabels["restrictedbind"] = "true"
						vmLabels["restrictedbindvalue"] = vmClaim.Spec.RestrictedBindValue
					} else {
						vmLabels["restrictedbind"] = "false"
					}

					vms, err := v.vmLister.List(vmLabels.AsSelector())

					if err != nil {
						glog.Error(err)
						acceptable = false
						break
					}
					// if the number of vm's available in this environment is less than the number of vm's we need for this template
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
				if vmClaim.Status.StaticBindAttempts > StaticBindAttemptThreshold && vmClaim.Spec.DynamicCapable {
					// need to create a dynamic bind request
					dbrName := strings.Join([]string{vmClaim.Name + "-", fmt.Sprintf("%08x", rand.Uint32())}, "-")
					dbr := &hfv1.DynamicBindRequest{
						ObjectMeta: metav1.ObjectMeta{
							Name: dbrName,
							OwnerReferences: []metav1.OwnerReference{
								{
									APIVersion: "v1",
									Kind:       "VirtualMachineClaim",
									Name:       vmClaim.Name,
									UID:        vmClaim.UID,
								},
							},
						},
						Spec: hfv1.DynamicBindRequestSpec{
							Id:                  dbrName,
							VirtualMachineClaim: vmClaim.Spec.Id,
							Attempts:            DynamicBindAttemptThreshold,
						},
						Status: hfv1.DynamicBindRequestStatus{
							CurrentAttempts:            0,
							Expired:                    false,
							Fulfilled:                  false,
							DynamicBindConfigurationId: "",
						},
					}

					dbr, err := v.hfClientSet.HobbyfarmV1().DynamicBindRequests().Create(dbr)
					if err != nil {
						glog.Errorf("Error creating dynamic bind request for VMClaim %s: %v", vmClaim.Spec.Id, err)
						//v.vmClaimWorkqueue.AddRateLimited(obj)
						v.vmClaimWorkqueue.Add(obj)
						return nil
					}

					v.updateVMClaimBindMode("dynamic", dbr.Spec.Id, vmClaim.Spec.Id)
					glog.V(6).Infof("Created dynamic bind request %s for VM Claim %s", dbr.Spec.Id, vmClaim.Spec.Id)
				} else {
					v.updateVMClaimStaticBindAttempts(vmClaim.Status.StaticBindAttempts+1, vmClaim.Spec.Id)
				}
				//v.vmClaimWorkqueue.AddRateLimited(obj)
				v.vmClaimWorkqueue.Add(obj)
				return nil
			}

			for name, vmStruct := range vmClaim.Spec.VirtualMachines {
				if vmStruct.VirtualMachineId == "" {
					vmId, err := v.assignNextFreeVM(vmClaim.Spec.Id, vmClaim.Spec.UserId, vmStruct.Template, chosenEnvironmentId, vmClaim.Spec.RestrictedBind, vmClaim.Spec.RestrictedBindValue)
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

			//v.vmClaimWorkqueue.Forget(obj)
			glog.V(4).Infof("vmclaim processed and assigned by controller %v", objName)

			return nil
		}

	}()

	if err != nil {
		return true
	}

	return true
}

func (v *VMClaimController) assignNextFreeVM(vmClaimId string, user string, template string, environmentId string, restrictedBind bool, restrictedBindValue string) (string, error) {

	vmLabels := labels.Set{
		"bound":       "false",
		"environment": environmentId,
		"ready":       "true",
		"template":    template,
	}

	if restrictedBind {
		vmLabels["restrictedbind"] = "true"
		vmLabels["restrictedbindvalue"] = restrictedBindValue
	} else {
		vmLabels["restrictedbind"] = "false"
	}

	vms, err := v.vmLister.List(vmLabels.AsSelector())

	if err != nil {
		return "", fmt.Errorf("error while listing all vms %v", err)
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
			break
		}
	}

	if assigned {
		return vmId, nil
	}

	return vmId, fmt.Errorf("unknown error while assigning next free vm")

}

func (v *VMClaimController) updateVMClaimWithVM(vmName string, vmId string, vmClaimId string) error {

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

func (v *VMClaimController) updateVMClaimStatus(bound bool, ready bool, vmClaimId string) error {

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

func (v *VMClaimController) updateVMClaimBindMode(bindMode string, dynamicBindRequestId string, vmClaimId string) error {

	retryErr := retry.RetryOnConflict(retry.DefaultRetry, func() error {
		result, getErr := v.hfClientSet.HobbyfarmV1().VirtualMachineClaims().Get(vmClaimId, metav1.GetOptions{})
		if getErr != nil {
			return fmt.Errorf("Error retrieving latest version of Virtual Machine Claim %s: %v", vmClaimId, getErr)
		}

		if bindMode == "static" {
			result.Status.StaticBindAttempts = 0
		}
		result.Status.BindMode = bindMode
		result.Status.DynamicBindRequestId = dynamicBindRequestId

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

func (v *VMClaimController) updateVMClaimStaticBindAttempts(staticBindAttempts int, vmClaimId string) error {

	retryErr := retry.RetryOnConflict(retry.DefaultRetry, func() error {
		result, getErr := v.hfClientSet.HobbyfarmV1().VirtualMachineClaims().Get(vmClaimId, metav1.GetOptions{})
		if getErr != nil {
			return fmt.Errorf("Error retrieving latest version of Virtual Machine Claim %s: %v", vmClaimId, getErr)
		}

		result.Status.StaticBindAttempts = staticBindAttempts

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
