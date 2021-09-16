package vmclaimcontroller

import (
	"context"
	"fmt"
	"github.com/hobbyfarm/gargantua/pkg/controllers/scheduledevent"
	"github.com/hobbyfarm/gargantua/pkg/sessionserver"
	"math/rand"
	"time"

	"github.com/golang/glog"
	hfv1 "github.com/hobbyfarm/gargantua/pkg/apis/hobbyfarm.io/v1"
	hfClientset "github.com/hobbyfarm/gargantua/pkg/client/clientset/versioned"
	hfInformers "github.com/hobbyfarm/gargantua/pkg/client/informers/externalversions"
	hfListers "github.com/hobbyfarm/gargantua/pkg/client/listers/hobbyfarm.io/v1"
	"github.com/hobbyfarm/gargantua/pkg/util"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/retry"
	"k8s.io/client-go/util/workqueue"
)

const (
	StaticBindAttemptThreshold  int = 3
	DynamicBindAttemptThreshold int = 2
)

type VMClaimController struct {
	hfClientSet hfClientset.Interface

	vmLister      hfListers.VirtualMachineLister
	vmClaimLister hfListers.VirtualMachineClaimLister

	vmClaimWorkqueue workqueue.Interface

	vmWorkqueue workqueue.Interface

	vmClaimHasSynced cache.InformerSynced
	vmHasSynced      cache.InformerSynced
	ctx              context.Context
}

func NewVMClaimController(hfClientSet hfClientset.Interface, hfInformerFactory hfInformers.SharedInformerFactory, ctx context.Context) (*VMClaimController, error) {
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
			vmClaimController.enqueueVM(new)
		},
		DeleteFunc: vmClaimController.enqueueVM,
	}, time.Minute*30)

	vmClaimController.vmClaimHasSynced = hfInformerFactory.Hobbyfarm().V1().VirtualMachineClaims().Informer().HasSynced
	vmClaimController.vmHasSynced = hfInformerFactory.Hobbyfarm().V1().VirtualMachines().Informer().HasSynced
	vmClaimController.ctx = ctx

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
		_, objName, err := cache.SplitMetaNamespaceKey(obj.(string))
		vm, err := v.hfClientSet.HobbyfarmV1().VirtualMachines().Get(v.ctx, objName, metav1.GetOptions{})

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
		if vm.Spec.VirtualMachineClaimId != "" && vm.Status.Status == hfv1.VmStatusRunning {
			v.vmClaimWorkqueue.Add(vm.Spec.VirtualMachineClaimId)
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

func (v *VMClaimController) processNextVMClaim() bool {
	obj, shutdown := v.vmClaimWorkqueue.Get()

	glog.V(8).Infof("processing VM Claim")

	if shutdown {
		return false
	}

	err := func() error {
		_, objName, err := cache.SplitMetaNamespaceKey(obj.(string))
		if err != nil {
			glog.Errorf("error while splitting meta namespace key %v", err)
			return err
		}

		// fetch vmClaim
		vmClaim, err := v.hfClientSet.HobbyfarmV1().VirtualMachineClaims().Get(v.ctx, objName, metav1.GetOptions{})
		if err != nil {
			if errors.IsNotFound(err) {
				glog.Infof("vmClaim %s not found on queue.. ignoring", objName)
				return nil
			} else {
				glog.Errorf("error while retrieving vmclaim %s from queue with err %v", objName, err)
				return err
			}
		}

		// ignore vm objects which are being deleted
		if vmClaim.DeletionTimestamp.IsZero() {
			return v.processVMClaim(vmClaim)
		}
		return nil
	}()

	if err != nil {
		// requeue object
		//v.vmClaimWorkqueue.Add(obj)
		return true
	}

	v.vmClaimWorkqueue.Done(obj)
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
				result, getErr := v.hfClientSet.HobbyfarmV1().VirtualMachines().Get(v.ctx, vmId, metav1.GetOptions{})
				if getErr != nil {
					return fmt.Errorf("Error retrieving latest version of Virtual Machine %s: %v", vmId, getErr)
				}

				result.Status.Allocated = true
				result.Spec.VirtualMachineClaimId = vmClaimId
				result.Spec.UserId = user

				result.Labels["bound"] = "true"

				vm, updateErr := v.hfClientSet.HobbyfarmV1().VirtualMachines().Update(v.ctx, result, metav1.UpdateOptions{})
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

func (v *VMClaimController) updateVMClaimWithVM(vmDetails map[string]string, vmClaimId string) error {

	retryErr := retry.RetryOnConflict(retry.DefaultRetry, func() error {
		result, getErr := v.hfClientSet.HobbyfarmV1().VirtualMachineClaims().Get(v.ctx, vmClaimId, metav1.GetOptions{})
		if getErr != nil {
			return fmt.Errorf("Error retrieving latest version of Virtual Machine Claim %s: %v", vmClaimId, getErr)
		}

		for vmName, vmId := range vmDetails {
			vmClaimVM := result.Spec.VirtualMachines[vmName]
			vmClaimVM.VirtualMachineId = vmId

			result.Spec.VirtualMachines[vmName] = vmClaimVM
		}

		vmc, updateErr := v.hfClientSet.HobbyfarmV1().VirtualMachineClaims().Update(v.ctx, result, metav1.UpdateOptions{})
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

func (v *VMClaimController) updateVMClaimStatus(bound bool, ready bool, vmc *hfv1.VirtualMachineClaim) error {

	vmc.Status.Bound = bound
	vmc.Status.Ready = ready
	retryErr := retry.RetryOnConflict(retry.DefaultRetry, func() error {
		vmc, updateErr := v.hfClientSet.HobbyfarmV1().VirtualMachineClaims().Update(v.ctx, vmc, metav1.UpdateOptions{})
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
		return fmt.Errorf("Error updating Virtual Machine Claim: %s, %v", vmc.Name, retryErr)
	}
	return nil
}

func (v *VMClaimController) processVMClaim(vmc *hfv1.VirtualMachineClaim) (err error) {
	if vmc.Status.Tainted {
		glog.Infof("vmclaim %v is tainted.. ignoring", vmc.Name)
		return nil
	}

	if !vmc.Status.Bound && !vmc.Status.Ready {
		// submit VM requests //
		// update status
		err = v.submitVirtualMachines(vmc)
		if err != nil {
			return err
		}
		return v.updateVMClaimStatus(true, false, vmc)
	}

	if vmc.Status.Bound && !vmc.Status.Ready {
		// reconcile triggered by VM being ready
		// lets check the VM's
		ready, err := v.checkVMStatus(vmc)
		if err != nil {
			glog.Errorf("error checking vmStatus for vmc: %s %v", vmc.Name, err)
			return err
		}
		// update status
		glog.V(4).Infof("vm's have been requested for vmclaim: %s", vmc.Name)
		return v.updateVMClaimStatus(true, ready, vmc)
	}

	if vmc.Status.Bound && vmc.Status.Ready {
		// nothing else needs to be done.. ignore and move along
		glog.V(4).Infof("vmclaim %s is ready", vmc.Name)
	}

	return nil
}

func (v *VMClaimController) submitVirtualMachines(vmc *hfv1.VirtualMachineClaim) (err error) {
	accessCode, ok := vmc.Labels[sessionserver.AccessCodeLabel]
	if !ok {
		glog.Error("accessCode label not set on vmc, aborting")
		return fmt.Errorf("accessCode label not set on vmc, aborting")
	}

	env, seName, dbc, err := v.findEnvironmentForVM(accessCode)
	if err != nil {
		glog.Errorf("error fetching environment for access code %s  %v", accessCode, err)
		return err
	}
	vmMap := make(map[string]hfv1.VirtualMachineClaimVM)
	for vmName, vmDetails := range vmc.Spec.VirtualMachines {
		genName := fmt.Sprintf("%s-%08x", vmc.Name, rand.Uint32())
		vm := &hfv1.VirtualMachine{
			ObjectMeta: metav1.ObjectMeta{
				Name: genName,
				OwnerReferences: []metav1.OwnerReference{
					{
						APIVersion: "hobbyfarm.io/v1",
						Kind:       "VirtualMachineClaim",
						Name:       vmc.Name,
						UID:        vmc.UID,
					},
				},
				Labels: map[string]string{
					"dynamic":                          "true",
					"vmc":                              vmc.Name,
					"template":                         vmDetails.Template,
					"environment":                      env.Name,
					"bound":                            "true",
					"ready":                            "false",
					scheduledevent.ScheduledEventLabel: seName,
				},
			},
			Spec: hfv1.VirtualMachineSpec{
				Id:                       genName,
				VirtualMachineTemplateId: vmDetails.Template,
				KeyPair:                  "",
				VirtualMachineClaimId:    vmc.Name,
				UserId:                   vmc.Spec.UserId,
				Provision:                true,
				VirtualMachineSetId:      "",
			},
			Status: hfv1.VirtualMachineStatus{
				Status:        hfv1.VmStatusRFP,
				Allocated:     true,
				Tainted:       false,
				WsEndpoint:    env.Spec.WsEndpoint,
				EnvironmentId: env.Name,
				PublicIP:      "",
				PrivateIP:     "",
			},
		}
		// used to later repopulate the info back //
		vmMap[vmName] = hfv1.VirtualMachineClaimVM{
			Template:         vmDetails.Template,
			VirtualMachineId: genName,
		}
		sshUser, exists := env.Spec.TemplateMapping[vmDetails.Template]["ssh_username"]
		if exists {
			vm.Spec.SshUsername = sshUser
		}

		// extra label to indicate external provisioning so tfpcontroller ignores this request //
		if provisionMethod, ok := env.Annotations["hobbyfarm.io/provisioner"]; ok {
			vm.Labels["hobbyfarm.io/provisioner"] = provisionMethod
			vm.Spec.Provision = false
		}

		if dbc.Spec.RestrictedBind {
			vm.ObjectMeta.Labels["restrictedbind"] = "true"
			vm.ObjectMeta.Labels["restrictedbindvalue"] = dbc.Spec.RestrictedBindValue
		} else {
			vm.ObjectMeta.Labels["restrictedbind"] = "false"
		}

		vm.Labels["hobbyfarm.io/vmtemplate"] = vm.Spec.VirtualMachineTemplateId

		_, err = v.hfClientSet.HobbyfarmV1().VirtualMachines().Create(v.ctx, vm, metav1.CreateOptions{})
	}

	vmc.Spec.VirtualMachines = vmMap
	return nil
}

func (v *VMClaimController) findEnvironmentForVM(accessCode string) (env *hfv1.Environment, seName string, dbc *hfv1.DynamicBindConfiguration, err error) {
	// find scheduledEvent for this accessCode
	var envName string
	seList, err := v.hfClientSet.HobbyfarmV1().ScheduledEvents().List(v.ctx, metav1.ListOptions{})
	if err != nil {
		return env, seName, dbc, err
	}

	for _, se := range seList.Items {
		if se.Spec.AccessCode == accessCode {
			seName = se.Name
			for k, _ := range se.Spec.RequiredVirtualMachines {
				envName = k
			}
		}
	}

	env, err = v.hfClientSet.HobbyfarmV1().Environments().Get(v.ctx, envName, metav1.GetOptions{})

	dbcList, err := v.hfClientSet.HobbyfarmV1().DynamicBindConfigurations().List(v.ctx, metav1.ListOptions{
		LabelSelector: fmt.Sprintf("restrictedbindvalue=%s", seName),
	})

	if err != nil {
		glog.Errorf("error listing dbc %v", err)
		return env, seName, dbc, err
	}

	if len(dbcList.Items) != 1 {
		return env, seName, dbc, fmt.Errorf("incorrect number of dbc matching sessionName found")
	}

	dbc = &hfv1.DynamicBindConfiguration{}
	dbc = &dbcList.Items[0]
	return env, seName, dbc, err
}

func (v *VMClaimController) checkVMStatus(vmc *hfv1.VirtualMachineClaim) (ready bool, err error) {
	ready = true
	for _, vmTemplate := range vmc.Spec.VirtualMachines {
		vm, err := v.hfClientSet.HobbyfarmV1().VirtualMachines().Get(v.ctx, vmTemplate.VirtualMachineId, metav1.GetOptions{})
		if err != nil {
			return ready, err
		}
		if vm.Status.Status == hfv1.VmStatusRunning {
			ready = ready && true
		} else {
			ready = ready && false
		}
	}

	return ready, err
}
