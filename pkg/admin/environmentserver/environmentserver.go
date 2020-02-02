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

type AdminEnvironmentServer struct {
	auth        *authclient.AuthClient
	hfClientSet *hfClientset.Clientset
}

func NewAdminEnvironmentServer(authClient *authclient.AuthClient, hfClientset *hfClientset.Clientset) (*AdminEnvironmentServer, error) {
	es := AdminEnvironmentServer{}

	es.hfClientSet = hfClientset
	es.auth = authClient

	return &es, nil
}

func (a AdminEnvironmentServer) getEnvironment(id string) (hfv1.Environment, error) {

	empty := hfv1.Environment{}

	if len(id) == 0 {
		return empty, fmt.Errorf("vm claim id passed in was empty")
	}

	obj, err := a.hfClientSet.HobbyfarmV1().Environments().Get(id, metav1.GetOptions{})
	if err != nil {
		return empty, fmt.Errorf("error while retrieving Environment by id: %s with error: %v", id, err)
	}

	return *obj, nil

}

func (a AdminEnvironmentServer) SetupRoutes(r *mux.Router) {
	r.HandleFunc("/a/environment/list", a.ListFunc).Methods("GET")
	r.HandleFunc("/a/environment/{id}", a.GetFunc).Methods("GET")
	r.HandleFunc("/a/environment/create", a.CreateFunc).Methods("POST")
	r.HandleFunc("/a/environment/{id}/update", a.UpdateFunc).Methods("PUT")
	r.HandleFunc("/a/environment/{environment_id}/available", a.PostEnvironmentAvailableFunc).Methods("POST")
	glog.V(2).Infof("set up routes for environment server")
}

type PreparedEnvironment struct {
	Name string `json:"name"`
	hfv1.EnvironmentSpec
	hfv1.EnvironmentStatus
}

func (a AdminEnvironmentServer) GetFunc(w http.ResponseWriter, r *http.Request) {
	_, err := a.auth.AuthNAdmin(w, r)
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

	environment, err := a.getEnvironment(environmentId)

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

func (a AdminEnvironmentServer) ListFunc(w http.ResponseWriter, r *http.Request) {
	_, err := a.auth.AuthNAdmin(w, r)
	if err != nil {
		util.ReturnHTTPMessage(w, r, 403, "forbidden", "no access to list environments")
		return
	}

	environments, err := a.hfClientSet.HobbyfarmV1().Environments().List(metav1.ListOptions{})

	if err != nil {
		glog.Errorf("error while listing all environments %v", err)
		util.ReturnHTTPMessage(w, r, 500, "error", "error listing all environments")
		return
	}

	preparedEnvironments := []PreparedEnvironment{}

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

func (a AdminEnvironmentServer) CreateFunc(w http.ResponseWriter, r *http.Request) {
	_, err := a.auth.AuthNAdmin(w, r)
	if err != nil {
		util.ReturnHTTPMessage(w, r, 403, "forbidden", "no access to create environments")
		return
	}

	display_name := r.PostFormValue("display_name")
	if display_name == "" {
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

	template_mapping := r.PostFormValue("template_mapping")
	if template_mapping == "" {
		util.ReturnHTTPMessage(w, r, 400, "badrequest", "no template_mapping passed in")
		return
	}

	environment_specifics := r.PostFormValue("environment_specifics")
	if environment_specifics == "" {
		util.ReturnHTTPMessage(w, r, 400, "badrequest", "no environment_specifics passed in")
		return
	}

	ip_translation_map := r.PostFormValue("ip_translation_map")
	if ip_translation_map == "" {
		util.ReturnHTTPMessage(w, r, 400, "badrequest", "no ip_translation_map passed in")
		return
	}

	ws_endpoint := r.PostFormValue("ws_endpoint")
	if ws_endpoint == "" {
		util.ReturnHTTPMessage(w, r, 400, "badrequest", "no ws_endpoint passed in")
		return
	}

	capacity_mode := r.PostFormValue("capacity_mode")
	if capacity_mode == "" {
		util.ReturnHTTPMessage(w, r, 400, "badrequest", "no capacity_mode passed in")
		return
	} else if capacity_mode != "raw" && capacity_mode != "count" {
		// invalid capacity mode passed in
		util.ReturnHTTPMessage(w, r, 400, "badrequest", "invalid capacity_mode passed in")
		return
	}

	burst_capable := r.PostFormValue("burst_capable")
	if burst_capable == "" {
		util.ReturnHTTPMessage(w, r, 400, "badrequest", "no burst_capable passed in")
		return
	}
	burstCapableBool, err := strconv.ParseBool(burst_capable)
	if err != nil {
		util.ReturnHTTPMessage(w, r, 400, "badrequest", "invalid burst_capacity passed in")
		return
	}

	templateMappingUnmarshaled := map[string]map[string]string{} // lol
	err = json.Unmarshal([]byte(template_mapping), &templateMappingUnmarshaled)
	if err != nil {
		glog.Errorf("error while unmarshaling template_mapping (create environment) %v", err)
		util.ReturnHTTPMessage(w, r, 500, "internalerror", "error parsing")
		return
	}

	environmentSpecificsUnmarshaled := map[string]string{}
	err = json.Unmarshal([]byte(environment_specifics), &environmentSpecificsUnmarshaled)
	if err != nil {
		glog.Errorf("error while unmarshaling environment_specifics (create environment) %v", err)
		util.ReturnHTTPMessage(w, r, 500, "internalerror", "error parsing")
		return
	}

	ipTranslationUnmarshaled := map[string]string{}
	err = json.Unmarshal([]byte(ip_translation_map), &ipTranslationUnmarshaled)
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

	environment.Spec.DisplayName = display_name
	environment.Spec.DNSSuffix = dnssuffix
	environment.Spec.Provider = provider
	environment.Spec.TemplateMapping = templateMappingUnmarshaled
	environment.Spec.EnvironmentSpecifics = environmentSpecificsUnmarshaled
	environment.Spec.IPTranslationMap = ipTranslationUnmarshaled
	environment.Spec.WsEndpoint = ws_endpoint

	if capacity_mode == "raw" {
		environment.Spec.CapacityMode = hfv1.CapacityModeRaw
	} else {
		// not validating "count" here as we already validated input var above
		environment.Spec.CapacityMode = hfv1.CapacityModeCount
	}

	environment.Spec.BurstCapable = burstCapableBool

	environment, err = a.hfClientSet.HobbyfarmV1().Environments().Create(environment)
	if err != nil {
		glog.Errorf("error creating environment %v", err)
		util.ReturnHTTPMessage(w, r, 500, "internalerror", "error creating environment")
		return
	}

	util.ReturnHTTPMessage(w, r, 201, "created", environment.Name)
	return
}

func (a AdminEnvironmentServer) UpdateFunc(w http.ResponseWriter, r *http.Request) {
	_, err := a.auth.AuthNAdmin(w, r)
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
		environment, err := a.getEnvironment(environmentId)
		if err != nil {
			glog.Errorf("error while retrieving environment %v", err)
			util.ReturnHTTPMessage(w, r, 500, "error", "no environment found")
			return fmt.Errorf("bad")
		}

		display_name := r.PostFormValue("display_name")
		dnssuffix := r.PostFormValue("dnssuffix")
		provider := r.PostFormValue("provider")
		template_mapping := r.PostFormValue("template_mapping")
		environment_specifics := r.PostFormValue("environment_specifics")
		ip_translation_map := r.PostFormValue("ip_translation_map")
		ws_endpoint := r.PostFormValue("ws_endpoint")
		capacity_mode := r.PostFormValue("capacity_mode")
		burst_capable := r.PostFormValue("burst_capable")

		if len(display_name) > 0 {
			environment.Spec.DisplayName = display_name
		}

		// empty string is a valid dnssuffix value (because it is optional), so not
		// performing string length check here
		environment.Spec.DNSSuffix = dnssuffix

		if len(provider) > 0 {
			environment.Spec.Provider = provider
		}

		if len(template_mapping) > 0 {
			templateMappingUnmarshaled := map[string]map[string]string{} // lol
			err = json.Unmarshal([]byte(template_mapping), &templateMappingUnmarshaled)
			if err != nil {
				glog.Errorf("error while unmarshaling template_mapping (update environment) %v", err)
				util.ReturnHTTPMessage(w, r, 500, "internalerror", "error parsing")
				return fmt.Errorf("bad")
			}
			environment.Spec.TemplateMapping = templateMappingUnmarshaled
		}

		if len(environment_specifics) > 0 {
			environmentSpecificsUnmarshaled := map[string]string{}
			err = json.Unmarshal([]byte(environment_specifics), &environmentSpecificsUnmarshaled)
			if err != nil {
				glog.Errorf("error while unmarshaling environment_specifics (update environment) %v", err)
				util.ReturnHTTPMessage(w, r, 500, "internalerror", "error parsing")
				return fmt.Errorf("bad")
			}
			environment.Spec.EnvironmentSpecifics = environmentSpecificsUnmarshaled
		}

		if len(ip_translation_map) > 0 {
			ipTranslationUnmarshaled := map[string]string{}
			err = json.Unmarshal([]byte(ip_translation_map), &ipTranslationUnmarshaled)
			if err != nil {
				glog.Errorf("error while unmarshaling ip_translation_map (create environment) %v", err)
				util.ReturnHTTPMessage(w, r, 500, "internalerror", "error parsing")
				return fmt.Errorf("bad")
			}
			environment.Spec.IPTranslationMap = ipTranslationUnmarshaled
		}

		if len(ws_endpoint) > 0 {
			environment.Spec.WsEndpoint = ws_endpoint
		}

		if len(capacity_mode) > 0 {
			if capacity_mode == "raw" {
				environment.Spec.CapacityMode = hfv1.CapacityModeRaw
			} else if capacity_mode == "count" {
				environment.Spec.CapacityMode = hfv1.CapacityModeCount
			} else {
				util.ReturnHTTPMessage(w, r, 400, "badrequest", "invalid capacity_mode passed in")
				return fmt.Errorf("bad")
			}
		}

		if len(burst_capable) > 0 {
			burstCapableBool, err := strconv.ParseBool(burst_capable)
			if err != nil {
				util.ReturnHTTPMessage(w, r, 400, "badrequest", "invalid burst_capable passed in")
				return fmt.Errorf("bad")
			}
			environment.Spec.BurstCapable = burstCapableBool
		}

		_, updateErr := a.hfClientSet.HobbyfarmV1().Environments().Update(&environment)
		return updateErr
	})

	if retryErr != nil {
		util.ReturnHTTPMessage(w, r, 500, "internalerror", "error attempting to update")
		return
	}

	util.ReturnHTTPMessage(w, r, 200, "updated", "")
	return
}

func (a AdminEnvironmentServer) PostEnvironmentAvailableFunc(w http.ResponseWriter, r *http.Request) {
	_, err := a.auth.AuthNAdmin(w, r)
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

	environment, err := a.getEnvironment(environmentId)

	if err != nil {
		glog.Errorf("error while retrieving environment %v", err)
		util.ReturnHTTPMessage(w, r, 500, "error", "no environment found")
		return
	}
	max, err := util.MaxAvailableDuringPeriod(a.hfClientSet, environmentId, start, end)
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
