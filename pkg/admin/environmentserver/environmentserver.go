package environmentserver

import (
	"encoding/json"
	"fmt"
	"github.com/golang/glog"
	"github.com/gorilla/mux"
	hfv1 "github.com/hobbyfarm/gargantua/pkg/apis/hobbyfarm.io/v1"
	"github.com/hobbyfarm/gargantua/pkg/authclient"
	hfClientset "github.com/hobbyfarm/gargantua/pkg/client/clientset/versioned"
	"github.com/hobbyfarm/gargantua/pkg/util"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"net/http"
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
	r.HandleFunc("/a/environment/{id}", a.GetFunc).Methods("GET")
	r.HandleFunc("/a/environment/list", a.ListFunc).Methods("GET")
	r.HandleFunc("/a/environment/{environment_id}/available", a.PostEnvironmentAvailableFunc).Methods("POST")
	glog.V(2).Infof("set up routes for environment server")
}

type PreparedEnvironment struct {
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

	preparedEnvironment := PreparedEnvironment{environment.Spec, environment.Status}

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
		preparedEnvironments = append(preparedEnvironments, PreparedEnvironment{e.Spec, e.Status})
	}

	encodedEnvironments, err := json.Marshal(preparedEnvironments)
	if err != nil {
		glog.Error(err)
	}
	util.ReturnHTTPContent(w, r, 200, "success", encodedEnvironments)

	glog.V(2).Infof("retrieved list of all environments")
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
