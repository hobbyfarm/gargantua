package vmtemplateservice

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/hobbyfarm/gargantua/v3/pkg/rbac"
	"github.com/hobbyfarm/gargantua/v3/pkg/util"
	generalpb "github.com/hobbyfarm/gargantua/v3/protos/general"
	scheduledeventpb "github.com/hobbyfarm/gargantua/v3/protos/scheduledevent"
	vmtemplatepb "github.com/hobbyfarm/gargantua/v3/protos/vmtemplate"

	"github.com/golang/glog"
	"github.com/gorilla/mux"
	hferrors "github.com/hobbyfarm/gargantua/v3/pkg/errors"
	hflabels "github.com/hobbyfarm/gargantua/v3/pkg/labels"
)

const (
	resourcePlural = rbac.ResourcePluralVMTemplate
)

// Prepared struct for API endpoints which only need to provide vmt id, name and image
type PreparedVMTemplate struct {
	Id            string `json:"id"`
	Name          string `json:"name"`
	Image         string `json:"image"`
	CostBasePrice string `json:"cost_base_price,omitempty"`
	CostTimeUnit  string `json:"cost_time_unit,omitempty"`
}

// Prepared struct for API endpoints which additionally to the PreparedVMTemplate struct also need to provide config details
type PreparedVMTemplateWithConfig struct {
	PreparedVMTemplate
	ConfigMap map[string]string `json:"config_map"`
}

func (v VirtualMachineTemplateServer) GetFunc(w http.ResponseWriter, r *http.Request) {
	user, err := rbac.AuthenticateRequest(r, v.authnClient)
	if err != nil {
		util.ReturnHTTPMessage(w, r, 401, "unauthorized", "authentication failed")
		return
	}

	impersonatedUserId := user.GetId()
	authrResponse, err := rbac.AuthorizeSimple(r, v.authrClient, impersonatedUserId, rbac.HobbyfarmPermission(resourcePlural, rbac.VerbGet))
	if err != nil || !authrResponse.Success {
		util.ReturnHTTPMessage(w, r, 403, "forbidden", "no access to get vm template")
		return
	}

	vars := mux.Vars(r)

	vmtId := vars["id"]

	if len(vmtId) == 0 {
		util.ReturnHTTPMessage(w, r, 400, "bad request", "no vm template id passed in")
		return
	}

	vmt, err := v.internalVMTemplateServer.GetVMTemplate(r.Context(), &generalpb.GetRequest{Id: vmtId})
	if err != nil {
		glog.Errorf("error while retrieving virtual machine template: %s", hferrors.GetErrorMessage(err))
		if hferrors.IsGrpcNotFound(err) {
			errMsg := fmt.Sprintf("virtual machine template %s not found", vmtId)
			util.ReturnHTTPMessage(w, r, http.StatusNotFound, "not found", errMsg)
			return
		}
		errMsg := fmt.Sprintf("error retrieving virtual machine template %s", vmtId)
		util.ReturnHTTPMessage(w, r, http.StatusInternalServerError, "error", errMsg)
		return
	}

	preparedVmt := PreparedVMTemplateWithConfig{
		PreparedVMTemplate: PreparedVMTemplate{
			Id:            vmtId,
			Name:          vmt.GetName(),
			Image:         vmt.GetImage(),
			CostBasePrice: vmt.GetCostBasePrice(),
			CostTimeUnit:  vmt.GetCostTimeUnit(),
		},
		ConfigMap: vmt.GetConfigMap(),
	}

	encodedVmt, err := json.Marshal(preparedVmt)
	if err != nil {
		glog.Error(err)
	}
	util.ReturnHTTPContent(w, r, 200, "success", encodedVmt)

	glog.V(2).Infof("retrieved vmt %s", vmtId)
}

func (v VirtualMachineTemplateServer) ListFunc(w http.ResponseWriter, r *http.Request) {
	user, err := rbac.AuthenticateRequest(r, v.authnClient)
	if err != nil {
		util.ReturnHTTPMessage(w, r, 401, "unauthorized", "authentication failed")
		return
	}

	impersonatedUserId := user.GetId()
	authrResponse, err := rbac.AuthorizeSimple(r, v.authrClient, impersonatedUserId, rbac.HobbyfarmPermission(resourcePlural, rbac.VerbList))
	if err != nil || !authrResponse.Success {
		util.ReturnHTTPMessage(w, r, 403, "forbidden", "no access to list vmts")
		return
	}

	vmtList, err := v.internalVMTemplateServer.ListVMTemplate(r.Context(), &generalpb.ListOptions{})
	if err != nil {
		glog.Errorf("error while listing all vmts: %s", hferrors.GetErrorMessage(err))
		util.ReturnHTTPMessage(w, r, 500, "error", "error listing all vmts")
		return
	}

	preparedVirtualMachineTemplates := []PreparedVMTemplate{}

	for _, vmt := range vmtList.GetVmtemplates() {
		preparedVirtualMachineTemplates = append(preparedVirtualMachineTemplates, PreparedVMTemplate{
			Id:            vmt.GetId(),
			Name:          vmt.GetName(),
			Image:         vmt.GetImage(),
			CostBasePrice: vmt.GetCostBasePrice(),
			CostTimeUnit:  vmt.GetCostTimeUnit(),
		})
	}

	encodedVirtualMachineTemplates, err := json.Marshal(preparedVirtualMachineTemplates)
	if err != nil {
		glog.Error(err)
	}
	util.ReturnHTTPContent(w, r, 200, "success", encodedVirtualMachineTemplates)

	glog.V(2).Infof("retrieved list of all environments")
}

func (v VirtualMachineTemplateServer) CreateFunc(w http.ResponseWriter, r *http.Request) {
	user, err := rbac.AuthenticateRequest(r, v.authnClient)
	if err != nil {
		util.ReturnHTTPMessage(w, r, 401, "unauthorized", "authentication failed")
		return
	}

	impersonatedUserId := user.GetId()
	authrResponse, err := rbac.AuthorizeSimple(r, v.authrClient, impersonatedUserId, rbac.HobbyfarmPermission(resourcePlural, rbac.VerbCreate))
	if err != nil || !authrResponse.Success {
		util.ReturnHTTPMessage(w, r, 403, "forbidden", "no access to create vmt")
		return
	}

	name := r.PostFormValue("name")
	if name == "" {
		util.ReturnHTTPMessage(w, r, 400, "bad request", "missing name")
		return
	}

	image := r.PostFormValue("image")
	if image == "" {
		util.ReturnHTTPMessage(w, r, 400, "bad request", "missing image")
		return
	}

	configMapRaw := r.PostFormValue("config_map") // no validation, config_map not required

	costBasePrice, costTimeUnit, err := normalizeCost(
		r.PostFormValue("cost_base_price"),
		r.PostFormValue("cost_time_unit"))

	if err != nil {
		util.ReturnHTTPMessage(w, r, 400, "bad request", err.Error())
		return
	}

	glog.V(2).Infof("user %s is creating vmtemplate", user.GetId())

	vmTemplateId, err := v.internalVMTemplateServer.CreateVMTemplate(r.Context(), &vmtemplatepb.CreateVMTemplateRequest{
		Name:          name,
		Image:         image,
		ConfigMapRaw:  configMapRaw,
		CostBasePrice: costBasePrice,
		CostTimeUnit:  costTimeUnit,
	})
	if err != nil {
		glog.Errorf("error creating vmtemplate: %s", hferrors.GetErrorMessage(err))
		util.ReturnHTTPMessage(w, r, 500, "internalerror", "error creating vmtemplate")
		return
	}

	util.ReturnHTTPMessage(w, r, 201, "created", vmTemplateId.GetId())
}

func normalizeCost(costBasePrice, costTimeUnit string) (basePrice, timeUnit *string, err error) {
	if util.IsDefaultValue(costBasePrice) && util.IsDefaultValue(costTimeUnit) {
		return
	}

	// if one is set the other needs to be set as well
	if !util.IsDefaultValue(costBasePrice) && util.IsDefaultValue(costTimeUnit) {
		err = errors.New("missing cost_time_unit")
		return
	}
	if util.IsDefaultValue(costBasePrice) && !util.IsDefaultValue(costTimeUnit) {
		err = errors.New("missing cost_base_price")
		return
	}

	_, err = strconv.ParseFloat(costBasePrice, 64)
	if err != nil {
		err = errors.New("cost_base_price needs to be a float64")
		return
	}

	parsedTimeUnit, err := util.ParseTimeUnit(costTimeUnit)
	if err != nil {
		err = errors.New("invalid cost_time_unit")
		return
	}

	return util.RefOrNil(costBasePrice), util.RefOrNil(parsedTimeUnit), nil
}

func (v VirtualMachineTemplateServer) UpdateFunc(w http.ResponseWriter, r *http.Request) {
	user, err := rbac.AuthenticateRequest(r, v.authnClient)
	if err != nil {
		util.ReturnHTTPMessage(w, r, 401, "unauthorized", "authentication failed")
		return
	}

	impersonatedUserId := user.GetId()
	authrResponse, err := rbac.AuthorizeSimple(r, v.authrClient, impersonatedUserId, rbac.HobbyfarmPermission(resourcePlural, rbac.VerbUpdate))
	if err != nil || !authrResponse.Success {
		util.ReturnHTTPMessage(w, r, 403, "forbidden", "no access to update vmt")
		return
	}

	vars := mux.Vars(r)

	id := vars["id"]
	if id == "" {
		util.ReturnHTTPMessage(w, r, 400, "bad request", "no id passed in")
		return
	}

	glog.V(2).Infof("user %s updating vmtemplate %s", impersonatedUserId, id)

	name := r.PostFormValue("name")
	image := r.PostFormValue("image")
	configMapRaw := r.PostFormValue("config_map")

	costBasePrice, costTimeUnit, err := normalizeCost(
		r.PostFormValue("cost_base_price"),
		r.PostFormValue("cost_time_unit"))

	if err != nil {
		util.ReturnHTTPMessage(w, r, 400, "bad request", err.Error())
		return
	}

	_, err = v.internalVMTemplateServer.UpdateVMTemplate(r.Context(), &vmtemplatepb.UpdateVMTemplateRequest{
		Id:            id,
		Name:          name,
		Image:         image,
		ConfigMapRaw:  configMapRaw,
		CostBasePrice: costBasePrice,
		CostTimeUnit:  costTimeUnit,
	})

	if err != nil {
		util.ReturnHTTPMessage(w, r, 500, "error", "error attempting to update vmtemplate")
		return
	}

	util.ReturnHTTPMessage(w, r, 200, "updated", "")
}

func (v VirtualMachineTemplateServer) DeleteFunc(w http.ResponseWriter, r *http.Request) {
	// deleting a vmtemplate requires none of the following objects having reference to it
	// - future scheduled events
	// - virtualmachines
	// - virtualmachineclaims
	// - virtualmachinesets
	user, err := rbac.AuthenticateRequest(r, v.authnClient)
	if err != nil {
		util.ReturnHTTPMessage(w, r, 401, "unauthorized", "authentication failed")
		return
	}

	impersonatedUserId := user.GetId()
	authrResponse, err := rbac.AuthorizeSimple(r, v.authrClient, impersonatedUserId, rbac.HobbyfarmPermission(resourcePlural, rbac.VerbDelete))
	if err != nil || !authrResponse.Success {
		util.ReturnHTTPMessage(w, r, 403, "forbidden", "no access to delete vmt")
		return
	}

	// first, check if the vmt exists
	vars := mux.Vars(r)
	vmtId := vars["id"]
	if vmtId == "" {
		util.ReturnHTTPMessage(w, r, 400, "badrequest", "no id passed in")
		return
	}

	glog.V(2).Infof("user %s deleting vmtemplate %s", user.GetId(), vmtId)

	// first check if the vmt actually exists
	_, err = v.internalVMTemplateServer.GetVMTemplate(r.Context(), &generalpb.GetRequest{Id: vmtId})
	if err != nil {
		glog.Errorf("error while retrieving virtual machine template: %s", hferrors.GetErrorMessage(err))
		if hferrors.IsGrpcNotFound(err) {
			errMsg := fmt.Sprintf("error retrieving vmt while attempting vmt deletion: vmt %s not found", vmtId)
			util.ReturnHTTPMessage(w, r, http.StatusNotFound, "not found", errMsg)
			return
		}
		errMsg := fmt.Sprintf("error retrieving vmt %s while attempting vmt deletion", vmtId)
		util.ReturnHTTPMessage(w, r, http.StatusInternalServerError, "error", errMsg)
		return
	}

	// vmt exists, now we need to check all other objects for references
	// start with vmList
	vmList, err := v.vmClient.ListVM(r.Context(), &generalpb.ListOptions{
		LabelSelector: fmt.Sprintf("%s=%s", hflabels.VirtualMachineTemplate, vmtId),
	})
	if err != nil {
		util.ReturnHTTPMessage(w, r, 500, "internalerror", "error listing virtual machines while attempting vmt deletion")
		return
	}

	if len(vmList.GetVms()) > 0 {
		util.ReturnHTTPMessage(w, r, 409, "conflict", "existing virtual machines reference this vmtemplate")
		return
	}

	// now check scheduledevents
	scheduledEventList, err := v.scheduledEventClient.ListScheduledEvent(r.Context(), &generalpb.ListOptions{})
	if err != nil {
		util.ReturnHTTPMessage(w, r, 500, "internalerror",
			"error listing scheduled events while attempting vmt deletion")
		return
	}

	scheduledEvents := scheduledEventList.GetScheduledevents()
	if len(scheduledEvents) > 0 {
		for _, se := range scheduledEvents {
			if !se.GetStatus().GetFinished() {
				// unfinished SE. Is it going on now or in the future?
				startTime, err := time.Parse(time.UnixDate, se.GetStartTime())
				if err != nil {
					util.ReturnHTTPMessage(w, r, 500, "internalerror",
						"error parsing time while checking scheduledevent for conflict")
					return
				}
				endTime, err := time.Parse(time.UnixDate, se.GetEndTime())
				if err != nil {
					util.ReturnHTTPMessage(w, r, 500, "internalerror",
						"error parsing time while checking scheduledevent for conflict")
					return
				}

				// if this starts in the future, or hasn't ended
				if startTime.After(time.Now()) || endTime.After(time.Now()) {
					// check for template existence
					if exists := searchForTemplateInRequiredVMs(se.GetRequiredVms(), vmtId); exists {
						// if template exists in this to-be-happening SE, we can't delete it
						util.ReturnHTTPMessage(w, r, 409, "conflict",
							"existing or future scheduled event references this vmtemplate")
					}
				}
			}
		}
	}

	// now check virtual machine claims
	vmcList, err := v.vmClaimClient.ListVMClaim(r.Context(), &generalpb.ListOptions{
		LabelSelector: fmt.Sprintf("virtualmachinetemplate.hobbyfarm.io/%s=%s", vmtId, "true"),
	})
	if err != nil {
		util.ReturnHTTPMessage(w, r, 500, "internalerror",
			"error listing virtual machine claims while attempting vmt deletion")
		return
	}

	if len(vmcList.GetVmclaims()) > 0 {
		util.ReturnHTTPMessage(w, r, 409, "conflict",
			"existing virtual machine claims reference this vmtemplate")
		return
	}

	// now check virtualmachinesets (theoretically the VM checks above should catch this, but let's be safe)
	vmsetList, err := v.vmSetClient.ListVMSet(r.Context(), &generalpb.ListOptions{
		LabelSelector: fmt.Sprintf("virtualmachinetemplate.hobbyfarm.io/%s=%s", vmtId, "true"),
	})
	if err != nil {
		util.ReturnHTTPMessage(w, r, 500, "internalerror",
			"error listing virtual machine sets while attempting vmt deletion")
		return
	}

	if len(vmsetList.GetVmsets()) > 0 {
		util.ReturnHTTPMessage(w, r, 409, "conflict",
			"existing virtual machine sets reference this vmtemplate")
		return
	}

	// if we get here, shouldn't be anything in our path stopping us from deleting the vmtemplate
	// so do it!
	_, err = v.internalVMTemplateServer.DeleteVMTemplate(r.Context(), &generalpb.ResourceId{Id: vmtId})
	if err != nil {
		glog.Errorf("error deleting vmtemplate: %s", hferrors.GetErrorMessage(err))
		util.ReturnHTTPMessage(w, r, 500, "internalerror", "error deleting vmtemplate")
		return
	}

	util.ReturnHTTPMessage(w, r, 200, "deleted", "vmtemplate deleted")
}

func searchForTemplateInRequiredVMs(req map[string]*scheduledeventpb.VMTemplateCountMap, template string) bool {
	for _, v := range req {
		// k is environment, v is map[string]string
		for kk := range v.GetVmTemplateCounts() {
			// kk is vmtemplate, vv is count
			if kk == template {
				return true
			}
		}
	}
	return false
}
