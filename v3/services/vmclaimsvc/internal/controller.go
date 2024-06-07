package vmclaimservice

import (
	"context"
	"fmt"
	"math/rand"
	"time"

	hfv1 "github.com/hobbyfarm/gargantua/v3/pkg/apis/hobbyfarm.io/v1"
	hfInformers "github.com/hobbyfarm/gargantua/v3/pkg/client/informers/externalversions"
	hflabels "github.com/hobbyfarm/gargantua/v3/pkg/labels"
	controllers "github.com/hobbyfarm/gargantua/v3/pkg/microservices/controller"
	"github.com/hobbyfarm/gargantua/v3/pkg/util"
	accesscodepb "github.com/hobbyfarm/gargantua/v3/protos/accesscode"
	dbconfigpb "github.com/hobbyfarm/gargantua/v3/protos/dbconfig"
	environmentpb "github.com/hobbyfarm/gargantua/v3/protos/environment"
	generalpb "github.com/hobbyfarm/gargantua/v3/protos/general"
	progresspb "github.com/hobbyfarm/gargantua/v3/protos/progress"
	scheduledeventpb "github.com/hobbyfarm/gargantua/v3/protos/scheduledevent"
	sessionpb "github.com/hobbyfarm/gargantua/v3/protos/session"
	vmpb "github.com/hobbyfarm/gargantua/v3/protos/vm"
	vmclaimpb "github.com/hobbyfarm/gargantua/v3/protos/vmclaim"
	vmtemplatepb "github.com/hobbyfarm/gargantua/v3/protos/vmtemplate"
	"google.golang.org/protobuf/types/known/wrapperspb"

	"github.com/golang/glog"
	hferrors "github.com/hobbyfarm/gargantua/v3/pkg/errors"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/kubernetes"
)

const (
	StaticBindAttemptThreshold  int = 3
	DynamicBindAttemptThreshold int = 2
)

type VMClaimController struct {
	controllers.DelayingWorkqueueController
	controllers.Reconciler
	internalVmClaimServer *GrpcVMClaimServer
	accessCodeClient      accesscodepb.AccessCodeSvcClient
	sessionClient         sessionpb.SessionSvcClient
	progressClient        progresspb.ProgressSvcClient
	environmentClient     environmentpb.EnvironmentSvcClient
	dbConfigClient        dbconfigpb.DynamicBindConfigSvcClient
	vmClient              vmpb.VMSvcClient
	vmTemplateClient      vmtemplatepb.VMTemplateSvcClient
	eventClient           scheduledeventpb.ScheduledEventSvcClient
}

func NewVMClaimController(
	kubeClient *kubernetes.Clientset,
	internalVmClaimServer *GrpcVMClaimServer,
	hfInformerFactory hfInformers.SharedInformerFactory,
	acClient accesscodepb.AccessCodeSvcClient,
	dbConfigClient dbconfigpb.DynamicBindConfigSvcClient,
	environmentClient environmentpb.EnvironmentSvcClient,
	eventClient scheduledeventpb.ScheduledEventSvcClient,
	progressClient progresspb.ProgressSvcClient,
	sessionClient sessionpb.SessionSvcClient,
	vmClient vmpb.VMSvcClient,
	vmTemplateClient vmtemplatepb.VMTemplateSvcClient,
	ctx context.Context,
) (*VMClaimController, error) {
	vmClaimInformer := hfInformerFactory.Hobbyfarm().V1().VirtualMachineClaims().Informer()
	delayingWorkqueueController := *controllers.NewDelayingWorkqueueController(
		ctx,
		vmClaimInformer,
		kubeClient,
		"vmclaim-controller",
		time.Minute*30,
		internalVmClaimServer.vmClaimWorkqueue,
	)

	vmClaimController := &VMClaimController{
		DelayingWorkqueueController: delayingWorkqueueController,
		internalVmClaimServer:       internalVmClaimServer,
		accessCodeClient:            acClient,
		dbConfigClient:              dbConfigClient,
		environmentClient:           environmentClient,
		eventClient:                 eventClient,
		progressClient:              progressClient,
		sessionClient:               sessionClient,
		vmClient:                    vmClient,
		vmTemplateClient:            vmTemplateClient,
	}
	vmClaimController.SetReconciler(vmClaimController)
	vmClaimController.SetWorkScheduler(vmClaimController)

	return vmClaimController, nil
}

func (v *VMClaimController) Reconcile(objName string) error {
	glog.V(8).Infof("reconciling vmclaim %s inside vm claim controller", objName)
	// fetch vmClaim
	vmClaim, err := v.internalVmClaimServer.GetVMClaim(v.Context, &generalpb.GetRequest{Id: objName})
	if err != nil {
		if hferrors.IsGrpcNotFound(err) {
			glog.Infof("vmClaim %s not found on queue.. ignoring", objName)
			return nil
		} else {
			return fmt.Errorf("error while retrieving vmclaim %s from queue with err %v", objName, err)
		}
	}

	// ignore vm objects which are being deleted
	if vmClaim.GetDeletionTimestamp() == nil {
		return v.processVMClaim(vmClaim)
	}
	return nil
}

func (v *VMClaimController) updateVMClaimStatus(bound bool, ready bool, vmc *vmclaimpb.VMClaim) error {
	_, err := v.internalVmClaimServer.UpdateVMClaimStatus(v.Context, &vmclaimpb.UpdateVMClaimStatusRequest{
		Bound: wrapperspb.Bool(bound),
		Ready: wrapperspb.Bool(ready),
	})

	return err
}

func (v *VMClaimController) processVMClaim(vmc *vmclaimpb.VMClaim) (err error) {
	if vmc.Status.Tainted {
		glog.Infof("vmclaim %s is tainted.. cleaning it up", vmc.GetId())
		_, err := v.internalVmClaimServer.DeleteVMClaim(v.Context, &generalpb.ResourceId{Id: vmc.GetId()})
		return err
	}

	if !vmc.Status.Bound && !vmc.Status.Ready {
		// submit VM requests //
		// update status
		if vmc.Status.BindMode == "dynamic" {
			err = v.submitVirtualMachines(vmc)
			if err != nil {
				// VirtualMachines could not be submitted. Delete Session
				glog.Errorf("error processing vmc %s, taint session: %v", vmc.GetId(), err)
				return v.taintSession(vmc.Labels[hflabels.SessionLabel])
			}
		} else if vmc.Status.BindMode == "static" {
			err = v.findVirtualMachines(vmc)
			if err != nil {
				// VirtualMachines could not be bound. Delete Session
				glog.Errorf("error processing vmc %s, taint session: %v", vmc.GetId(), err)
				return v.taintSession(vmc.Labels[hflabels.SessionLabel])
			}
		} else {
			glog.Errorf("vmc bind mode needs to be either dynamic or static.. ignoring this object %s", vmc.GetId())
			return nil
		}

		return v.updateVMClaimStatus(true, false, vmc)
	}

	if vmc.Status.Bound && !vmc.Status.Ready {
		// reconcile triggered by VM being ready
		// lets check the VM's
		ready, err := v.checkVMStatus(vmc)
		if err != nil {
			glog.Errorf("error checking vmStatus for vmc: %s %v", vmc.GetId(), err)
			return err
		}
		// update status
		glog.V(4).Infof("vm's have been requested for vmclaim: %s", vmc.GetId())
		return v.updateVMClaimStatus(true, ready, vmc)
	}

	if vmc.Status.Bound && vmc.Status.Ready {
		// nothing else needs to be done.. ignore and move along
		glog.V(4).Infof("vmclaim %s is ready", vmc.GetId())
	}

	return nil
}

func (v *VMClaimController) taintSession(session string) error {
	_, err := v.sessionClient.UpdateSessionStatus(v.Context, &sessionpb.UpdateSessionStatusRequest{
		Id:             session,
		ExpirationTime: time.Now().Format(time.UnixDate),
		Active:         wrapperspb.Bool(false),
	})
	if err != nil {
		return err
	}

	// Remove outstanding Progresses as there was an error with this session
	_, err = v.progressClient.DeleteCollectionProgress(v.Context, &generalpb.ListOptions{
		LabelSelector: fmt.Sprintf("%s=%s,finished=false", hflabels.SessionLabel, session),
	})

	return err
}

type VMEnvironment struct {
	Environment              *environmentpb.Environment
	DynamicBindConfiguration *dbconfigpb.DynamicBindConfig
}

func (v *VMClaimController) submitVirtualMachines(vmc *vmclaimpb.VMClaim) (err error) {
	accessCode, ok := vmc.Labels[hflabels.AccessCodeLabel]
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
	for _, vmDetails := range vmc.GetVms() {
		if count, found := requiredTemplateCount[vmDetails.Template]; found {
			requiredTemplateCount[vmDetails.Template] = count + 1
		} else {
			requiredTemplateCount[vmDetails.Template] = 1
		}
	}

	environmentMap := make(map[string]VMEnvironment)                                                              // Maps node to the environment it should use
	bestDBC, err := v.findBestDBCForVMs(dbcList, requiredTemplateCount, vmc.Labels[hflabels.ScheduledEventLabel]) // Try to find if one environment can provision all VMs

	if err != nil {
		// We can not provision all VirtualMachines in one environment. Figure out which environments we want to use

		reservedCapacity := make(map[string]map[string]int) // EnvironmentID -> TemplateID -> Count
		// Initialize reservedCapacity with 0 for all environments + associated templates
		for _, environment := range environments {
			reserved := make(map[string]int)
			for template := range environment.GetTemplateMapping() {
				reserved[template] = 0
			}
			reservedCapacity[environment.GetId()] = reserved
		}
		for vmName, vmDetails := range vmc.GetVms() {
			env, dbc, err := v.findSuitableEnvironmentForVMTemplate(environments, dbcList, vmDetails.Template, reservedCapacity, vmc.Labels[hflabels.ScheduledEventLabel])
			if err != nil {
				glog.Errorf("no suitable environment for %s (%s): %v", vmName, vmDetails.GetTemplate(), err)
				return err
			}
			environmentMap[vmName] = VMEnvironment{env, dbc}
			reservedCapacity[env.GetId()][vmDetails.GetTemplate()] += 1
		}
	} else {
		// One DBC for them all
		enviroment := &environmentpb.Environment{}
		for _, e := range environments {
			if e.GetId() == bestDBC.GetEnvironment() {
				enviroment = e
				break
			}
		}
		for vmName := range vmc.GetVms() {
			environmentMap[vmName] = VMEnvironment{enviroment, bestDBC}
		}
	}

	vmMap := make(map[string]*vmclaimpb.VMClaimVM)
	for vmName, vmDetails := range vmc.GetVms() {
		genName := fmt.Sprintf("%s-%08x", vmc.GetBaseName(), rand.Uint32())
		environment := environmentMap[vmName].Environment
		dbc := environmentMap[vmName].DynamicBindConfiguration
		vm := &vmpb.CreateVMRequest{
			Id:           genName,
			VmTemplateId: vmDetails.Template,
			Protocol:     "ssh",
			SecretName:   "",
			VmClaimId:    vmc.GetId(),
			VmClaimUid:   vmc.GetUid(),
			User:         vmc.GetUserId(),
			Provision:    true,
			Labels: map[string]string{
				"dynamic":                       "true",
				"vmc":                           vmc.GetId(),
				hflabels.EnvironmentLabel:       environment.GetId(),
				"bound":                         "true",
				"ready":                         "false",
				hflabels.VirtualMachineTemplate: vmDetails.Template,
				hflabels.ScheduledEventLabel:    seName,
				"restrictedbind":                fmt.Sprintf("%t", dbc.GetRestrictedBind()),
			},
		}
		// used to later repopulate the info back //
		vmMap[vmName] = &vmclaimpb.VMClaimVM{
			Template: vmDetails.Template,
			VmId:     genName,
		}

		vmt, err := v.vmTemplateClient.GetVMTemplate(v.Context, &generalpb.GetRequest{Id: vmDetails.Template, LoadFromCache: true})
		if err != nil {
			glog.Errorf("error getting vmt %v", err)
			return err
		}

		config := util.GetVMConfig(environment, vmt)

		protocol, exists := config["protocol"]
		if exists {
			vm.Protocol = protocol
		}

		sshUser, exists := config["ssh_username"]
		if exists {
			vm.SshUsername = sshUser
		}

		// extra label to indicate external provisioning so tfpcontroller ignores this request //
		if provisionMethod, ok := environment.Annotations["hobbyfarm.io/provisioner"]; ok && provisionMethod != "" {
			vm.Labels["hobbyfarm.io/provisioner"] = provisionMethod
			vm.Provision = false
		}

		if dbc.RestrictedBind {
			vm.Labels["restrictedbindvalue"] = dbc.RestrictedBindValue
		}

		_, err = v.vmClient.CreateVM(v.Context, vm)
		if err != nil {
			return err
		}

		_, err = v.vmClient.UpdateVMStatus(v.Context, &vmpb.UpdateVMStatusRequest{
			Id:            genName,
			Status:        string(hfv1.VmStatusRFP),
			Allocated:     wrapperspb.Bool(true),
			Tainted:       wrapperspb.Bool(false),
			WsEndpoint:    environment.GetWsEndpoint(),
			EnvironmentId: environment.GetId(),
			PublicIp:      wrapperspb.String(""),
			PrivateIp:     wrapperspb.String(""),
		})
		if err != nil {
			return err
		}
	}

	vmc.Vms = vmMap

	_, err = v.internalVmClaimServer.UpdateVMClaim(v.Context, &vmclaimpb.UpdateVMClaimRequest{
		Vmset: vmMap,
	})
	if err != nil {
		return err
	}

	return nil
}

// Based on the given VirtualMachineClaim and ScheduledEvent find all suitable Environments (e.g. environment provides required VMTeplate & ScheduledEvents allows this environment and VMTemplate configuration etc.)
func (v *VMClaimController) findEnvironmentsForVM(accessCode string, vmc *vmclaimpb.VMClaim) (environments []*environmentpb.Environment, seName string, dbc []*dbconfigpb.DynamicBindConfig, err error) {
	seName, _, err = v.findScheduledEvent(accessCode)
	if err != nil {
		return environments, seName, dbc, err
	}

	dbcList, err := v.dbConfigClient.ListDynamicBindConfig(v.Context, &generalpb.ListOptions{
		LabelSelector: fmt.Sprintf("%s=%s", hflabels.ScheduledEventLabel, seName),
	})

	if err != nil {
		glog.Errorf("error listing dbc %v", err)
		return environments, seName, dbc, err
	}

	for _, dbc := range dbcList.GetDbConfig() {
		env, err := v.environmentClient.GetEnvironment(v.Context, &generalpb.GetRequest{Id: dbc.GetId()})
		if err != nil {
			glog.Errorf("error fetching environment %v", err)
			return environments, seName, dbcList.GetDbConfig(), err
		}
		environments = append(environments, env)
	}

	if len(dbcList.GetDbConfig()) < 1 {
		return environments, seName, dbc, fmt.Errorf("incorrect number of dbc matching sessionName found")
	}

	return environments, seName, dbcList.GetDbConfig(), err
}

// Can one DBC provide all VMs when considering the limits? Return the DBC if there exists one
func (v *VMClaimController) findBestDBCForVMs(dbcList []*dbconfigpb.DynamicBindConfig, requiredTemplateCount map[string]int, scheduledEvent string) (*dbconfigpb.DynamicBindConfig, error) {
	// Try to find best possible environment / DBC = All required VMs can be provisioned here
	for _, dbc := range dbcList {
		satisfiedDBC := true
		env, err := v.environmentClient.GetEnvironment(v.Context, &generalpb.GetRequest{Id: dbc.Environment})
		if err != nil {
			return &dbconfigpb.DynamicBindConfig{}, fmt.Errorf("error fetching environment")
		}
		for requiredTemplate, requiredCount := range requiredTemplateCount {
			dbcCapacity, foundDBC := dbc.BurstCountCapacity[requiredTemplate]
			envCapacity, foundEnv := env.CountCapacity[requiredTemplate]
			if foundDBC && foundEnv {
				// Does the DBC satisfy this amount?
				count, err := util.CountMachinesPerTemplateAndEnvironmentAndScheduledEvent(v.Context, v.vmClient, requiredTemplate, dbc.Environment, scheduledEvent)
				if err != nil {
					satisfiedDBC = false
					break
				}
				if requiredCount >= (int(dbcCapacity) - count) {
					satisfiedDBC = false
					break
				}

				// Does the environment satisfy this amount?
				count, err = util.CountMachinesPerTemplateAndEnvironment(v.Context, v.vmClient, requiredTemplate, dbc.Environment)
				if err != nil {
					satisfiedDBC = false
					break
				}
				if requiredCount >= (int(envCapacity) - count) {
					satisfiedDBC = false
					break
				}

			} else {
				satisfiedDBC = false
				break
			}
		}

		if satisfiedDBC {
			// This DBC works for all templates and has the required Counts available!
			glog.V(4).Infof("found best environment suitable for all VMs: %s", dbc.GetEnvironment())
			return dbc, nil
		}
	}
	return &dbconfigpb.DynamicBindConfig{}, fmt.Errorf("there is no best environment")
}

func (v *VMClaimController) findSuitableEnvironmentForVMTemplate(environments []*environmentpb.Environment, dbcList []*dbconfigpb.DynamicBindConfig, template string, reservedCapacity map[string]map[string]int, scheduledEvent string) (*environmentpb.Environment, *dbconfigpb.DynamicBindConfig, error) {
	for _, environment := range environments {
		countEnv, err := util.CountMachinesPerTemplateAndEnvironment(v.Context, v.vmClient, template, environment.GetId())
		if err != nil {
			continue
		}
		// We have also reserved capacity for other VMs
		countEnv += reservedCapacity[environment.GetId()][template]

		if countEnv >= int(environment.GetCountCapacity()[template]) {
			// Environment is at limit
			continue
		}

		countDBC, err := util.CountMachinesPerTemplateAndEnvironmentAndScheduledEvent(v.Context, v.vmClient, template, environment.GetId(), scheduledEvent)
		if err != nil {
			continue
		}
		// We have also reserved capacity for other VMs
		countDBC += reservedCapacity[environment.GetId()][template]

		// found environment that satisfies capacity for this template
		for _, dbc := range dbcList {
			if dbc.GetEnvironment() == environment.GetId() {
				if capacity, found := dbc.GetBurstCountCapacity()[template]; found {
					if countDBC < int(capacity) {
						// Capacity also satisfied for environment + scheduledEvent via DBC
						return environment, dbc, nil
					}
				}
				break
			}
		}

	}

	return &environmentpb.Environment{}, &dbconfigpb.DynamicBindConfig{}, fmt.Errorf("no suitable environment found. capacity reached")
}

func (v *VMClaimController) checkVMStatus(vmc *vmclaimpb.VMClaim) (ready bool, err error) {
	ready = true
	for _, vmTemplate := range vmc.Vms {
		vm, err := v.vmClient.GetVM(v.Context, &generalpb.GetRequest{Id: vmTemplate.GetVmId()})
		if err != nil {
			return ready, err
		}
		if vm.Status != nil && vm.Status.Status == string(hfv1.VmStatusRunning) {
			ready = ready && true
		} else {
			ready = ready && false
		}
	}

	return ready, err
}

func (v *VMClaimController) findScheduledEvent(accessCode string) (schedEvent string, environments map[string]*scheduledeventpb.VMTemplateCountMap, err error) {
	ac, err := v.accessCodeClient.GetAccessCodeWithOTACs(v.Context, &generalpb.ResourceId{Id: accessCode})
	if err != nil {
		return schedEvent, environments, err
	}

	se, err := v.eventClient.GetScheduledEvent(v.Context, &generalpb.GetRequest{Id: ac.Labels[hflabels.ScheduledEventLabel]})

	if err != nil {
		return schedEvent, environments, err
	}

	schedEvent = se.GetName()
	environments = se.GetRequiredVms()
	return schedEvent, environments, nil
}

func (v *VMClaimController) findVirtualMachines(vmc *vmclaimpb.VMClaim) (err error) {
	accessCode, ok := vmc.Labels[hflabels.AccessCodeLabel]
	if !ok {
		glog.Error("accessCode label not set on vmc, aborting")
		return fmt.Errorf("accessCode label not set on vmc, aborting")
	}
	_, environments, err := v.findScheduledEvent(accessCode)

	if err != nil {
		glog.Error("error finding scheduledevent during static bind")
		return err
	}

	vmMap := make(map[string]*vmclaimpb.VMClaimVM)
	for name, vmStruct := range vmc.GetVms() {
		if vmStruct.GetVmId() == "" {
			glog.Info("assigning a vm")
			vmID, err := v.assignNextFreeVM(vmc.GetId(), vmc.GetUserId(), environments, vmStruct.Template, vmc.RestrictedBind, vmc.RestrictedBindValue)
			if err != nil {
				// If we run into any issue assigning a VM we need to unassign the previously assigned VMs
				for _, vm := range vmMap {
					v.unassignVM(vm.GetVmId())
				}
				return err
			}
			vmMap[name] = &vmclaimpb.VMClaimVM{
				Template: vmStruct.Template,
				VmId:     vmID,
			}
		}
	}
	_, err = v.internalVmClaimServer.UpdateVMClaim(v.Context, &vmclaimpb.UpdateVMClaimRequest{
		Vmset: vmMap,
	})
	if err != nil {
		return err
	}

	return nil
}

func (v *VMClaimController) assignVM(vmClaimId string, user string, vmId string) error {
	_, err := v.vmClient.UpdateVM(v.Context, &vmpb.UpdateVMRequest{
		Id:        vmId,
		Bound:     "true",
		VmClaimId: wrapperspb.String(vmClaimId),
		User:      wrapperspb.String(user),
	})
	if err != nil {
		return err
	}
	_, err = v.vmClient.UpdateVMStatus(v.Context, &vmpb.UpdateVMStatusRequest{
		Id:        vmId,
		Allocated: wrapperspb.Bool(true),
	})
	if err != nil {
		return err
	}

	glog.V(4).Infof("updated result for virtual machine")
	return nil
}

func (v *VMClaimController) unassignVM(vmId string) (string, error) {
	_, err := v.vmClient.UpdateVM(v.Context, &vmpb.UpdateVMRequest{
		Id:        vmId,
		Bound:     "false",
		VmClaimId: wrapperspb.String(""),
		User:      wrapperspb.String(""),
	})
	if err != nil {
		return "", err
	}
	_, err = v.vmClient.UpdateVMStatus(v.Context, &vmpb.UpdateVMStatusRequest{
		Id:        vmId,
		Allocated: wrapperspb.Bool(false),
	})
	if err != nil {
		return "", err
	}

	return vmId, nil
}

func (v *VMClaimController) assignNextFreeVM(vmClaimId string, user string, environments map[string]*scheduledeventpb.VMTemplateCountMap, template string, restrictedBind bool, restrictedBindValue string) (string, error) {
	vmLabels := labels.Set{
		"bound":                         "false",
		hflabels.VirtualMachineTemplate: template,
	}

	if restrictedBind {
		vmLabels["restrictedbind"] = "true"
		vmLabels["restrictedbindvalue"] = restrictedBindValue
	} else {
		vmLabels["restrictedbind"] = "false"
	}

	vmList, err := v.vmClient.ListVM(v.Context, &generalpb.ListOptions{LabelSelector: vmLabels.AsSelector().String(), LoadFromCache: true})
	vms := vmList.GetVms()
	glog.V(4).Infof("found %d vm's matching this requirement", len(vms))
	if err != nil {
		return "", fmt.Errorf("error while listing all vms %v", err)
	}

	if len(vms) == 0 {
		return "", fmt.Errorf("all static VMs are in use, no static VMs matching template: %s", template)
	}

	assigned := false
	vmId := ""
	for _, vm := range vms {
		// Check for Supported environment
		if vmts, found := environments[vm.Labels[hflabels.EnvironmentLabel]]; found {
			// This virtualmachine is one of the supported environments
			if _, foundVMT := vmts.GetVmTemplateCounts()[vm.VmTemplateId]; !foundVMT {
				// ... but this environment does not support this virtualmachinetemplate
				continue
			}
		} else {
			// This virtualmachine is in a non supported environment
			continue
		}
		if !vm.Status.Allocated && !vm.Status.Tainted {
			// we can assign this vm
			assigned = true
			vmId = vm.GetId()

			// Prefer running machines
			if vm.Status.Status == string(hfv1.VmStatusRunning) {
				break
			}
		}
	}

	if assigned {
		err = v.assignVM(vmClaimId, user, vmId)

		if err != nil {
			return "", err
		}

		return vmId, nil
	}

	return vmId, fmt.Errorf("unknown error while assigning next free vm")

}
