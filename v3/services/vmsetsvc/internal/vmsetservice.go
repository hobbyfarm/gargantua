package vmsetservice

import (
	"encoding/json"
	"fmt"
	"net/http"

	hferrors "github.com/hobbyfarm/gargantua/v3/pkg/errors"
	hflabels "github.com/hobbyfarm/gargantua/v3/pkg/labels"
	"github.com/hobbyfarm/gargantua/v3/pkg/rbac"
	util2 "github.com/hobbyfarm/gargantua/v3/pkg/util"

	"github.com/golang/glog"
	"github.com/gorilla/mux"
	generalpb "github.com/hobbyfarm/gargantua/v3/protos/general"
	vmsetpb "github.com/hobbyfarm/gargantua/v3/protos/vmset"
)

const (
	resourcePlural = rbac.ResourcePluralVMSet
)

type PreparedVirtualMachineSet struct {
	Id                  string                 `json:"id"`
	Count               uint32                 `json:"count"`
	Environment         string                 `json:"environment"`
	VMTemplate          string                 `json:"vm_template"`
	BaseName            string                 `json:"base_name"`
	RestrictedBind      bool                   `json:"restricted_bind"`
	RestrictedBindValue string                 `json:"restricted_bind_value"`
	Machines            []*vmsetpb.VMProvision `json:"machines"`
	AvailableCount      uint32                 `json:"available"`
	ProvisionedCount    uint32                 `json:"provisioned"`
}

func (vms VMSetServer) GetVMSetListByScheduledEventFunc(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)

	id := vars["se_id"]

	if len(id) == 0 {
		util2.ReturnHTTPMessage(w, r, 400, "bad request", "no scheduledEvent id passed in")
		return
	}

	lo := &generalpb.ListOptions{LabelSelector: fmt.Sprintf("%s=%s", hflabels.ScheduledEventLabel, id)}

	vms.GetVMSetListFunc(w, r, lo)
}

func (vms VMSetServer) GetAllVMSetListFunc(w http.ResponseWriter, r *http.Request) {
	vms.GetVMSetListFunc(w, r, &generalpb.ListOptions{})
}

func (vms VMSetServer) GetVMSetListFunc(w http.ResponseWriter, r *http.Request, listOptions *generalpb.ListOptions) {
	user, err := rbac.AuthenticateRequest(r, vms.authnClient)
	if err != nil {
		util2.ReturnHTTPMessage(w, r, 401, "unauthorized", "authentication failed")
		return
	}

	impersonatedUserId := user.GetId()
	authrResponse, err := rbac.AuthorizeSimple(r, vms.authrClient, impersonatedUserId, rbac.HobbyfarmPermission(resourcePlural, rbac.VerbList))
	if err != nil || !authrResponse.Success {
		util2.ReturnHTTPMessage(w, r, 403, "forbidden", "no access to list vmsets")
		return
	}

	vmSetList, err := vms.internalVMSetServer.ListVMSet(r.Context(), listOptions)

	if err != nil {
		glog.Errorf("error while retrieving vmsets: %s", hferrors.GetErrorMessage(err))
		util2.ReturnHTTPMessage(w, r, 500, "error", "error retreiving vmsets")
		return
	}

	preparedVMSets := []PreparedVirtualMachineSet{}
	for _, vmSet := range vmSetList.GetVmsets() {
		pVMSet := PreparedVirtualMachineSet{
			Id:                  vmSet.GetId(),
			Count:               vmSet.GetCount(),
			Environment:         vmSet.GetEnvironment(),
			VMTemplate:          vmSet.GetVmTemplate(),
			BaseName:            vmSet.GetBaseName(),
			RestrictedBind:      vmSet.GetRestrictedBind(),
			RestrictedBindValue: vmSet.GetRestrictedBindValue(),
			Machines:            vmSet.GetStatus().GetMachines(),
			AvailableCount:      vmSet.GetStatus().GetAvailable(),
			ProvisionedCount:    vmSet.GetStatus().GetProvisioned(),
		}
		preparedVMSets = append(preparedVMSets, pVMSet)
	}

	encodedVMSets, err := json.Marshal(preparedVMSets)
	if err != nil {
		glog.Error(err)
	}
	util2.ReturnHTTPContent(w, r, 200, "success", encodedVMSets)
}
