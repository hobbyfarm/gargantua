package environmentserver

import (
	"context"
	"crypto/sha256"
	"encoding/base32"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/golang/glog"
	"github.com/gorilla/mux"
	hfv1 "github.com/hobbyfarm/gargantua/v3/pkg/apis/hobbyfarm.io/v1"
	hfClientset "github.com/hobbyfarm/gargantua/v3/pkg/client/clientset/versioned"
	"github.com/hobbyfarm/gargantua/v3/pkg/rbac"
	"github.com/hobbyfarm/gargantua/v3/pkg/util"
	"github.com/hobbyfarm/gargantua/v3/protos/authn"
	"github.com/hobbyfarm/gargantua/v3/protos/authr"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/util/retry"
)

const (
	resourcePlural = rbac.ResourcePluralEnvironment
)

type EnvironmentServer struct {
	authnClient authn.AuthNClient
	authrClient authr.AuthRClient
	hfClientSet hfClientset.Interface
	ctx         context.Context
}

func NewEnvironmentServer(authnClient authn.AuthNClient, authrClient authr.AuthRClient, hfClientset hfClientset.Interface, ctx context.Context) (*EnvironmentServer, error) {
	es := EnvironmentServer{}

	es.hfClientSet = hfClientset
	es.authnClient = authnClient
	es.authrClient = authrClient
	es.ctx = ctx

	return &es, nil
}

func (e EnvironmentServer) getEnvironment(id string) (hfv1.Environment, error) {

	empty := hfv1.Environment{}

	if len(id) == 0 {
		return empty, fmt.Errorf("vm claim id passed in was empty")
	}

	obj, err := e.hfClientSet.HobbyfarmV1().Environments(util.GetReleaseNamespace()).Get(e.ctx, id, metav1.GetOptions{})
	if err != nil {
		return empty, fmt.Errorf("error while retrieving Environment by id: %s with error: %v", id, err)
	}

	return *obj, nil

}

func (e EnvironmentServer) SetupRoutes(r *mux.Router) {
	r.HandleFunc("/a/environment/list", e.ListFunc).Methods("GET")
	r.HandleFunc("/a/environment/{id}", e.GetFunc).Methods("GET")
	r.HandleFunc("/a/environment/create", e.CreateFunc).Methods("POST")
	r.HandleFunc("/a/environment/{id}/update", e.UpdateFunc).Methods("PUT")
	r.HandleFunc("/a/environment/{environment_id}/available", e.PostEnvironmentAvailableFunc).Methods("POST")
	glog.V(2).Infof("set up routes for environment server")
}

type PreparedEnvironment struct {
	Name string `json:"name"`
	hfv1.EnvironmentSpec
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

	environment, err := e.getEnvironment(environmentId)

	if err != nil {
		glog.Errorf("error while retrieving environment %v", err)
		util.ReturnHTTPMessage(w, r, 500, "error", "no environment found")
		return
	}

	preparedEnvironment := PreparedEnvironment{environment.Name, environment.Spec}

	encodedEnvironment, err := json.Marshal(preparedEnvironment)
	if err != nil {
		glog.Error(err)
	}
	util.ReturnHTTPContent(w, r, 200, "success", encodedEnvironment)

	glog.V(2).Infof("retrieved environment %s", environment.Name)
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

	environments, err := e.hfClientSet.HobbyfarmV1().Environments(util.GetReleaseNamespace()).List(e.ctx, metav1.ListOptions{})

	if err != nil {
		glog.Errorf("error while listing all environments %v", err)
		util.ReturnHTTPMessage(w, r, 500, "error", "error listing all environments")
		return
	}

	preparedEnvironments := []PreparedListEnvironment{} // must be declared this way so as to JSON marshal into [] instead of null

	for _, e := range environments.Items {
		keys := make(map[string]map[string]string)
		for k, _ := range e.Spec.TemplateMapping {
			keys[k] = map[string]string{}
		}
		preparedEnvironments = append(preparedEnvironments, PreparedListEnvironment{e.Name, e.Spec.DisplayName, e.Spec.Provider, keys})
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

	templateMappingUnmarshaled := map[string]map[string]string{} // lol
	err = json.Unmarshal([]byte(templateMapping), &templateMappingUnmarshaled)
	if err != nil {
		glog.Errorf("error while unmarshaling template_mapping (create environment) %v", err)
		util.ReturnHTTPMessage(w, r, 500, "internalerror", "error parsing")
		return
	}

	countCapacityUnmarshaled := map[string]int{}
	err = json.Unmarshal([]byte(countCapacity), &countCapacityUnmarshaled)
	if err != nil {
		glog.Errorf("error while unmarshaling count_capacity (create environment) %v", err)
		util.ReturnHTTPMessage(w, r, 500, "internalerror", "error parsing")
		return
	}

	environmentSpecificsUnmarshaled := map[string]string{}
	err = json.Unmarshal([]byte(environmentSpecifics), &environmentSpecificsUnmarshaled)
	if err != nil {
		glog.Errorf("error while unmarshaling environment_specifics (create environment) %v", err)
		util.ReturnHTTPMessage(w, r, 500, "internalerror", "error parsing")
		return
	}

	ipTranslationUnmarshaled := map[string]string{}
	err = json.Unmarshal([]byte(ipTranslationMap), &ipTranslationUnmarshaled)
	if err != nil {
		glog.Errorf("error while unmarshaling ip_translation_map (create environment) %v", err)
		util.ReturnHTTPMessage(w, r, 500, "internalerror", "error parsing")
		return
	}

	environment := &hfv1.Environment{}
	hasher := sha256.New()
	hasher.Write([]byte(time.Now().String())) // generate random name
	sha := base32.StdEncoding.WithPadding(-1).EncodeToString(hasher.Sum(nil))[:10]
	environment.Name = "env-" + strings.ToLower(sha)

	environment.Spec.DisplayName = displayName
	environment.Spec.DNSSuffix = dnssuffix
	environment.Spec.Provider = provider
	environment.Spec.TemplateMapping = templateMappingUnmarshaled
	environment.Spec.EnvironmentSpecifics = environmentSpecificsUnmarshaled
	environment.Spec.IPTranslationMap = ipTranslationUnmarshaled
	environment.Spec.WsEndpoint = wsEndpoint
	environment.Spec.CountCapacity = countCapacityUnmarshaled

	environment, err = e.hfClientSet.HobbyfarmV1().Environments(util.GetReleaseNamespace()).Create(e.ctx, environment, metav1.CreateOptions{})
	if err != nil {
		glog.Errorf("error creating environment %v", err)
		util.ReturnHTTPMessage(w, r, 500, "internalerror", "error creating environment")
		return
	}

	util.ReturnHTTPMessage(w, r, 201, "created", environment.Name)
	return
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

	retryErr := retry.RetryOnConflict(retry.DefaultRetry, func() error {
		environment, err := e.getEnvironment(environmentId)
		if err != nil {
			glog.Errorf("error while retrieving environment %v", err)
			util.ReturnHTTPMessage(w, r, 500, "error", "no environment found")
			return fmt.Errorf("bad")
		}

		displayName := r.PostFormValue("display_name")
		dnssuffix := r.PostFormValue("dnssuffix")
		provider := r.PostFormValue("provider")
		templateMapping := r.PostFormValue("template_mapping")
		environmentSpecifics := r.PostFormValue("environment_specifics")
		ipTranslationMap := r.PostFormValue("ip_translation_map")
		wsEndpoint := r.PostFormValue("ws_endpoint")
		countCapacity := r.PostFormValue("count_capacity")

		if len(displayName) > 0 {
			environment.Spec.DisplayName = displayName
		}

		// empty string is e valid dnssuffix value (because it is optional), so not
		// performing string length check here
		environment.Spec.DNSSuffix = dnssuffix

		if len(provider) > 0 {
			environment.Spec.Provider = provider
		}

		if len(templateMapping) > 0 {
			templateMappingUnmarshaled := map[string]map[string]string{} // lol
			err = json.Unmarshal([]byte(templateMapping), &templateMappingUnmarshaled)
			if err != nil {
				glog.Errorf("error while unmarshaling template_mapping (update environment) %v", err)
				util.ReturnHTTPMessage(w, r, 500, "internalerror", "error parsing")
				return fmt.Errorf("bad")
			}
			environment.Spec.TemplateMapping = templateMappingUnmarshaled
		}

		if len(environmentSpecifics) > 0 {
			environmentSpecificsUnmarshaled := map[string]string{}
			err = json.Unmarshal([]byte(environmentSpecifics), &environmentSpecificsUnmarshaled)
			if err != nil {
				glog.Errorf("error while unmarshaling environment_specifics (update environment) %v", err)
				util.ReturnHTTPMessage(w, r, 500, "internalerror", "error parsing")
				return fmt.Errorf("bad")
			}
			environment.Spec.EnvironmentSpecifics = environmentSpecificsUnmarshaled
		}

		if len(countCapacity) > 0 {
			countCapacityUnmarshaled := map[string]int{}
			err = json.Unmarshal([]byte(countCapacity), &countCapacityUnmarshaled)
			if err != nil {
				glog.Errorf("error while unmarshaling count_capacity (update environment) %v", err)
				util.ReturnHTTPMessage(w, r, 500, "internalerror", "error parsing")
				return fmt.Errorf("bad")
			}
			environment.Spec.CountCapacity = countCapacityUnmarshaled
		}

		if len(ipTranslationMap) > 0 {
			ipTranslationUnmarshaled := map[string]string{}
			err = json.Unmarshal([]byte(ipTranslationMap), &ipTranslationUnmarshaled)
			if err != nil {
				glog.Errorf("error while unmarshaling ip_translation_map (update environment) %v", err)
				util.ReturnHTTPMessage(w, r, 500, "internalerror", "error parsing")
				return fmt.Errorf("bad")
			}
			environment.Spec.IPTranslationMap = ipTranslationUnmarshaled
		}

		if len(wsEndpoint) > 0 {
			environment.Spec.WsEndpoint = wsEndpoint
		}

		_, updateErr := e.hfClientSet.HobbyfarmV1().Environments(util.GetReleaseNamespace()).Update(e.ctx, &environment, metav1.UpdateOptions{})
		return updateErr
	})

	if retryErr != nil {
		util.ReturnHTTPMessage(w, r, 500, "internalerror", "error attempting to update")
		return
	}

	util.ReturnHTTPMessage(w, r, 200, "updated", "")
	return
}

func (e EnvironmentServer) PostEnvironmentAvailableFunc(w http.ResponseWriter, r *http.Request) {
	user, err := rbac.AuthenticateRequest(r, e.authnClient)
	if err != nil {
		util.ReturnHTTPMessage(w, r, 401, "unauthorized", "authentication failed")
		return
	}

	impersonatedUserId := user.GetId()
	authrResponse, err := rbac.Authorize(r, e.authrClient, impersonatedUserId, []*authr.Permission{
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

	environment, err := e.getEnvironment(environmentId)

	if err != nil {
		glog.Errorf("error while retrieving environment %v", err)
		util.ReturnHTTPMessage(w, r, 500, "error", "no environment found")
		return
	}
	max, err := util.MaxAvailableDuringPeriod(e.hfClientSet, environmentId, start, end, e.ctx)
	if err != nil {
		glog.Errorf("error while getting max available %v", err)
		util.ReturnHTTPMessage(w, r, 500, "error", "error getting max available vms for environment")
		return
	}

	encodedEnvironment, err := json.Marshal(max)
	if err != nil {
		glog.Error(err)
	}
	util.ReturnHTTPContent(w, r, 200, "success", encodedEnvironment)

	glog.V(2).Infof("retrieved max available in environment %s", environment.Name)
}
