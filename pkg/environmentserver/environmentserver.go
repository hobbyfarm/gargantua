package environmentserver

import (
	"crypto/sha256"
	"encoding/base32"
	"encoding/json"
	"fmt"
	"github.com/golang/glog"
	"github.com/gorilla/mux"
	hfv1 "github.com/hobbyfarm/gargantua/pkg/apis/hobbyfarm.io/v1"
	"github.com/hobbyfarm/gargantua/pkg/authclient"
	hfClientset "github.com/hobbyfarm/gargantua/pkg/client/clientset/versioned"
	"github.com/hobbyfarm/gargantua/pkg/util"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/util/retry"
	"net/http"
	"strconv"
	"strings"
	"time"
)

type EnvironmentServer struct {
	auth        *authclient.AuthClient
	hfClientSet *hfClientset.Clientset
}

func NewEnvironmentServer(authClient *authclient.AuthClient, hfClientset *hfClientset.Clientset) (*EnvironmentServer, error) {
	es := EnvironmentServer{}

	es.hfClientSet = hfClientset
	es.auth = authClient

	return &es, nil
}

func (e EnvironmentServer) getEnvironment(id string) (hfv1.Environment, error) {

	empty := hfv1.Environment{}

	if len(id) == 0 {
		return empty, fmt.Errorf("vm claim id passed in was empty")
	}

	obj, err := e.hfClientSet.HobbyfarmV1().Environments().Get(id, metav1.GetOptions{})
	if err != nil {
		return empty, fmt.Errorf("error while retrieving Environment by id: %s with error: %v", id, err)
	}

	return *obj, nil

}

func (e EnvironmentServer) SetupRoutes(r *mux.Router) {
	r.HandleFunc("/e/environment/list", e.ListFunc).Methods("GET")
	r.HandleFunc("/e/environment/{id}", e.GetFunc).Methods("GET")
	r.HandleFunc("/e/environment/create", e.CreateFunc).Methods("POST")
	r.HandleFunc("/e/environment/{id}/update", e.UpdateFunc).Methods("PUT")
	r.HandleFunc("/e/environment/{environment_id}/available", e.PostEnvironmentAvailableFunc).Methods("POST")
	glog.V(2).Infof("set up routes for environment server")
}

type PreparedEnvironment struct {
	Name string `json:"name"`
	hfv1.EnvironmentSpec
	hfv1.EnvironmentStatus
}

func (e EnvironmentServer) GetFunc(w http.ResponseWriter, r *http.Request) {
	_, err := e.auth.AuthNAdmin(w, r)
	if err != nil {
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

	preparedEnvironment := PreparedEnvironment{environment.Name,environment.Spec, environment.Status}

	encodedEnvironment, err := json.Marshal(preparedEnvironment)
	if err != nil {
		glog.Error(err)
	}
	util.ReturnHTTPContent(w, r, 200, "success", encodedEnvironment)

	glog.V(2).Infof("retrieved environment %s", environment.Name)
}

func (e EnvironmentServer) ListFunc(w http.ResponseWriter, r *http.Request) {
	_, err := e.auth.AuthNAdmin(w, r)
	if err != nil {
		util.ReturnHTTPMessage(w, r, 403, "forbidden", "no access to list environments")
		return
	}

	environments, err := e.hfClientSet.HobbyfarmV1().Environments().List(metav1.ListOptions{})

	if err != nil {
		glog.Errorf("error while listing all environments %v", err)
		util.ReturnHTTPMessage(w, r, 500, "error", "error listing all environments")
		return
	}

	preparedEnvironments := []PreparedEnvironment{} // must be declared this way so as to JSON marshal into [] instead of null

	for _, e := range environments.Items {
		preparedEnvironments = append(preparedEnvironments, PreparedEnvironment{e.Name,e.Spec, e.Status})
	}

	encodedEnvironments, err := json.Marshal(preparedEnvironments)
	if err != nil {
		glog.Error(err)
	}
	util.ReturnHTTPContent(w, r, 200, "success", encodedEnvironments)

	glog.V(2).Infof("retrieved list of all environments")
}

func (e EnvironmentServer) CreateFunc(w http.ResponseWriter, r *http.Request) {
	_, err := e.auth.AuthNAdmin(w, r)
	if err != nil {
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

	capacityMode := r.PostFormValue("capacity_mode")
	if capacityMode == "" {
		util.ReturnHTTPMessage(w, r, 400, "badrequest", "no capacity_mode passed in")
		return
	} else if capacityMode != "raw" && capacityMode != "count" {
		// invalid capacity mode passed in
		util.ReturnHTTPMessage(w, r, 400, "badrequest", "invalid capacity_mode passed in")
		return
	}

	burstCapable := r.PostFormValue("burst_capable")
	if burstCapable == "" {
		util.ReturnHTTPMessage(w, r, 400, "badrequest", "no burst_capable passed in")
		return
	}
	burstCapableBool, err := strconv.ParseBool(burstCapable)
	if err != nil {
		util.ReturnHTTPMessage(w, r, 400, "badrequest", "invalid burst_capacity passed in")
		return
	}

	templateMappingUnmarshaled := map[string]map[string]string{} // lol
	err = json.Unmarshal([]byte(templateMapping), &templateMappingUnmarshaled)
	if err != nil {
		glog.Errorf("error while unmarshaling template_mapping (create environment) %v", err)
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

	if capacityMode == "raw" {
		environment.Spec.CapacityMode = hfv1.CapacityModeRaw
	} else {
		// not validating "count" here as we already validated input var above
		environment.Spec.CapacityMode = hfv1.CapacityModeCount
	}

	environment.Spec.BurstCapable = burstCapableBool

	environment, err = e.hfClientSet.HobbyfarmV1().Environments().Create(environment)
	if err != nil {
		glog.Errorf("error creating environment %v", err)
		util.ReturnHTTPMessage(w, r, 500, "internalerror", "error creating environment")
		return
	}

	util.ReturnHTTPMessage(w, r, 201, "created", environment.Name)
	return
}

func (e EnvironmentServer) UpdateFunc(w http.ResponseWriter, r *http.Request) {
	_, err := e.auth.AuthNAdmin(w, r)
	if err != nil {
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
		capacityMode := r.PostFormValue("capacity_mode")
		burstCapable := r.PostFormValue("burst_capable")

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

		if len(ipTranslationMap) > 0 {
			ipTranslationUnmarshaled := map[string]string{}
			err = json.Unmarshal([]byte(ipTranslationMap), &ipTranslationUnmarshaled)
			if err != nil {
				glog.Errorf("error while unmarshaling ip_translation_map (create environment) %v", err)
				util.ReturnHTTPMessage(w, r, 500, "internalerror", "error parsing")
				return fmt.Errorf("bad")
			}
			environment.Spec.IPTranslationMap = ipTranslationUnmarshaled
		}

		if len(wsEndpoint) > 0 {
			environment.Spec.WsEndpoint = wsEndpoint
		}

		if len(capacityMode) > 0 {
			if capacityMode == "raw" {
				environment.Spec.CapacityMode = hfv1.CapacityModeRaw
			} else if capacityMode == "count" {
				environment.Spec.CapacityMode = hfv1.CapacityModeCount
			} else {
				util.ReturnHTTPMessage(w, r, 400, "badrequest", "invalid capacity_mode passed in")
				return fmt.Errorf("bad")
			}
		}

		if len(burstCapable) > 0 {
			burstCapableBool, err := strconv.ParseBool(burstCapable)
			if err != nil {
				util.ReturnHTTPMessage(w, r, 400, "badrequest", "invalid burst_capable passed in")
				return fmt.Errorf("bad")
			}
			environment.Spec.BurstCapable = burstCapableBool
		}

		_, updateErr := e.hfClientSet.HobbyfarmV1().Environments().Update(&environment)
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
	_, err := e.auth.AuthNAdmin(w, r)
	if err != nil {
		util.ReturnHTTPMessage(w, r, 403, "forbidden", "no access to get environment")
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
	max, err := util.MaxAvailableDuringPeriod(e.hfClientSet, environmentId, start, end)
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
