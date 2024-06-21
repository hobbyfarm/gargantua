package vmclaimservice

import (
	"encoding/json"
	"fmt"
	"net/http"

	hferrors "github.com/hobbyfarm/gargantua/v3/pkg/errors"
	"github.com/hobbyfarm/gargantua/v3/pkg/rbac"
	"github.com/hobbyfarm/gargantua/v3/pkg/util"
	generalpb "github.com/hobbyfarm/gargantua/v3/protos/general"
	vmclaimpb "github.com/hobbyfarm/gargantua/v3/protos/vmclaim"

	"github.com/golang/glog"
	"github.com/gorilla/mux"
)

type PreparedVirtualMachineClaim struct {
	Id                  string                          `json:"id"`
	UserId              string                          `json:"user"`
	RestrictedBind      bool                            `json:"restricted_bind"`
	RestrictedBindValue string                          `json:"restricted_bind_value"`
	VirtualMachines     map[string]*vmclaimpb.VMClaimVM `json:"vm"`
	DynamicCapable      bool                            `json:"dynamic_bind_capable"`
	BaseName            string                          `json:"base_name"`
	BindMode            string                          `json:"bind_mode"`
	StaticBindAttempts  uint32                          `json:"static_bind_attempts"`
	Bound               bool                            `json:"bound"`
	Ready               bool                            `json:"ready"`
	Tainted             bool                            `json:"tainted"`
}

func (vmcs VMClaimServer) GetVMClaimFunc(w http.ResponseWriter, r *http.Request) {
	user, err := rbac.AuthenticateRequest(r, vmcs.authnClient)
	if err != nil {
		util.ReturnHTTPMessage(w, r, 403, "forbidden", "no access to get vmc")
		return
	}

	vars := mux.Vars(r)

	vmcId := vars["vm_claim_id"]

	if len(vmcId) == 0 {
		util.ReturnHTTPMessage(w, r, 400, "bad request", "no vmc id passed in")
		return
	}

	vmc, err := vmcs.internalVMClaimServer.GetVMClaim(r.Context(), &generalpb.GetRequest{Id: vmcId, LoadFromCache: true})
	if err != nil {
		glog.Error(hferrors.GetErrorMessage(err))
		if hferrors.IsGrpcNotFound(err) {
			util.ReturnHTTPMessage(w, r, 404, "not found", fmt.Sprintf("vm claim %s not found", vmcId))
			return
		} else {
			util.ReturnHTTPMessage(w, r, 500, "internalerror", fmt.Sprintf("error retrieving vm claim %s", vmcId))
			return
		}
	}

	if vmc.GetUserId() != user.GetId() {
		impersonatedUserId := user.GetId()
		authrResponse, err := rbac.AuthorizeSimple(r, vmcs.authrClient, impersonatedUserId, rbac.HobbyfarmPermission(resourcePlural, rbac.VerbGet))
		if err != nil || !authrResponse.Success {
			util.ReturnHTTPMessage(w, r, 403, "forbidden", "access denied to get vmclaim")
			return
		}
	}

	preparedVMC := PreparedVirtualMachineClaim{
		Id:                  vmcId,
		UserId:              vmc.GetUserId(),
		RestrictedBind:      vmc.GetRestrictedBind(),
		RestrictedBindValue: vmc.GetRestrictedBindValue(),
		VirtualMachines:     vmc.GetVms(),
		DynamicCapable:      vmc.GetDynamicCapable(),
		BaseName:            vmc.GetBaseName(),
		BindMode:            vmc.GetStatus().GetBindMode(),
		StaticBindAttempts:  vmc.GetStatus().GetStaticBindAttempts(),
		Bound:               vmc.GetStatus().GetBound(),
		Ready:               vmc.GetStatus().GetReady(),
		Tainted:             vmc.GetStatus().GetTainted(),
	}

	encodedVMC, err := json.Marshal(preparedVMC)
	if err != nil {
		glog.Error(err)
	}
	util.ReturnHTTPContent(w, r, 200, "success", encodedVMC)

	glog.V(2).Infof("retrieved vmc %s", vmcId)
}
