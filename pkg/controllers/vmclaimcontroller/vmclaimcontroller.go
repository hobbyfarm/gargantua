package vmclaimcontroller

import (
	"context"
	"fmt"
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
	vmtLister 	  hfListers.VirtualMachineTemplateLister

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
	vmClaimController.vmtLister = hfInformerFactory.Hobbyfarm().V1().VirtualMachineTemplates().Lister()

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
		vmClaim, err := v.hfClientSet.HobbyfarmV1().VirtualMachineClaims(util.GetReleaseNamespace()).Get(v.ctx, objName, metav1.GetOptions{})
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

func (v *VMClaimController) updateVMClaimStatus(bound bool, ready bool, vmc *hfv1.VirtualMachineClaim) error {
	retryErr := retry.RetryOnConflict(retry.DefaultRetry, func() error {
		newestVmc, err := v.hfClientSet.HobbyfarmV1().VirtualMachineClaims(util.GetReleaseNamespace()).Get(v.ctx, vmc.Name, metav1.GetOptions{})
		newestVmc.Status.Bound = bound
		newestVmc.Status.Ready = ready
		newestVmc, err = v.hfClientSet.HobbyfarmV1().VirtualMachineClaims(util.GetReleaseNamespace()).UpdateStatus(v.ctx, newestVmc, metav1.UpdateOptions{})
		if err != nil {
			return err
		}
		glog.V(4).Infof("updated result for virtual machine claim")

		verifyErr := util.VerifyVMClaim(v.vmClaimLister, newestVmc)

		if verifyErr != nil {
			return verifyErr
		}
		return nil
	})
	if retryErr != nil {
		return fmt.Errorf("error updating Virtual Machine Claim: %s, %v", vmc.Name, retryErr)
	}
	return nil
}

func (v *VMClaimController) processVMClaim(vmc *hfv1.VirtualMachineClaim) (err error) {
	if vmc.Status.Tainted {
		glog.Infof("vmclaim %v is tainted.. cleaning it up", vmc.Name)
		return v.hfClientSet.HobbyfarmV1().VirtualMachineClaims(util.GetReleaseNamespace()).Delete(v.ctx, vmc.Name, metav1.DeleteOptions{})
	}

	if !vmc.Status.Bound && !vmc.Status.Ready {
		// submit VM requests //
		// update status
		if vmc.Status.BindMode == "dynamic" {
			err = v.submitVirtualMachines(vmc)
			if err != nil {
				// VirtualMachines could not be submitted. Delete Claim
				glog.Errorf("error processing vmc %s - %s", vmc.Name, err.Error())
				return v.hfClientSet.HobbyfarmV1().VirtualMachineClaims(util.GetReleaseNamespace()).Delete(v.ctx, vmc.Name, metav1.DeleteOptions{})
			}
		} else if vmc.Status.BindMode == "static" {
			err = v.findVirtualMachines(vmc)
			if err != nil {
				// VirtualMachines could not be bound. Delete Claim
				// TODO 17.01.2023 delete session?
				glog.Errorf("error processing vmc %s - %s", vmc.Name, err.Error())
				return v.hfClientSet.HobbyfarmV1().VirtualMachineClaims(util.GetReleaseNamespace()).Delete(v.ctx, vmc.Name, metav1.DeleteOptions{})
			}
		} else {
			glog.Errorf("vmc bind mode needs to be either dynamic or static.. ignoring this object %s", vmc.Name)
			return nil
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

type VMEnvironment struct {
 	Environment hfv1.Environment
	DynamicBindConfiguration hfv1.DynamicBindConfiguration
}

func (v *VMClaimController) submitVirtualMachines(vmc *hfv1.VirtualMachineClaim) (err error) {
	accessCode, ok := vmc.Labels[util.AccessCodeLabel]
	if !ok {
		glog.Error("accessCode label not set on vmc, aborting")
		return fmt.Errorf("accessCode label not set on vmc, aborting")
	}

	environments, seName, dbcList, err := v.findEnvironmentsForVM(accessCode, vmc)
	if err != nil {
		glog.Errorf("error fetching environment for access code %s  %v", accessCode, err)
		return err
	}

	// Calculate required VMs per template
	requiredTemplateCount := make(map[string]int)
	for _, vmDetails := range vmc.Spec.VirtualMachines {
		if count, found := requiredTemplateCount[vmDetails.Template]; found {
			requiredTemplateCount[vmDetails.Template] = count + 1
		}else{
			requiredTemplateCount[vmDetails.Template] = 1
		}
	}

	environmentMap := make(map[string]VMEnvironment) // Maps node to the environment it should use
	bestDBC, err := v.findBestDBCForVMs(dbcList, requiredTemplateCount) // Try to find if one environment can provision all VMs


	reservedCapacity := make(map[string]map[string]int) // EnvironmentID -> TemplateID -> Count
	// Initialize reservedCapacity with 0 for all environments + associated templates
	for _, environment := range environments {
		reserved := make(map[string]int)
		for template, _ := range environment.Spec.TemplateMapping {
			reserved[template] = 0
		}
		reservedCapacity[environment.Name] = reserved
	}

	if(err != nil) {
		// We can not provision all VirtualMachines in one environment. Figure out which environments we want to use
		for vmName, vmDetails := range vmc.Spec.VirtualMachines {
			env, dbc, err := v.findSuitableEnvironmentForVMTemplate(environments, dbcList, vmDetails.Template, reservedCapacity)
			if err != nil{
				glog.Errorf("no suitable environment for %s (%s): %v", vmName, vmDetails.Template, err)
				return err
			}
			environmentMap[vmName] = VMEnvironment{env, dbc}
			reservedCapacity[env.Name][vmDetails.Template] += 1
		}
	}else{
		// One DBC for them all
		enviroment := hfv1.Environment{}
		for _, e := range environments{
			if e.Name == bestDBC.Spec.Environment {
				enviroment = e
				break
			}
		}
		for vmName, _ := range vmc.Spec.VirtualMachines {
			environmentMap[vmName] = VMEnvironment{enviroment, bestDBC}
		}
	}

	vmMap := make(map[string]hfv1.VirtualMachineClaimVM)
	for vmName, vmDetails := range vmc.Spec.VirtualMachines {
		genName := fmt.Sprintf("%s-%08x", vmc.Spec.BaseName, rand.Uint32())
		environment := environmentMap[vmName].Environment
		dbc := environmentMap[vmName].DynamicBindConfiguration
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
					"environment":                      environment.Name,
					"bound":                            "true",
					"ready":                            "false",
					util.ScheduledEventLabel: seName,
				},
			},
			Spec: hfv1.VirtualMachineSpec{
				VirtualMachineTemplateId: vmDetails.Template,
				SecretName:               "",
				Protocol: 				  "ssh",  //default protocol is ssh
				VirtualMachineClaimId:    vmc.Name,
				UserId:                   vmc.Spec.UserId,
				Provision:                true,
				VirtualMachineSetId:      "",
			},
		}
		// used to later repopulate the info back //
		vmMap[vmName] = hfv1.VirtualMachineClaimVM{
			Template:         vmDetails.Template,
			VirtualMachineId: genName,
		}

		vmt, err := v.vmtLister.VirtualMachineTemplates(util.GetReleaseNamespace()).Get(vmDetails.Template)
		if err != nil {
			glog.Errorf("error getting vmt %v", err)
			return err
		}

		config := util.GetVMConfig(&environment,vmt)
     
    protocol, exists := config["protocol"]
    if exists {
		  vm.Spec.Protocol = protocol
		}
		
    sshUser, exists := config["ssh_username"]
		if exists {
			vm.Spec.SshUsername = sshUser
		}

		// extra label to indicate external provisioning so tfpcontroller ignores this request //
		if provisionMethod, ok := environment.Annotations["hobbyfarm.io/provisioner"]; ok {
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

		createdVM, err := v.hfClientSet.HobbyfarmV1().VirtualMachines(util.GetReleaseNamespace()).Create(v.ctx, vm, metav1.CreateOptions{})
		if err != nil {
			return err
		}

		createdVM.Status = hfv1.VirtualMachineStatus{
			Status:        hfv1.VmStatusRFP,
			Allocated:     true,
			Tainted:       false,
			WsEndpoint:    environment.Spec.WsEndpoint,
			EnvironmentId: environment.Name,
			PublicIP:      "",
			PrivateIP:     "",
		}

		_, err = v.hfClientSet.HobbyfarmV1().VirtualMachines(util.GetReleaseNamespace()).UpdateStatus(v.ctx, createdVM, metav1.UpdateOptions{})
		if err != nil {
			return err
		}
	}

	vmc.Spec.VirtualMachines = vmMap

	_, err = v.hfClientSet.HobbyfarmV1().VirtualMachineClaims(util.GetReleaseNamespace()).Update(v.ctx, vmc, metav1.UpdateOptions{})
	if err != nil {
		return err
	}

	return nil
}

// Based on the given VirtualMachineClaim and ScheduledEvent find all suitable Environments (e.g. environment provides required VMTeplate & ScheduledEvents allows this environment and VMTemplate configuration etc.)
func (v *VMClaimController) findEnvironmentsForVM(accessCode string, vmc *hfv1.VirtualMachineClaim) (environments []hfv1.Environment, seName string, dbc []hfv1.DynamicBindConfiguration, err error) {
	seName, _, err = v.findScheduledEvent(accessCode)
	if err != nil {
		return environments, seName, dbc, err
	}

	dbcList, err := v.hfClientSet.HobbyfarmV1().DynamicBindConfigurations(util.GetReleaseNamespace()).List(v.ctx, metav1.ListOptions{
		LabelSelector: fmt.Sprintf("%s=%s", util.ScheduledEventLabel, seName),
	})

	if err != nil {
		glog.Errorf("error listing dbc %v", err)
		return environments, seName, dbc, err
	}

	for _ , dbc := range dbcList.Items {
		env, err := v.hfClientSet.HobbyfarmV1().Environments(util.GetReleaseNamespace()).Get(v.ctx, dbc.Spec.Environment, metav1.GetOptions{})

		if err != nil {
			glog.Errorf("error fetching environment %v", err)
			return environments, seName, dbcList.Items, err
		}
		environments = append(environments, *env)
	}

	if len(dbcList.Items) < 1 {
		return environments, seName, dbc, fmt.Errorf("incorrect number of dbc matching sessionName found")
	}

	return environments, seName, dbcList.Items, err
}

// Can one DBC provide all VMs when considering the limits? Return the DBC if there exists one
func (v *VMClaimController) findBestDBCForVMs(dbcList []hfv1.DynamicBindConfiguration, requiredTemplateCount map[string]int) (hfv1.DynamicBindConfiguration, error) {
	// Try to find best possible environment / DBC = All required VMs can be provisioned here
	for _, dbc := range dbcList {
		satisfiedDBC := true
		for requiredTemplate, requiredCount := range requiredTemplateCount {
			if dbcCapacity, found := dbc.Spec.BurstCountCapacity[requiredTemplate]; found {
				count, err := util.CountMachinesPerTemplateAndEnvironment(v.vmLister, requiredTemplate, dbc.Spec.Environment)
				if(err != nil){
					satisfiedDBC = false
					break
				}
				if requiredCount >= (dbcCapacity - count) {
					satisfiedDBC = false
					break
				}
			}else{
				satisfiedDBC = false
				break
			}
		}

		if satisfiedDBC {
			// This DBC works for all templates and has the required Counts available!
			glog.V(4).Infof("found best environment suitable for all VMs: %s", dbc.Spec.Environment)
			return dbc, nil
		}
	}
	return hfv1.DynamicBindConfiguration{}, fmt.Errorf("there is no best environment")
}

func (v *VMClaimController) findSuitableEnvironmentForVMTemplate(environments []hfv1.Environment, dbcList []hfv1.DynamicBindConfiguration, template string, reservedCapacity map[string]map[string]int) (hfv1.Environment, hfv1.DynamicBindConfiguration, error) {
	for _, environment := range environments {
		count, err := util.CountMachinesPerTemplateAndEnvironment(v.vmLister, template, environment.Name)
		if(err != nil){
			continue
		}

		// We have also reserved capacity for other VMs
		count += reservedCapacity[environment.Name][template]

		if(count >= environment.Spec.CountCapacity[template]){
			// Environment is at limit
			continue
		}
		// found environment that satisfies capacity for this template
		for _, dbc := range dbcList {
			if(dbc.Spec.Environment == environment.Name){
				if capacity, found := dbc.Spec.BurstCountCapacity[template]; found {
					if(count < capacity){
						// Capacity also satisfied for environment + scheduledEvent via DBC
						return environment, dbc,  nil
					}
				}
				break
			}
		}

	}

	return hfv1.Environment{}, hfv1.DynamicBindConfiguration{}, fmt.Errorf("no suitable environment found. capacity reached")
}

func (v *VMClaimController) checkVMStatus(vmc *hfv1.VirtualMachineClaim) (ready bool, err error) {
	ready = true
	for _, vmTemplate := range vmc.Spec.VirtualMachines {
		vm, err := v.hfClientSet.HobbyfarmV1().VirtualMachines(util.GetReleaseNamespace()).Get(v.ctx, vmTemplate.VirtualMachineId, metav1.GetOptions{})
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

func (v *VMClaimController) findScheduledEvent(accessCode string) (schedEvent string, environments map[string]map[string]int, err error) {
	seList, err := v.hfClientSet.HobbyfarmV1().ScheduledEvents(util.GetReleaseNamespace()).List(v.ctx, metav1.ListOptions{})
	if err != nil {
		return schedEvent, environments, err
	}

	for _, se := range seList.Items {
		if se.Spec.AccessCode == accessCode {
			schedEvent = se.Name
			environments = se.Spec.RequiredVirtualMachines
			break
		}
	}

	if schedEvent == "" {
		return schedEvent, environments, fmt.Errorf("no scheduled event matching access code %s found", accessCode)
	}

	return schedEvent, environments, nil
}

func (v *VMClaimController) findVirtualMachines(vmc *hfv1.VirtualMachineClaim) (err error) {
	accessCode, ok := vmc.Labels[util.AccessCodeLabel]
	if !ok {
		glog.Error("accessCode label not set on vmc, aborting")
		return fmt.Errorf("accessCode label not set on vmc, aborting")
	}
	_, environments, err := v.findScheduledEvent(accessCode)

	if err != nil {
		glog.Error("error finding scheduledevent during static bind")
		return err
	}

	vmMap := make(map[string]hfv1.VirtualMachineClaimVM)
	for name, vmStruct := range vmc.Spec.VirtualMachines {
		if vmStruct.VirtualMachineId == "" {
			glog.Info("assigning a vm")
			vmID, err := v.assignNextFreeVM(vmc.Name, vmc.Spec.UserId, environments, vmStruct.Template, vmc.Spec.RestrictedBind, vmc.Spec.RestrictedBindValue)
			if err != nil {
				return err
			}
			vmMap[name] = hfv1.VirtualMachineClaimVM{
				Template:         vmStruct.Template,
				VirtualMachineId: vmID,
			}
		}
	}
	vmc.Spec.VirtualMachines = vmMap

	_, err = v.hfClientSet.HobbyfarmV1().VirtualMachineClaims(util.GetReleaseNamespace()).Update(v.ctx, vmc, metav1.UpdateOptions{})
	if err != nil {
		return err
	}

	
	return nil
}

func  (v *VMClaimController) assignVM(vmClaimId string, user string, vmId string) (string, error) {
	retryErr := retry.RetryOnConflict(retry.DefaultRetry, func() error {
		result, getErr := v.hfClientSet.HobbyfarmV1().VirtualMachines(util.GetReleaseNamespace()).Get(v.ctx, vmId, metav1.GetOptions{})
		if getErr != nil {
			return fmt.Errorf("error retrieving latest version of Virtual Machine %s: %v", vmId, getErr)
		}


		result.Labels["bound"] = "true"
		result.Spec.VirtualMachineClaimId = vmClaimId
		result.Spec.UserId = user

		vm, updateErr := v.hfClientSet.HobbyfarmV1().VirtualMachines(util.GetReleaseNamespace()).Update(v.ctx, result, metav1.UpdateOptions{})
		if updateErr != nil {
			return updateErr
		}

		vm.Status.Allocated = true

		_, updateErr = v.hfClientSet.HobbyfarmV1().VirtualMachines(util.GetReleaseNamespace()).UpdateStatus(v.ctx, vm, metav1.UpdateOptions{})
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
		return "", fmt.Errorf("error updating Virtual Machine: %s, %v", vmId, retryErr)
	}

	return vmId, nil
}

func (v *VMClaimController) assignNextFreeVM(vmClaimId string, user string, environments map[string]map[string]int, template string, restrictedBind bool, restrictedBindValue string) (string, error) {
	vmLabels := labels.Set{
		"bound":       "false",
		"template":    template,
	}

	if restrictedBind {
		vmLabels["restrictedbind"] = "true"
		vmLabels["restrictedbindvalue"] = restrictedBindValue
	} else {
		vmLabels["restrictedbind"] = "false"
	}

	vms, err := v.vmLister.List(vmLabels.AsSelector())
	glog.V(4).Infof("found %d vm's matching this requirement", len(vms))
	if err != nil {
		return "", fmt.Errorf("error while listing all vms %v", err)
	}

	assigned := false
	vmId := ""
	for _, vm := range vms {
		// Check for Supported environment
		if vmts, found := environments[vm.Labels["environment"]]; found {
			// This virtualmachine is one of the supported environments
			if _, foundVMT := vmts[vm.Spec.VirtualMachineTemplateId]; !foundVMT {
				// ... but this environment does not support this virtualmachinetemplate
				continue
			}
		}else{
			// This virtualmachine is in a non supported environment
			continue;
		}
		if !vm.Status.Allocated && !vm.Status.Tainted {
			// we can assign this vm
			assigned = true
			vmId = vm.Name
		
			// Prefer running machines
			if( vm.Status.Status == hfv1.VmStatusRunning){
				break
			}
		}
	}

	if assigned {
		vmId, err = v.assignVM(vmClaimId, user, vmId)

		if err != nil {
			return "", err
		}

		return vmId, nil
	}

	return vmId, fmt.Errorf("unknown error while assigning next free vm")

}
