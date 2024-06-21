package environmentservice

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	hferrors "github.com/hobbyfarm/gargantua/v3/pkg/errors"
	"github.com/hobbyfarm/gargantua/v3/pkg/rbac"
	"github.com/hobbyfarm/gargantua/v3/pkg/util"
	"google.golang.org/protobuf/types/known/wrapperspb"

	"github.com/golang/glog"
	"github.com/gorilla/mux"
	authrpb "github.com/hobbyfarm/gargantua/v3/protos/authr"
	environmentpb "github.com/hobbyfarm/gargantua/v3/protos/environment"
	generalpb "github.com/hobbyfarm/gargantua/v3/protos/general"
)

const (
	resourcePlural = rbac.ResourcePluralEnvironment
)

type PreparedAvailableCount struct {
	AvailableCount map[string]uint32 `json:"available_count"`
}

type PreparedEnvironment struct {
	Name                 string                       `json:"name"`
	DisplayName          string                       `json:"display_name"`
	DNSSuffix            string                       `json:"dnssuffix"`
	Provider             string                       `json:"provider"`         // aws,vsphere,azure,custom ;)
	TemplateMapping      map[string]map[string]string `json:"template_mapping"` //  lol
	EnvironmentSpecifics map[string]string            `json:"environment_specifics"`
	IPTranslationMap     map[string]string            `json:"ip_translation_map"`
	WsEndpoint           string                       `json:"ws_endpoint"`
	CountCapacity        map[string]uint32            `json:"count_capacity"`
}

type PreparedListEnvironment struct {
	Name            string                       `json:"name"`
	DisplayName     string                       `json:"display_name"`
	Provider        string                       `json:"provider"`
	TemplateMapping map[string]map[string]string `json:"template_mapping"`
}

func (e EnvironmentServer) GetFunc(w http.ResponseWriter, r *http.Request) {
	user, err := rbac.AuthenticateRequest(r, e.authnClient)
	if err != nil {
		util.ReturnHTTPMessage(w, r, 401, "unauthorized", "authentication failed")
		return
	}

	impersonatedUserId := user.GetId()
	authrResponse, err := rbac.AuthorizeSimple(r, e.authrClient, impersonatedUserId, rbac.HobbyfarmPermission(resourcePlural, rbac.VerbGet))
	if err != nil || !authrResponse.Success {
		util.ReturnHTTPMessage(w, r, 403, "forbidden", "no access to get environment")
		return
	}

	vars := mux.Vars(r)

	environmentId := vars["id"]

	if len(environmentId) == 0 {
		util.ReturnHTTPMessage(w, r, 500, "error", "no environment id passed in")
		return
	}

	environment, err := e.internalEnvironmentServer.GetEnvironment(r.Context(), &generalpb.GetRequest{Id: environmentId})
	if err != nil {
		glog.Errorf("error while retrieving environment: %s", hferrors.GetErrorMessage(err))
		if hferrors.IsGrpcNotFound(err) {
			returnMsg := fmt.Sprintf("environment with id %s not found", environmentId)
			util.ReturnHTTPMessage(w, r, 404, "error", returnMsg)
			return
		}
		util.ReturnHTTPMessage(w, r, 500, "error", "internal error while retrieving environment")
		return
	}

	preparedEnvironment := PreparedEnvironment{
		Name:                 environment.GetId(),
		DisplayName:          environment.GetDisplayName(),
		DNSSuffix:            environment.GetDnssuffix(),
		Provider:             environment.GetProvider(),
		TemplateMapping:      util.ConvertMapStruct(environment.GetTemplateMapping(), util.GetRawStringMap),
		EnvironmentSpecifics: environment.GetEnvironmentSpecifics(),
		IPTranslationMap:     environment.GetIpTranslationMap(),
		WsEndpoint:           environment.GetWsEndpoint(),
		CountCapacity:        environment.GetCountCapacity(),
	}

	encodedEnvironment, err := json.Marshal(preparedEnvironment)
	if err != nil {
		glog.Error(err)
	}
	util.ReturnHTTPContent(w, r, 200, "success", encodedEnvironment)

	glog.V(2).Infof("retrieved environment %s", environment.GetId())
}

func (e EnvironmentServer) ListFunc(w http.ResponseWriter, r *http.Request) {
	user, err := rbac.AuthenticateRequest(r, e.authnClient)
	if err != nil {
		util.ReturnHTTPMessage(w, r, 401, "unauthorized", "authentication failed")
		return
	}

	impersonatedUserId := user.GetId()
	authrResponse, err := rbac.AuthorizeSimple(r, e.authrClient, impersonatedUserId, rbac.HobbyfarmPermission(resourcePlural, rbac.VerbList))
	if err != nil || !authrResponse.Success {
		util.ReturnHTTPMessage(w, r, 403, "forbidden", "no access to list environments")
		return
	}

	environmentList, err := e.internalEnvironmentServer.ListEnvironment(r.Context(), &generalpb.ListOptions{})

	if err != nil {
		glog.Errorf("error while listing all environments: %s", hferrors.GetErrorMessage(err))
		util.ReturnHTTPMessage(w, r, 500, "error", "error listing all environments")
		return
	}

	preparedEnvironments := []PreparedListEnvironment{} // must be declared this way so as to JSON marshal into [] instead of null

	for _, e := range environmentList.GetEnvironments() {
		keys := make(map[string]map[string]string)
		for k := range util.ConvertMapStruct(e.GetTemplateMapping(), util.GetRawStringMap) {
			keys[k] = map[string]string{} // reset template mapping entries -> @TODO: Figure out why?
		}
		preparedEnvironments = append(preparedEnvironments, PreparedListEnvironment{e.GetId(), e.GetDisplayName(), e.GetProvider(), keys})
	}

	encodedEnvironments, err := json.Marshal(preparedEnvironments)
	if err != nil {
		glog.Error(err)
	}
	util.ReturnHTTPContent(w, r, 200, "success", encodedEnvironments)

	glog.V(2).Infof("retrieved list of all environments")
}

func (e EnvironmentServer) CreateFunc(w http.ResponseWriter, r *http.Request) {
	user, err := rbac.AuthenticateRequest(r, e.authnClient)
	if err != nil {
		util.ReturnHTTPMessage(w, r, 401, "unauthorized", "authentication failed")
		return
	}

	impersonatedUserId := user.GetId()
	authrResponse, err := rbac.AuthorizeSimple(r, e.authrClient, impersonatedUserId, rbac.HobbyfarmPermission(resourcePlural, rbac.VerbCreate))
	if err != nil || !authrResponse.Success {
		util.ReturnHTTPMessage(w, r, 403, "forbidden", "no access to create environments")
		return
	}

	displayName := r.PostFormValue("display_name")
	if displayName == "" {
		util.ReturnHTTPMessage(w, r, 400, "badrequest", "no display_name passed in")
		return
	}

	dnssuffix := r.PostFormValue("dnssuffix")
	// dnssuffix optional so no validation performed

	provider := r.PostFormValue("provider")
	if provider == "" {
		util.ReturnHTTPMessage(w, r, 400, "badrequest", "no provider passed in")
		return
	}

	templateMapping := r.PostFormValue("template_mapping")
	if templateMapping == "" {
		util.ReturnHTTPMessage(w, r, 400, "badrequest", "no template_mapping passed in")
		return
	}

	environmentSpecifics := r.PostFormValue("environment_specifics")
	if environmentSpecifics == "" {
		util.ReturnHTTPMessage(w, r, 400, "badrequest", "no environment_specifics passed in")
		return
	}

	countCapacity := r.PostFormValue("count_capacity")
	if environmentSpecifics == "" {
		util.ReturnHTTPMessage(w, r, 400, "badrequest", "no count_capacity passed in")
		return
	}

	ipTranslationMap := r.PostFormValue("ip_translation_map")
	if ipTranslationMap == "" {
		util.ReturnHTTPMessage(w, r, 400, "badrequest", "no ip_translation_map passed in")
		return
	}

	wsEndpoint := r.PostFormValue("ws_endpoint")
	if wsEndpoint == "" {
		util.ReturnHTTPMessage(w, r, 400, "badrequest", "no ws_endpoint passed in")
		return
	}

	environmentId, err := e.internalEnvironmentServer.CreateEnvironment(r.Context(), &environmentpb.CreateEnvironmentRequest{
		DisplayName:          displayName,
		Dnssuffix:            dnssuffix,
		Provider:             provider,
		TemplateMapping:      templateMapping,
		EnvironmentSpecifics: environmentSpecifics,
		IpTranslationMap:     ipTranslationMap,
		WsEndpoint:           wsEndpoint,
		CountCapacity:        countCapacity,
	})

	if err != nil {
		glog.Errorf("error creating environment: %s", hferrors.GetErrorMessage(err))
		util.ReturnHTTPMessage(w, r, 500, "internalerror", "error creating environment")
		return
	}

	util.ReturnHTTPMessage(w, r, 201, "created", environmentId.GetId())
}

func (e EnvironmentServer) UpdateFunc(w http.ResponseWriter, r *http.Request) {
	user, err := rbac.AuthenticateRequest(r, e.authnClient)
	if err != nil {
		util.ReturnHTTPMessage(w, r, 401, "unauthorized", "authentication failed")
		return
	}

	impersonatedUserId := user.GetId()
	authrResponse, err := rbac.AuthorizeSimple(r, e.authrClient, impersonatedUserId, rbac.HobbyfarmPermission(resourcePlural, rbac.VerbUpdate))
	if err != nil || !authrResponse.Success {
		util.ReturnHTTPMessage(w, r, 403, "forbidden", "no access to update environment")
		return
	}

	vars := mux.Vars(r)

	environmentId := vars["id"]
	if len(environmentId) == 0 {
		util.ReturnHTTPMessage(w, r, 400, "badrequest", "no environment id passed in")
		return
	}

	displayName := r.PostFormValue("display_name")
	dnssuffix := r.PostFormValue("dnssuffix")
	provider := r.PostFormValue("provider")
	templateMapping := r.PostFormValue("template_mapping")
	environmentSpecifics := r.PostFormValue("environment_specifics")
	ipTranslationMap := r.PostFormValue("ip_translation_map")
	wsEndpoint := r.PostFormValue("ws_endpoint")
	countCapacity := r.PostFormValue("count_capacity")

	_, err = e.internalEnvironmentServer.UpdateEnvironment(r.Context(), &environmentpb.UpdateEnvironmentRequest{
		Id:                   environmentId,
		DisplayName:          displayName,
		Dnssuffix:            wrapperspb.String(dnssuffix),
		Provider:             provider,
		TemplateMapping:      templateMapping,
		EnvironmentSpecifics: environmentSpecifics,
		IpTranslationMap:     ipTranslationMap,
		WsEndpoint:           wsEndpoint,
		CountCapacity:        countCapacity,
	})

	if err != nil {
		glog.Errorf("error while updating environment: %s", hferrors.GetErrorMessage(err))
		if hferrors.IsGrpcParsingError(err) {
			util.ReturnHTTPMessage(w, r, 500, "internalerror", "error parsing")
			return
		}
		util.ReturnHTTPMessage(w, r, 500, "internalerror", "error attempting to update")
		return
	}

	util.ReturnHTTPMessage(w, r, 200, "updated", "")
}

func (e EnvironmentServer) PostEnvironmentAvailableFunc(w http.ResponseWriter, r *http.Request) {
	user, err := rbac.AuthenticateRequest(r, e.authnClient)
	if err != nil {
		util.ReturnHTTPMessage(w, r, 401, "unauthorized", "authentication failed")
		return
	}

	impersonatedUserId := user.GetId()
	authrResponse, err := rbac.Authorize(r, e.authrClient, impersonatedUserId, []*authrpb.Permission{
		rbac.HobbyfarmPermission(resourcePlural, rbac.VerbList),
		rbac.HobbyfarmPermission(rbac.ResourcePluralVMTemplate, rbac.VerbList),
	}, rbac.OperatorAND)
	if err != nil || !authrResponse.Success {
		util.ReturnHTTPMessage(w, r, 403, "forbidden", "no access to list environments")
		return
	}

	vars := mux.Vars(r)

	start := r.PostFormValue("start")
	end := r.PostFormValue("end")
	if start == "" || end == "" {
		util.ReturnHTTPMessage(w, r, 400, "bad request", "start or end time not provided")
		return
	}

	environmentId := vars["environment_id"]

	if len(environmentId) == 0 {
		util.ReturnHTTPMessage(w, r, 500, "error", "no environment id passed in")
		return
	}

	environment, err := e.internalEnvironmentServer.GetEnvironment(r.Context(), &generalpb.GetRequest{Id: environmentId})
	if err != nil {
		glog.Errorf("error while retrieving environment: %s", hferrors.GetErrorMessage(err))
		if hferrors.IsGrpcNotFound(err) {
			returnMsg := fmt.Sprintf("environment with id %s not found", environmentId)
			util.ReturnHTTPMessage(w, r, 404, "error", returnMsg)
			return
		}
		util.ReturnHTTPMessage(w, r, 500, "error", "internal error while retrieving environment")
		return
	}

	if err != nil {
		glog.Errorf("error while retrieving environment %v", err)
		util.ReturnHTTPMessage(w, r, 500, "error", "no environment found")
		return
	}
	max, err := e.maxAvailableDuringPeriod(environmentId, start, end, r.Context())
	if err != nil {
		glog.Errorf("error while getting max available count: %v", err)
		util.ReturnHTTPMessage(w, r, 500, "error", "error getting max available vms for environment")
		return
	}

	encodedEnvironment, err := json.Marshal(max)
	if err != nil {
		glog.Error(err)
	}
	util.ReturnHTTPContent(w, r, 200, "success", encodedEnvironment)

	glog.V(2).Infof("retrieved max available count in environment %s", environment.GetId())
}

func (e EnvironmentServer) maxAvailableDuringPeriod(environment string, startString string, endString string, ctx context.Context) (PreparedAvailableCount, error) {
	_, maximumVirtualMachineCount, err := util.VirtualMachinesUsedDuringPeriod(e.scheduledEventClient, environment, startString, endString, ctx)

	if err != nil {
		return PreparedAvailableCount{}, err
	}

	envObj, err := e.internalEnvironmentServer.GetEnvironment(ctx, &generalpb.GetRequest{Id: environment})
	if err != nil {
		return PreparedAvailableCount{}, fmt.Errorf("error retrieving environment %v", err)
	}

	max := PreparedAvailableCount{}
	max.AvailableCount = make(map[string]uint32)
	for k, v := range envObj.GetCountCapacity() {
		max.AvailableCount[k] = v
	}
	for vmt, count := range maximumVirtualMachineCount {
		if vmtCap, ok := envObj.GetCountCapacity()[vmt]; ok {
			max.AvailableCount[vmt] = vmtCap - count
		} else {
			glog.Errorf("Error looking for maximum count capacity of virtual machine template %s", vmt)
			max.AvailableCount[vmt] = 0
		}
	}
	return max, nil
}
