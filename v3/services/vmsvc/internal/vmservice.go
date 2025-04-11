package vmservice

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	hferrors "github.com/hobbyfarm/gargantua/v3/pkg/errors"
	hflabels "github.com/hobbyfarm/gargantua/v3/pkg/labels"
	"github.com/hobbyfarm/gargantua/v3/pkg/rbac"
	"github.com/hobbyfarm/gargantua/v3/pkg/util"
	generalpb "github.com/hobbyfarm/gargantua/v3/protos/general"
	vmpb "github.com/hobbyfarm/gargantua/v3/protos/vm"

	"github.com/golang/glog"
	"github.com/gorilla/mux"
)

const (
	idIndex        = "vms.hobbyfarm.io/id-index"
	resourcePlural = rbac.ResourcePluralVM
)

type PreparedVirtualMachine struct {
	Id                       string `json:"id"`
	CreationTimestamp        string `json:"creation_timestamp"`
	VirtualMachineTemplateId string `json:"vm_template_id"`
	SshUsername              string `json:"ssh_username"`
	Protocol                 string `json:"protocol"`
	SecretName               string `json:"secret_name"` // this refers to the secret name for the keypair
	VirtualMachineClaimId    string `json:"vm_claim_id"`
	UserId                   string `json:"user"`
	Provision                bool   `json:"provision"`
	VirtualMachineSetId      string `json:"vm_set_id"`
	Status                   string `json:"status"` // default is nothing, but could be one of the following: readyforprovisioning, provisioning, running, terminating
	Allocated                bool   `json:"allocated"`
	Tainted                  bool   `json:"tainted"`
	PublicIP                 string `json:"public_ip"`
	PrivateIP                string `json:"private_ip"`
	EnvironmentId            string `json:"environment_id"`
	Hostname                 string `json:"hostname"`          // ideally <hostname>.<enviroment dnssuffix> should be the FQDN to this machine
	TFState                  string `json:"tfstate,omitempty"` // Terraform state name
	WsEndpoint               string `json:"ws_endpoint"`
}

/*
* Checks if VMTemplate used to create VM has "webinterfaces" in ConfigMap.
* Returns those webinterface definitions or http Error Codes.
 */
func (vms VMServer) getWebinterfaces(w http.ResponseWriter, r *http.Request) {
	// Check if User has access to VMs
	user, err := rbac.AuthenticateRequest(r, vms.authnClient)
	if err != nil {
		util.ReturnHTTPMessage(w, r, 403, "forbidden", "no access to get vm")
		return
	}
	impersonatedUserId := user.GetId()

	vars := mux.Vars(r)
	// Check if id for the VM was provided
	vmId := vars["vm_id"]
	if len(vmId) == 0 {
		util.ReturnHTTPMessage(w, r, 400, "bad request", "no vm id passed in")
		return
	}
	// Get the VM, Error if none is found for the given id
	vm, err := vms.internalVMServer.GetVM(r.Context(), &generalpb.GetRequest{Id: vmId, LoadFromCache: true})
	if err != nil {
		glog.Errorf("error retrieving virtual machine %s from cache: %s", vmId, hferrors.GetErrorMessage(err))
		if hferrors.IsGrpcNotFound(err) {
			errMsg := fmt.Sprintf("vm %s not found", vmId)
			util.ReturnHTTPMessage(w, r, http.StatusNotFound, "not found", errMsg)
			return
		}
		errMsg := fmt.Sprintf("error retrieving vm %s", vmId)
		util.ReturnHTTPMessage(w, r, http.StatusInternalServerError, "error", errMsg)
		return
	}

	// Check if the VM belongs to the User or User has RBAC-Rights to access VMs
	if vm.GetUser() != impersonatedUserId {
		authrResponse, err := rbac.AuthorizeSimple(r, vms.authrClient, impersonatedUserId, rbac.HobbyfarmPermission(resourcePlural, rbac.VerbGet))
		if err != nil || !authrResponse.Success {
			glog.Errorf("user forbidden from accessing vm id %s", vm.GetId())
			util.ReturnHTTPMessage(w, r, 403, "forbidden", "no access to get vm")
			return
		}
	}

	// Get the corresponding VMTemplate for the VM and Check for "ide"
	vmtId := vm.GetVmTemplateId()
	vmt, err := vms.vmTemplateClient.GetVMTemplate(r.Context(), &generalpb.GetRequest{Id: vmtId})
	if err != nil {
		glog.Errorf("error retrieving vm's vm template %s: %s", vmtId, hferrors.GetErrorMessage(err))
		if hferrors.IsGrpcNotFound(err) {
			errMsg := fmt.Sprintf("vm template %s of vm %s not found", vmtId, vmId)
			util.ReturnHTTPMessage(w, r, http.StatusNotFound, "not found", errMsg)
			return
		}
		errMsg := fmt.Sprintf("error retrieving vm template %s of vm %s", vmtId, vmId)
		util.ReturnHTTPMessage(w, r, http.StatusInternalServerError, "error", errMsg)
		return
	}

	services, found := vmt.GetConfigMap()["webinterfaces"]
	if !found {
		util.ReturnHTTPMessage(w, r, 404, "error", "No Webinterfaces found for this VM")
		return
	}

	encodedWebinterfaceDefinitions, err := json.Marshal(services)
	if err != nil {
		glog.Error(err)
	}
	util.ReturnHTTPContent(w, r, 200, "success", encodedWebinterfaceDefinitions)
}

func (vms VMServer) GetVMFunc(w http.ResponseWriter, r *http.Request) {
	user, err := rbac.AuthenticateRequest(r, vms.authnClient)
	if err != nil {
		util.ReturnHTTPMessage(w, r, 403, "forbidden", "no access to get vm")
		return
	}
	impersonatedUserId := user.GetId()

	vars := mux.Vars(r)

	vmId := vars["vm_id"]

	if len(vmId) == 0 {
		util.ReturnHTTPMessage(w, r, 400, "bad request", "no vm id passed in")
		return
	}

	vm, err := vms.internalVMServer.GetVM(r.Context(), &generalpb.GetRequest{Id: vmId, LoadFromCache: true})
	if err != nil {
		glog.Errorf("error retrieving virtual machine %s from cache: %s", vmId, hferrors.GetErrorMessage(err))
		if hferrors.IsGrpcNotFound(err) {
			errMsg := fmt.Sprintf("vm %s not found", vmId)
			util.ReturnHTTPMessage(w, r, http.StatusNotFound, "not found", errMsg)
			return
		}
		errMsg := fmt.Sprintf("error retrieving vm %s", vmId)
		util.ReturnHTTPMessage(w, r, http.StatusInternalServerError, "error", errMsg)
		return
	}

	if vm.GetUser() != impersonatedUserId {
		authrResponse, err := rbac.AuthorizeSimple(r, vms.authrClient, impersonatedUserId, rbac.HobbyfarmPermission(resourcePlural, rbac.VerbGet))
		if err != nil || !authrResponse.Success {
			glog.Errorf("user forbidden from accessing vm id %s", vm.GetId())
			util.ReturnHTTPMessage(w, r, 403, "forbidden", "no access to get vm")
			return
		}
	}

	preparedVM := getPreparedVM(vm)
	encodedVM, err := json.Marshal(preparedVM)
	if err != nil {
		glog.Error(err)
	}
	util.ReturnHTTPContent(w, r, 200, "success", encodedVM)

	glog.V(2).Infof("retrieved vm %s", vm.GetId())
}

func (vms VMServer) DeleteVMFunc(w http.ResponseWriter, r *http.Request) {
	user, err := rbac.AuthenticateRequest(r, vms.authnClient)
	if err != nil {
		util.ReturnHTTPMessage(w, r, 403, "forbidden", "no access to get vm")
		return
	}
	impersonatedUserId := user.GetId()

	authrResponse, err := rbac.AuthorizeSimple(r, vms.authrClient, impersonatedUserId, rbac.HobbyfarmPermission(resourcePlural, rbac.VerbDelete))
	if err != nil || !authrResponse.Success {
		glog.Errorf("user forbidden from deleting vms")
		util.ReturnHTTPMessage(w, r, 403, "forbidden", "no access to delete vm")
		return
	}

	vars := mux.Vars(r)

	vmId := vars["vm_id"]

	if len(vmId) == 0 {
		util.ReturnHTTPMessage(w, r, 400, "bad request", "no vm id passed in")
		return
	}

	_, err = vms.internalVMServer.DeleteVM(r.Context(), &generalpb.ResourceId{Id: vmId})
	if err != nil {
		glog.Errorf("error retrieving virtual machine %s from cache: %s", vmId, hferrors.GetErrorMessage(err))
		if hferrors.IsGrpcNotFound(err) {
			errMsg := fmt.Sprintf("vm %s not found", vmId)
			util.ReturnHTTPMessage(w, r, http.StatusNotFound, "not found", errMsg)
			return
		}
		errMsg := fmt.Sprintf("error deleting vm %s", vmId)
		util.ReturnHTTPMessage(w, r, http.StatusInternalServerError, "error", errMsg)
		return
	}

	_, err = vms.internalVMServer.UpdateVM(r.Context(), &vmpb.UpdateVMRequest{Id: vmId, Finalizers: &generalpb.StringArray{}})
	if err != nil {
		glog.Errorf("error updating finalizers for virtual machine %s, :%s", vmId, hferrors.GetErrorMessage(err))
		errMsg := fmt.Sprintf("error updating vm %s", vmId)
		util.ReturnHTTPMessage(w, r, http.StatusInternalServerError, "error", errMsg)
		return
	}

	util.ReturnHTTPMessage(w, r, 200, "success", "deleted successfully")

	glog.V(2).Infof("deleted vm %s", vmId)
}

func (vms VMServer) GetVMListFunc(w http.ResponseWriter, r *http.Request, listOptions *generalpb.ListOptions) {
	user, err := rbac.AuthenticateRequest(r, vms.authnClient)
	if err != nil {
		util.ReturnHTTPMessage(w, r, 401, "unauthorized", "authentication failed")
		return
	}

	impersonatedUserId := user.GetId()
	authrResponse, err := rbac.AuthorizeSimple(r, vms.authrClient, impersonatedUserId, rbac.HobbyfarmPermission(resourcePlural, rbac.VerbList))
	if err != nil || !authrResponse.Success {
		util.ReturnHTTPMessage(w, r, 403, "forbidden", "no access to list virtualmachines")
		return
	}

	vmList, err := vms.internalVMServer.ListVM(r.Context(), listOptions)
	if err != nil {
		glog.Errorf("error while retrieving vms %s", hferrors.GetErrorMessage(err))
		util.ReturnHTTPMessage(w, r, 500, "error", "error retreiving vms")
		return
	}

	preparedVMs := []PreparedVirtualMachine{}
	for _, vm := range vmList.GetVms() {
		pVM := getPreparedVM(vm)
		preparedVMs = append(preparedVMs, pVM)
	}

	encodedVMs, err := json.Marshal(preparedVMs)
	if err != nil {
		glog.Error(err)
	}
	util.ReturnHTTPContent(w, r, 200, "success", encodedVMs)
}

func (vms VMServer) GetVMListByScheduledEventFunc(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)

	id := vars["se_id"]

	if len(id) == 0 {
		util.ReturnHTTPMessage(w, r, 400, "bad request", "no scheduledEvent id passed in")
		return
	}

	lo := &generalpb.ListOptions{LabelSelector: fmt.Sprintf("%s=%s", hflabels.ScheduledEventLabel, id)}

	vms.GetVMListFunc(w, r, lo)
}

func (vms VMServer) CountByScheduledEvent(w http.ResponseWriter, r *http.Request) {
	user, err := rbac.AuthenticateRequest(r, vms.authnClient)
	if err != nil {
		util.ReturnHTTPMessage(w, r, 401, "unauthorized", "authentication failed")
		return
	}

	impersonatedUserId := user.GetId()
	authrResponse, err := rbac.AuthorizeSimple(r, vms.authrClient, impersonatedUserId, rbac.HobbyfarmPermission(resourcePlural, rbac.VerbList))
	if err != nil || !authrResponse.Success {
		util.ReturnHTTPMessage(w, r, 403, "forbidden", "no access to list virtualmachines")
		return
	}

	vmList, err := vms.internalVMServer.ListVM(r.Context(), &generalpb.ListOptions{})
	if err != nil {
		glog.Errorf("error while retrieving virtualmachine %v", err)
		util.ReturnHTTPMessage(w, r, 500, "error", "no virtualmachine found")
		return
	}

	countMap := map[string]int{}
	for _, vm := range vmList.GetVms() {
		se := vm.GetLabels()[hflabels.ScheduledEventLabel]
		if _, ok := countMap[se]; ok {
			countMap[se] = countMap[se] + 1
		} else {
			countMap[se] = 1
		}
	}

	encodedMap, err := json.Marshal(countMap)
	if err != nil {
		glog.Error(err)
	}
	util.ReturnHTTPContent(w, r, 200, "success", encodedMap)
}

func (vms VMServer) GetAllVMListFunc(w http.ResponseWriter, r *http.Request) {
	vms.GetVMListFunc(w, r, &generalpb.ListOptions{})
}

func getPreparedVM(vm *vmpb.VM) PreparedVirtualMachine {
	return PreparedVirtualMachine{
		Id:                       vm.GetId(),
		CreationTimestamp:        vm.GetCreationTimestamp().AsTime().Format(time.UnixDate),
		VirtualMachineTemplateId: vm.GetVmTemplateId(),
		SshUsername:              vm.GetSshUsername(),
		Protocol:                 vm.GetProtocol(),
		SecretName:               vm.GetSecretName(),
		VirtualMachineClaimId:    vm.GetVmClaimId(),
		UserId:                   vm.GetUser(),
		Provision:                vm.GetProvision(),
		VirtualMachineSetId:      vm.GetVmSetId(),
		Status:                   vm.GetStatus().GetStatus(),
		Allocated:                vm.GetStatus().GetAllocated(),
		Tainted:                  vm.GetStatus().GetTainted(),
		PublicIP:                 vm.GetStatus().GetPublicIp(),
		PrivateIP:                vm.GetStatus().GetPrivateIp(),
		EnvironmentId:            vm.GetStatus().GetEnvironmentId(),
		Hostname:                 vm.GetStatus().GetHostname(),
		TFState:                  vm.GetStatus().GetTfstate(),
		WsEndpoint:               vm.GetStatus().GetWsEndpoint(),
	}
}
