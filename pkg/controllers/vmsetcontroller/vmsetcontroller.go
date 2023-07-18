package vmsetcontroller

import (
	"context"
	"fmt"
	"math/rand"
	"strings"
	"time"

	"github.com/golang/glog"
	hfv1 "github.com/hobbyfarm/gargantua/pkg/apis/hobbyfarm.io/v1"
	hfClientset "github.com/hobbyfarm/gargantua/pkg/client/clientset/versioned"
	hfInformers "github.com/hobbyfarm/gargantua/pkg/client/informers/externalversions"
	hfListers "github.com/hobbyfarm/gargantua/pkg/client/listers/hobbyfarm.io/v1"
	"github.com/hobbyfarm/gargantua/pkg/util"
	"k8s.io/apimachinery/pkg/api/errors"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/retry"
	"k8s.io/client-go/util/workqueue"
)

type VirtualMachineSetController struct {
	hfClientSet hfClientset.Interface
	//vmSetWorkqueue workqueue.RateLimitingInterface
	vmWorkqueue      workqueue.RateLimitingInterface
	vmSetWorkqueue   workqueue.Interface
	vmSetLister      hfListers.VirtualMachineSetLister
	vmLister         hfListers.VirtualMachineLister
	envLister        hfListers.EnvironmentLister
	vmTemplateLister hfListers.VirtualMachineTemplateLister

	vmSetSynced      cache.InformerSynced
	vmSynced         cache.InformerSynced
	envSynced        cache.InformerSynced
	vmTemplateSynced cache.InformerSynced
	ctx              context.Context
}

const (
	vmEnvironmentIndex = "vm.vmclaim.controllers.hobbyfarm.io/environment-index"
	vmSetFinalizer     = "finalizer.hobbyfarm.io/vmset"
)

func NewVirtualMachineSetController(hfClientSet hfClientset.Interface, hfInformerFactory hfInformers.SharedInformerFactory, ctx context.Context) (*VirtualMachineSetController, error) {
	vmSetController := VirtualMachineSetController{}
	vmSetController.hfClientSet = hfClientSet

	vmSetController.vmSetSynced = hfInformerFactory.Hobbyfarm().V1().VirtualMachineSets().Informer().HasSynced
	vmSetController.vmSynced = hfInformerFactory.Hobbyfarm().V1().VirtualMachines().Informer().HasSynced
	vmSetController.envSynced = hfInformerFactory.Hobbyfarm().V1().Environments().Informer().HasSynced
	vmSetController.vmTemplateSynced = hfInformerFactory.Hobbyfarm().V1().VirtualMachineTemplates().Informer().HasSynced

	//vmSetController.vmSetWorkqueue = workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "VMSet")
	vmSetController.vmWorkqueue = workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "VM")
	vmSetController.vmSetWorkqueue = workqueue.NewNamed("vmsc-vms")

	//vmClaimController.hfInformerFactory = hfInformerFactory

	vmSetController.vmSetLister = hfInformerFactory.Hobbyfarm().V1().VirtualMachineSets().Lister()
	vmSetController.vmLister = hfInformerFactory.Hobbyfarm().V1().VirtualMachines().Lister()
	vmSetController.envLister = hfInformerFactory.Hobbyfarm().V1().Environments().Lister()
	vmSetController.vmTemplateLister = hfInformerFactory.Hobbyfarm().V1().VirtualMachineTemplates().Lister()

	vmSetInformer := hfInformerFactory.Hobbyfarm().V1().VirtualMachineSets().Informer()

	vmSetInformer.AddEventHandlerWithResyncPeriod(cache.ResourceEventHandlerFuncs{
		AddFunc: vmSetController.enqueueVMSet,
		UpdateFunc: func(old, new interface{}) {
			vmSetController.enqueueVMSet(new)
		},
		DeleteFunc: vmSetController.enqueueVMSet,
	}, time.Minute*30)

	vmInformer := hfInformerFactory.Hobbyfarm().V1().VirtualMachines().Informer()

	vmInformer.AddEventHandlerWithResyncPeriod(cache.ResourceEventHandlerFuncs{
		AddFunc: vmSetController.handleVM,
		UpdateFunc: func(old, new interface{}) {
			vmSetController.handleVM(new)
		},
		DeleteFunc: vmSetController.handleVM,
	}, time.Minute*30)
	vmSetController.ctx = ctx

	return &vmSetController, nil
}

func (v *VirtualMachineSetController) handleVM(obj interface{}) {
	var key string
	var err error
	if key, err = cache.MetaNamespaceKeyFunc(obj); err != nil {
		return
	}
	glog.V(8).Infof("enqueueing vm %v to check associated vmsets in vmsetcontroller", key)
	v.vmWorkqueue.Add(key)
}

func (v *VirtualMachineSetController) enqueueVMSet(obj interface{}) {
	var key string
	var err error
	if key, err = cache.MetaNamespaceKeyFunc(obj); err != nil {
		//utilruntime.HandleError(err)
		return
	}
	glog.V(8).Infof("Enqueueing vm set %s", key)
	//v.vmSetWorkqueue.AddRateLimited(key)
	v.vmSetWorkqueue.Add(key)
}

func (v *VirtualMachineSetController) Run(stopCh <-chan struct{}) error {
	defer v.vmSetWorkqueue.ShutDown()

	glog.V(4).Infof("Starting vm set")
	glog.Info("Waiting for informer caches to sync")
	if ok := cache.WaitForCacheSync(stopCh, v.vmSynced, v.vmSetSynced, v.vmTemplateSynced, v.envSynced); !ok {
		return fmt.Errorf("failed to wait for caches to sync")
	}
	glog.Info("Starting vm set controller worker")
	go wait.Until(v.runVMSetWorker, time.Second, stopCh)
	go wait.Until(v.runVMWorker, time.Second, stopCh)
	glog.Info("Started vm set controller worker")
	//if ok := cache.WaitForCacheSync(stopCh, )
	<-stopCh
	return nil
}

func (v *VirtualMachineSetController) runVMSetWorker() {
	for v.processNextVMSet() {

	}
}

func (v *VirtualMachineSetController) runVMWorker() {
	for v.processNextVM() {

	}
}

func (v *VirtualMachineSetController) processNextVM() bool {
	obj, shutdown := v.vmWorkqueue.Get()
	glog.V(8).Infof("processing VM in vmsetcontroller controller for update")

	if shutdown {
		return false
	}

	err := func() error {
		_, objName, err := cache.SplitMetaNamespaceKey(obj.(string))
		if err != nil {
			return err
		}
		vm, err := v.hfClientSet.HobbyfarmV1().VirtualMachines(util.GetReleaseNamespace()).Get(v.ctx, objName, metav1.GetOptions{})

		if err != nil {

			// ideally should put logic here to determine if we need to retry and push this vm back onto the workqueue
			if errors.IsNotFound(err) {
				return nil

			} else {
				glog.Errorf("error while retrieving vm %s: %v, will be requeued", objName, err)
				return err
			}
		}

		// trigger reconcile on vmClaims only when associated VM is running
		// this should avoid triggering unwanted reconciles of VMClaims until the VM's are running
		if !vm.DeletionTimestamp.IsZero() {
			glog.V(4).Infof("requeuing vmset %s to account for tainted vm %s", vm.Spec.VirtualMachineSetId, vm.Name)
			err = v.removeVMFinalizer(vm)
			if err != nil {
				glog.Errorf("error removing vm finalizer on vm %s", vm.Name)
				return err
			}
			defer v.vmSetWorkqueue.Add(vm.Spec.VirtualMachineSetId)
		}

		return nil
	}()

	if err != nil {
		// return and requeue the object
		//v.vmWorkqueue.Add(obj)
		return true
	}
	//vm event has been processed successfully ignore it
	v.vmWorkqueue.Done(obj)
	return true
}
func (v *VirtualMachineSetController) processNextVMSet() bool {
	obj, shutdown := v.vmSetWorkqueue.Get()

	glog.V(8).Infof("processing VMSet")

	if shutdown {
		return false
	}
	err := func() error {
		defer v.vmSetWorkqueue.Done(obj)
		glog.V(4).Infof("processing vms in vmset controller: %v", obj)
		_, objName, err := cache.SplitMetaNamespaceKey(obj.(string)) // this is actually not necessary because VM's are not namespaced yet...
		if err != nil {
			glog.Errorf("error while splitting meta namespace key %v", err)
			//e.vmWorkqueue.AddRateLimited(obj)
			return nil
		}

		vmSet, err := v.vmSetLister.VirtualMachineSets(util.GetReleaseNamespace()).Get(objName)
		if err != nil {
			glog.Errorf("error while retrieving virtual machine set %s, likely deleted %v", objName, err)
			//v.vmSetWorkqueue.Forget(obj)
			return nil
		}

		err = v.reconcileVirtualMachineSet(vmSet)
		if err != nil {
			glog.Error(err)
		}
		//v.vmSetWorkqueue.Forget(obj)
		glog.V(4).Infof("vm set processed by vmset controller %v", objName)
		return nil

	}()

	if err != nil {
		return true
	}

	// successfully reconcilled, mark object as done
	v.vmSetWorkqueue.Done(obj)
	return true
}

func (v *VirtualMachineSetController) reconcileVirtualMachineSet(vmset *hfv1.VirtualMachineSet) error {

	currentVMs, err := v.vmLister.List(labels.Set{
		"vmset": vmset.Name,
	}.AsSelector())

	if err != nil {
		glog.Errorf("error listing vms in vmset controller")
		return err
	}

	if len(currentVMs) < vmset.Spec.Count { // if desired count is greater than the current provisioned
		// 1. let's check the environment to see if there is available capacity
		// 2. if available capacity is available let's create new VM's
		glog.V(4).Infof("vmset %s needs %d vm's but current vm count is %d", vmset.Name, vmset.Spec.Count, len(currentVMs))
		env, err := v.envLister.Environments(util.GetReleaseNamespace()).Get(vmset.Spec.Environment)
		var provision bool
		provision = true
		if provisionMethod, ok := env.Annotations["hobbyfarm.io/provisioner"]; ok && provisionMethod != "" {
			provision = false
		}
		if err != nil {
			if apierrors.IsNotFound(err) {
				glog.Errorf("environment invalid")
			}
			return err
		}

		vmt, err := v.vmTemplateLister.VirtualMachineTemplates(util.GetReleaseNamespace()).Get(vmset.Spec.VMTemplate)
		if err != nil {
			return fmt.Errorf("error while retrieving virtual machine template %s %v", vmset.Spec.VMTemplate, err)
		}
		needed := vmset.Spec.Count - len(currentVMs)

		glog.V(5).Infof("provisioning %d vms", needed)
		// this code is so... verbose...
		for i := 0; i < needed; i++ {
			vmName := strings.Join([]string{vmset.Spec.BaseName, fmt.Sprintf("%08x", rand.Uint32())}, "-")
			vm := &hfv1.VirtualMachine{
				ObjectMeta: metav1.ObjectMeta{
					Name: vmName,
					OwnerReferences: []metav1.OwnerReference{
						{
							APIVersion: "hobbyfarm.io/v1",
							Kind:       "VirtualMachineSet",
							Name:       vmset.Name,
							UID:        vmset.UID,
						},
					},
					Labels: map[string]string{
						"dynamic":                   "false",
						"vmset":                     vmset.Name,
						util.VirtualMachineTemplate: vmt.Name,
						util.EnvironmentLabel:       env.Name,
						"bound":                     "false",
						"ready":                     "false",
						util.ScheduledEventLabel:    vmset.ObjectMeta.Labels[util.ScheduledEventLabel],
					},
				},
				Spec: hfv1.VirtualMachineSpec{
					VirtualMachineTemplateId: vmt.Name,
					SecretName:               "",
					Protocol:                 "ssh",
					VirtualMachineClaimId:    "",
					UserId:                   "",
					Provision:                provision,
					VirtualMachineSetId:      vmset.Name,
				},
			}

			config := util.GetVMConfig(env, vmt)

			sshUser, exists := config["ssh_username"]
			if exists {
				vm.Spec.SshUsername = sshUser
			}
			protocol, exists := config["protocol"]
			if exists {
				vm.Spec.Protocol = protocol
			}
			if vmset.Spec.RestrictedBind {
				vm.ObjectMeta.Labels["restrictedbind"] = "true"
				vm.ObjectMeta.Labels["restrictedbindvalue"] = vmset.Spec.RestrictedBindValue
			} else {
				vm.ObjectMeta.Labels["restrictedbind"] = "false"
			}
			// adding a custom finalizer for reconcile of vmsets
			vm.SetFinalizers([]string{vmSetFinalizer})
			vm, err := v.hfClientSet.HobbyfarmV1().VirtualMachines(util.GetReleaseNamespace()).Create(v.ctx, vm, metav1.CreateOptions{})

			if err != nil {
				glog.Error(err)
			}

			vm.Status = hfv1.VirtualMachineStatus{
				Status:        hfv1.VmStatusRFP,
				Allocated:     false,
				Tainted:       false,
				WsEndpoint:    env.Spec.WsEndpoint,
				PublicIP:      "",
				PrivateIP:     "",
				EnvironmentId: env.Name,
				Hostname:      "",
			}

			_, err = v.hfClientSet.HobbyfarmV1().VirtualMachines(util.GetReleaseNamespace()).UpdateStatus(v.ctx, vm, metav1.UpdateOptions{})

			if err != nil {
				glog.Error(err)
			}

			err = util.VerifyVM(v.vmLister, vm)
			if err != nil {
				glog.Error(err)
			}
		}
	}
	//-----------------------handle case of scaling down VMSets
	if len(currentVMs) > vmset.Spec.Count {
		needed_delete := len(currentVMs) - vmset.Spec.Count
		for _, cur_vm := range currentVMs {
			if !cur_vm.Status.Allocated {
				v.hfClientSet.HobbyfarmV1().VirtualMachines(util.GetReleaseNamespace()).Delete(v.ctx, cur_vm.Name, metav1.DeleteOptions{})
				needed_delete--
				if needed_delete == 0 {
					break
				}
			}
		}
	}
	//-----------------------------------------------------
	vms, err := v.vmLister.List(labels.Set{
		"vmset": string(vmset.Name),
	}.AsSelector())

	if err != nil {
		glog.Errorf("error while retrieving vms owned by vmset %s", vmset.Name)
	}

	provisionedCount := 0
	activeCount := 0
	for _, x := range vms {
		if x.DeletionTimestamp == nil && !x.Status.Tainted {
			activeCount++
		}
		provisionedCount++
	}

	if activeCount < vmset.Spec.Count {
		glog.V(4).Infof("requeing VMset as there are not enough VMs ready")
		v.enqueueVMSet(vmset)
	}

	err = v.updateVMSetCount(vmset.Name, activeCount, provisionedCount)

	return err
}

func (v *VirtualMachineSetController) updateVMSetCount(vmSetName string, active int, prov int) error {
	retryErr := retry.RetryOnConflict(retry.DefaultRetry, func() error {
		result, getErr := v.hfClientSet.HobbyfarmV1().VirtualMachineSets(util.GetReleaseNamespace()).Get(v.ctx, vmSetName, metav1.GetOptions{})
		if getErr != nil {
			return fmt.Errorf("error retrieving latest version of Virtual Machine Set %s: %v", vmSetName, getErr)
		}

		result.Status.ProvisionedCount = prov
		result.Status.AvailableCount = active

		vms, updateErr := v.hfClientSet.HobbyfarmV1().VirtualMachineSets(util.GetReleaseNamespace()).UpdateStatus(v.ctx, result, metav1.UpdateOptions{})
		if updateErr != nil {
			glog.Error(updateErr)
			return updateErr
		}

		err := util.VerifyVMSet(v.vmSetLister, vms)
		if err != nil {
			glog.Error(err)
		}
		glog.V(4).Infof("updated result for virtual machine set")
		return nil
	})
	if retryErr != nil {
		return fmt.Errorf("error updating Virtual Machine Set: %s, %v", vmSetName, retryErr)
	}

	return nil
}

func (v *VirtualMachineSetController) removeVMFinalizer(vm *hfv1.VirtualMachine) (err error) {
	if ContainsFinalizer(vm, vmSetFinalizer) {
		RemoveFinalizer(vm, vmSetFinalizer)
		_, err = v.hfClientSet.HobbyfarmV1().VirtualMachines(util.GetReleaseNamespace()).Update(v.ctx, vm, metav1.UpdateOptions{})
	}
	return err
}

// From ControllerUtil to save dep issues

// RemoveFinalizer accepts an Object and removes the provided finalizer if present.
func RemoveFinalizer(vm *hfv1.VirtualMachine, finalizer string) {
	f := vm.GetFinalizers()
	for i := 0; i < len(f); i++ {
		if f[i] == finalizer {
			f = append(f[:i], f[i+1:]...)
			i--
		}
	}
	vm.SetFinalizers(f)
}

// ContainsFinalizer checks an Object that the provided finalizer is present.
func ContainsFinalizer(vm *hfv1.VirtualMachine, finalizer string) bool {
	f := vm.GetFinalizers()
	for _, e := range f {
		if e == finalizer {
			return true
		}
	}
	return false
}
