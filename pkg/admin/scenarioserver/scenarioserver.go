package scenarioserver

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
	"strings"
)

type AdminScenarioServer struct {
	auth        *authclient.AuthClient
	hfClientSet *hfClientset.Clientset
}

func NewAdminScenarioServer(authClient *authclient.AuthClient, hfClientset *hfClientset.Clientset) (*AdminScenarioServer, error) {
	s := AdminScenarioServer{}

	s.hfClientSet = hfClientset
	s.auth = authClient

	return &s, nil
}

func (a AdminScenarioServer) getScenario(id string) (hfv1.Scenario, error) {

	empty := hfv1.Scenario{}

	if len(id) == 0 {
		return empty, fmt.Errorf("scenario id passed in was empty")
	}

	obj, err := a.hfClientSet.HobbyfarmV1().Scenarios().Get(id, metav1.GetOptions{})
	if err != nil {
		return empty, fmt.Errorf("error while retrieving Scenario by id: %s with error: %v", id, err)
	}

	return *obj, nil

}

func (a AdminScenarioServer) SetupRoutes(r *mux.Router) {
	r.HandleFunc("/a/scenario/new", a.CreateFunc).Methods("POST")
	r.HandleFunc("/a/scenario/list", a.ListFunc).Methods("GET")
	r.HandleFunc("/a/scenario/{id}", a.GetFunc).Methods("GET")
	r.HandleFunc("/a/scenario/{id}", a.UpdateFunc).Methods("PUT")
	glog.V(2).Infof("set up routes for Scenario server")
}

type PreparedScenario struct {
	ID string `json:"id"`
	hfv1.ScenarioSpec
}

func (a AdminScenarioServer) GetFunc(w http.ResponseWriter, r *http.Request) {
	_, err := a.auth.AuthNAdmin(w, r)
	if err != nil {
		util.ReturnHTTPMessage(w, r, 403, "forbidden", "no access to get Scenario")
		return
	}

	vars := mux.Vars(r)

	id := vars["id"]

	if len(id) == 0 {
		util.ReturnHTTPMessage(w, r, 500, "error", "no id passed in")
		return
	}

	scenario, err := a.getScenario(id)

	if err != nil {
		glog.Errorf("error while retrieving scenario %v", err)
		util.ReturnHTTPMessage(w, r, 500, "error", "no scenario found")
		return
	}

	preparedScenario := PreparedScenario{scenario.Name, scenario.Spec}

	encodedScenario, err := json.Marshal(preparedScenario)
	if err != nil {
		glog.Error(err)
	}
	util.ReturnHTTPContent(w, r, 200, "success", encodedScenario)

	glog.V(2).Infof("retrieved scenario %s", scenario.Name)
}

func (a AdminScenarioServer) ListFunc(w http.ResponseWriter, r *http.Request) {
	_, err := a.auth.AuthNAdmin(w, r)
	if err != nil {
		util.ReturnHTTPMessage(w, r, 403, "forbidden", "no access to list scenarios")
		return
	}

	scenarios, err := a.hfClientSet.HobbyfarmV1().Scenarios().List(metav1.ListOptions{})

	if err != nil {
		glog.Errorf("error while retrieving scenarios %v", err)
		util.ReturnHTTPMessage(w, r, 500, "error", "no scenarios found")
		return
	}

	preparedScenarios := []PreparedScenario{}
	for _, s := range scenarios.Items {
		pScenario := PreparedScenario{s.Name,s.Spec}
		pScenario.Steps = nil
		preparedScenarios = append(preparedScenarios, pScenario)
	}

	encodedScenarios, err := json.Marshal(preparedScenarios)
	if err != nil {
		glog.Error(err)
	}
	util.ReturnHTTPContent(w, r, 200, "success", encodedScenarios)

	glog.V(2).Infof("listed scenarios")
}


func (a AdminScenarioServer) CreateFunc(w http.ResponseWriter, r *http.Request) {
	_, err := a.auth.AuthNAdmin(w, r)
	if err != nil {
		util.ReturnHTTPMessage(w, r, 403, "forbidden", "no access to create scenarios")
		return
	}

	name := r.PostFormValue("name")
	if name == "" {
		util.ReturnHTTPMessage(w, r, 400, "badrequest", "no name passed in")
		return
	}
	description := r.PostFormValue("description")
	if description == "" {
		util.ReturnHTTPMessage(w, r, 400, "badrequest", "no description passed in")
		return
	}
	rawSteps := r.PostFormValue("steps")
	if rawSteps == "" {
		util.ReturnHTTPMessage(w, r, 400, "badrequest", "no steps passed in")
		return
	}
	rawVirtualMachines := r.PostFormValue("virtualmachines")
	if rawVirtualMachines == "" {
		util.ReturnHTTPMessage(w, r, 400, "badrequest", "no virtualmachines passed in")
		return
	}

	steps := []hfv1.ScenarioStep{}
	virtualmachines := []map[string]string{}

	err = json.Unmarshal([]byte(rawSteps), steps)
	if err != nil {
		glog.Errorf("error while unmarshaling steps %v", err)
		util.ReturnHTTPMessage(w, r, 500, "internalerror", "error parsing")
		return
	}

	err = json.Unmarshal([]byte(rawVirtualMachines), virtualmachines)
	if err != nil {
		glog.Errorf("error while unmarshaling VMs %v", err)
		util.ReturnHTTPMessage(w, r, 500, "internalerror", "error parsing")
		return
	}

	scenario := &hfv1.Scenario{}

	hasher := sha256.New()
	hasher.Write([]byte(name))
	sha := base32.StdEncoding.WithPadding(-1).EncodeToString(hasher.Sum(nil))[:10]
	scenario.Name = "s-" + strings.ToLower(sha)
	scenario.Spec.Id = "s-" + strings.ToLower(sha) // LEGACY!!!!

	scenario.Spec.Name = name
	scenario.Spec.Description = description
	scenario.Spec.VirtualMachines = virtualmachines
	scenario.Spec.Steps = steps

	scenario, err = a.hfClientSet.HobbyfarmV1().Scenarios().Create(scenario)
	if err != nil {
		glog.Errorf("error creating scenario %v", err)
		util.ReturnHTTPMessage(w, r, 500, "internalerror", "error creating scenario")
		return
	}

	util.ReturnHTTPMessage(w, r, 201, "created", scenario.Name)
	return
}

func (a AdminScenarioServer) UpdateFunc(w http.ResponseWriter, r *http.Request) {
	_, err := a.auth.AuthNAdmin(w, r)
	if err != nil {
		util.ReturnHTTPMessage(w, r, 403, "forbidden", "no access to update scenarios")
		return
	}

	vars := mux.Vars(r)

	id := vars["id"]
	if id == "" {
		util.ReturnHTTPMessage(w, r, 400, "badrequest", "no ID passed in")
		return
	}

	retryErr := retry.RetryOnConflict(retry.DefaultRetry, func() error {
		scenario, err := a.hfClientSet.HobbyfarmV1().Scenarios().Get(id, metav1.GetOptions{})
		if err != nil {
			glog.Error(err)
			util.ReturnHTTPMessage(w, r, 400, "badrequest", "no ID found")
			return fmt.Errorf("bad")
		}

		name := r.PostFormValue("name")
		description := r.PostFormValue("description")
		rawSteps := r.PostFormValue("steps")
		rawVirtualMachines := r.PostFormValue("virtualmachines")

		if name != "" {
			scenario.Spec.Name = name
		}
		if description != "" {
			scenario.Spec.Description = description
		}

		if rawSteps != "" {
			steps := []hfv1.ScenarioStep{}

			err = json.Unmarshal([]byte(rawSteps), rawSteps)
			if err != nil {
				glog.Errorf("error while unmarshaling steps %v", err)
				return fmt.Errorf("bad")
			}
			scenario.Spec.Steps = steps
		}

		if rawVirtualMachines != "" {
				virtualmachines := []map[string]string{}
				err = json.Unmarshal([]byte(rawVirtualMachines), virtualmachines)
				if err != nil {
					glog.Errorf("error while unmarshaling VMs %v", err)
					return fmt.Errorf("bad")
				}
				scenario.Spec.VirtualMachines = virtualmachines
		}

		_, updateErr := a.hfClientSet.HobbyfarmV1().Scenarios().Update(scenario)
		return updateErr
	})

	if retryErr != nil {
		util.ReturnHTTPMessage(w, r, 500, "error", "error attempting to update")
		return
	}

	util.ReturnHTTPMessage(w, r, 200, "updated", "")
	return
}
