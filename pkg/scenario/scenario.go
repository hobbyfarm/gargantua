package scenario

import (
	"fmt"
	"github.com/golang/glog"
	"github.com/gorilla/mux"
	"github.com/hobbyfarm/gargantua/pkg/accesscode"
	"github.com/hobbyfarm/gargantua/pkg/authclient"
	hfv1 "github.com/hobbyfarm/gargantua/pkg/apis/hobbyfarm.io/v1"
	hfInformers "github.com/hobbyfarm/gargantua/pkg/client/informers/externalversions"
	hfClientset "github.com/hobbyfarm/gargantua/pkg/client/clientset/versioned"
	"github.com/hobbyfarm/gargantua/pkg/util"
	"k8s.io/client-go/tools/cache"
	"net/http"
	"encoding/json"
	"strconv"
)

const (
	idIndex = "scenario.hobbyfarm.io/id-index"
)

type Scenario struct {

	auth *authclient.AuthClient
	hfClientSet *hfClientset.Clientset
	acClient *accesscode.AccessCodeClient
	scenarioIndexer cache.Indexer

}

type PreparedScenarioStep struct {
	Title string `json:"title"`
	Content string `json:"content"`
}

type PreparedScenario struct {
	Id string `json:"id"`
	Name string `json:"name"`
	Description string `json:"description"`
	StepCount int `json:"stepcount"`
}

func NewScenario(authClient *authclient.AuthClient, acClient *accesscode.AccessCodeClient, hfClientset *hfClientset.Clientset, hfInformerFactory hfInformers.SharedInformerFactory) (*Scenario, error) {
	scenario := Scenario{}

	scenario.hfClientSet = hfClientset
	scenario.acClient = acClient
	scenario.auth = authClient
	inf := hfInformerFactory.Hobbyfarm().V1().Scenarios().Informer()
	indexers := map[string]cache.IndexFunc{idIndex: idIndexer}
	inf.AddIndexers(indexers)
	scenario.scenarioIndexer = inf.GetIndexer()
	return &scenario, nil
}


func (s Scenario) SetupRoutes(r *mux.Router) {
	r.HandleFunc("/scenario/list", s.ListScenarioFunc)
	r.HandleFunc("/scenario/{scenario_id}", s.GetScenarioFunc)
	r.HandleFunc("/scenario/{scenario_id}/step/{step_id:[0-9]+}", s.GetScenarioStepFunc)
	glog.V(2).Infof("set up route")
}

func (s Scenario) prepareScenario(scenario hfv1.Scenario) (PreparedScenario, error) {
	ps := PreparedScenario{}

	ps.Name = scenario.Name
	ps.Id = scenario.Spec.Id
	ps.Description = scenario.Spec.Description

	var steps []PreparedScenarioStep
	for _, step := range scenario.Spec.Steps {
		steps = append(steps, PreparedScenarioStep{step.Title, step.Content})
	}

	ps.StepCount = len(scenario.Spec.Steps)

	return ps, nil
}

func (s Scenario) getPreparedScenarioStepById(id string, step int) (PreparedScenarioStep, error){
	scenario, err := s.getScenarioById(id)
	if err != nil {
		return PreparedScenarioStep{}, fmt.Errorf("error while retrieving scenario step")
	}

	if step >= 0 && len(scenario.Spec.Steps) > step {
		stepContent := scenario.Spec.Steps[step]
		return PreparedScenarioStep{stepContent.Title, stepContent.Content}, nil
	}

	return PreparedScenarioStep{}, fmt.Errorf("error while retrieving scenario step, most likely doesn't exist in index")
}

func (s Scenario) getPreparedScenarioById(id string) (PreparedScenario, error) {
	scenario, err := s.getScenarioById(id)

	if err != nil {
		return PreparedScenario{}, fmt.Errorf("error while retrieving scenario %v", err)
	}

	preparedScenario, err := s.prepareScenario(scenario)

	if err != nil {
		return PreparedScenario{}, fmt.Errorf("error while preparing scenario %v", err)
	}

	return preparedScenario, nil
}


func (s Scenario) GetScenarioFunc(w http.ResponseWriter, r *http.Request) {
	_, err := s.auth.AuthN(w, r)
	if err != nil {
		util.ReturnHTTPMessage(w, r, 403, "forbidden", "no access to get scenarios")
		return
	}

	vars := mux.Vars(r)

	scenario, err := s.getPreparedScenarioById(vars["scenario_id"])
	if err != nil {
		util.ReturnHTTPMessage(w, r, 404, "not found", fmt.Sprintf("scenario %s not found", vars["scenario_id"]))
		returnok 
	}
	encodedScenario, err := json.Marshal(scenario)
	if err != nil {
		glog.Error(err)
	}
	util.ReturnHTTPContent(w, r, 200, "success", encodedScenario)
}
func (s Scenario) GetScenarioStepFunc(w http.ResponseWriter, r *http.Request) {
	_, err := s.auth.AuthN(w, r)
	if err != nil {
		util.ReturnHTTPMessage(w, r, 403, "forbidden", "no access to get scenario steps")
		return
	}

	vars := mux.Vars(r)

	step_id, err := strconv.Atoi(vars["step_id"])
	if err != nil {
		util.ReturnHTTPMessage(w, r, 404, "not found", fmt.Sprintf("scenario %s step %s not found", vars["scenario_id"], vars["step_id"]))
		return
	}
	step, err := s.getPreparedScenarioStepById(vars["scenario_id"], step_id)
	if err != nil {
		util.ReturnHTTPMessage(w, r, 404, "not found", fmt.Sprintf("scenario %s not found", vars["scenario_id"]))
		return
	}
	encodedStep, err := json.Marshal(step)
	if err != nil {
		glog.Error(err)
	}
	util.ReturnHTTPContent(w, r, 200, "success", encodedStep)

}

func (s Scenario) ListScenarioFunc(w http.ResponseWriter, r *http.Request) {
	user, err := s.auth.AuthN(w, r)
	if err != nil {
		util.ReturnHTTPMessage(w, r, 403, "forbidden", "no access to list scenarios")
		return
	}

	var scenarioIds []string
	for _, ac := range user.Spec.AccessCodes {
		tempScenarioIds, err := s.acClient.GetScenarioIds(ac)
		if err != nil {
			glog.Errorf("error retrieving scenario ids for access code: %s %v", ac, err)
		} else {
			scenarioIds = append(scenarioIds, tempScenarioIds...)
		}
	}

	scenarioIds = util.UniqueStringSlice(scenarioIds)

	var scenarios []PreparedScenario
	for _, scenarioId := range scenarioIds {
		scenario, err := s.getScenarioById(scenarioId)
		if err != nil {
			glog.Errorf("error retrieving scenario %v", err)
		} else {
			pScenario, err := s.prepareScenario(scenario)
			if err != nil {
				glog.Errorf("error preparing scenario %v", err)
			} else {
				scenarios = append(scenarios, pScenario)
			}
		}
	}

	encodedScenarios, err := json.Marshal(scenarios)
	if err != nil {
		glog.Error(err)
	}
	util.ReturnHTTPContent(w, r, 200, "success", encodedScenarios)
}

func (s Scenario) getScenarioById(id string) (hfv1.Scenario, error) {
	if len(id) == 0 {
		return hfv1.Scenario{}, fmt.Errorf("scenario id passed in was blank")
	}
	obj, err := s.scenarioIndexer.ByIndex(idIndex, id)

	if err != nil {
		return hfv1.Scenario{}, fmt.Errorf("error while retrieving scenario by ID %s %v", id, err)
	}

	if len(obj) < 1 {
		return hfv1.Scenario{} , fmt.Errorf("error while retrieving scenario by ID %s", id)
	}

	scenario, ok := obj[0].(*hfv1.Scenario)

	if !ok {
		return hfv1.Scenario{}, fmt.Errorf("error while retrieving scenario by ID %s %v", id, ok)
	}

	return *scenario, nil

}
func idIndexer(obj interface{}) ([]string, error) {
	scenario, ok := obj.(*hfv1.Scenario)
	if !ok {
		return []string{}, nil
	}
	return []string{scenario.Spec.Id}, nil
}