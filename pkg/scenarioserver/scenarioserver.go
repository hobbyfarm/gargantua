package scenarioserver

import (
	"crypto/sha256"
	"encoding/base32"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/golang/glog"
	"github.com/gorilla/mux"
	"github.com/hobbyfarm/gargantua/pkg/accesscode"
	hfv1 "github.com/hobbyfarm/gargantua/pkg/apis/hobbyfarm.io/v1"
	"github.com/hobbyfarm/gargantua/pkg/authclient"
	hfClientset "github.com/hobbyfarm/gargantua/pkg/client/clientset/versioned"
	hfInformers "github.com/hobbyfarm/gargantua/pkg/client/informers/externalversions"
	"github.com/hobbyfarm/gargantua/pkg/util"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/retry"
	"net/http"
	"strconv"
	"strings"
)

const (
	idIndex = "scenarioserver.hobbyfarm.io/id-index"
)

type ScenarioServer struct {
	auth            *authclient.AuthClient
	hfClientSet     *hfClientset.Clientset
	acClient        *accesscode.AccessCodeClient
	scenarioIndexer cache.Indexer
}

type PreparedScenarioStep struct {
	Title   string `json:"title"`
	Content string `json:"content"`
}

type PreparedScenario struct {
	Id              string              `json:"id"`
	Name            string              `json:"name"`
	Description     string              `json:"description"`
	StepCount       int                 `json:"stepcount"`
	VirtualMachines []map[string]string `json:"virtualmachines"`
	Pauseable       bool                `json:"pauseable"`
}

type AdminPreparedScenario struct {
	ID string `json:"id"`
	hfv1.ScenarioSpec
}

func NewScenarioServer(authClient *authclient.AuthClient, acClient *accesscode.AccessCodeClient, hfClientset *hfClientset.Clientset, hfInformerFactory hfInformers.SharedInformerFactory) (*ScenarioServer, error) {
	scenario := ScenarioServer{}

	scenario.hfClientSet = hfClientset
	scenario.acClient = acClient
	scenario.auth = authClient
	inf := hfInformerFactory.Hobbyfarm().V1().Scenarios().Informer()
	indexers := map[string]cache.IndexFunc{idIndex: idIndexer}
	err := inf.AddIndexers(indexers)
	if err != nil {
		glog.Errorf("error adding scenario indexer %s", idIndex)
	}
	scenario.scenarioIndexer = inf.GetIndexer()
	return &scenario, nil
}

func (s ScenarioServer) SetupRoutes(r *mux.Router) {
	r.HandleFunc("/scenario/list", s.ListScenarioForAccessCodes).Methods("GET")
	r.HandleFunc("/a/scenario/list", s.ListFunc).Methods("GET")
	r.HandleFunc("/a/scenario/{id}", s.AdminGetFunc).Methods("GET")
	r.HandleFunc("/scenario/{scenario_id}", s.GetScenarioFunc).Methods("GET")
	r.HandleFunc("/a/scenario/{id}/printable", s.PrintFunc).Methods("GET")
	r.HandleFunc("/a/scenario/new", s.CreateFunc).Methods("POST")
	r.HandleFunc("/a/scenario/{id}", s.UpdateFunc).Methods("PUT")
	r.HandleFunc("/scenario/{scenario_id}/step/{step_id:[0-9]+}", s.GetScenarioStepFunc).Methods("GET")
	glog.V(2).Infof("set up route")
}

func (s ScenarioServer) prepareScenario(scenario hfv1.Scenario) (PreparedScenario, error) {
	ps := PreparedScenario{}

	ps.Name = scenario.Spec.Name
	ps.Id = scenario.Spec.Id
	ps.Description = scenario.Spec.Description
	ps.VirtualMachines = scenario.Spec.VirtualMachines
	ps.Pauseable = scenario.Spec.Pauseable

	var steps []PreparedScenarioStep
	for _, step := range scenario.Spec.Steps {
		steps = append(steps, PreparedScenarioStep{step.Title, step.Content})
	}

	ps.StepCount = len(scenario.Spec.Steps)

	return ps, nil
}

func (s ScenarioServer) getPreparedScenarioStepById(id string, step int) (PreparedScenarioStep, error) {
	scenario, err := s.GetScenarioById(id)
	if err != nil {
		return PreparedScenarioStep{}, fmt.Errorf("error while retrieving scenario step")
	}

	if step >= 0 && len(scenario.Spec.Steps) > step {
		stepContent := scenario.Spec.Steps[step]
		return PreparedScenarioStep{stepContent.Title, stepContent.Content}, nil
	}

	return PreparedScenarioStep{}, fmt.Errorf("error while retrieving scenario step, most likely doesn't exist in index")
}

func (s ScenarioServer) getPreparedScenarioById(id string) (PreparedScenario, error) {
	scenario, err := s.GetScenarioById(id)

	if err != nil {
		return PreparedScenario{}, fmt.Errorf("error while retrieving scenario %v", err)
	}

	preparedScenario, err := s.prepareScenario(scenario)

	if err != nil {
		return PreparedScenario{}, fmt.Errorf("error while preparing scenario %v", err)
	}

	return preparedScenario, nil
}

func (s ScenarioServer) GetScenarioFunc(w http.ResponseWriter, r *http.Request) {
	_, err := s.auth.AuthN(w, r)
	if err != nil {
		util.ReturnHTTPMessage(w, r, 403, "forbidden", "no access to get scenarios")
		return
	}

	vars := mux.Vars(r)

	scenario, err := s.getPreparedScenarioById(vars["scenario_id"])
	if err != nil {
		util.ReturnHTTPMessage(w, r, 404, "not found", fmt.Sprintf("scenario %s not found", vars["scenario_id"]))
		return
	}
	encodedScenario, err := json.Marshal(scenario)
	if err != nil {
		glog.Error(err)
	}
	util.ReturnHTTPContent(w, r, 200, "success", encodedScenario)
}

func (s ScenarioServer) AdminGetFunc(w http.ResponseWriter, r *http.Request) {
	_, err := s.auth.AuthNAdmin(w, r)
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

	scenario, err := s.GetScenarioById(id)

	if err != nil {
		glog.Errorf("error while retrieving scenario %v", err)
		util.ReturnHTTPMessage(w, r, 500, "error", "no scenario found")
		return
	}

	preparedScenario := AdminPreparedScenario{scenario.Name, scenario.Spec}

	encodedScenario, err := json.Marshal(preparedScenario)
	if err != nil {
		glog.Error(err)
	}
	util.ReturnHTTPContent(w, r, 200, "success", encodedScenario)

	glog.V(2).Infof("retrieved scenario %s", scenario.Name)
}

func (s ScenarioServer) GetScenarioStepFunc(w http.ResponseWriter, r *http.Request) {
	_, err := s.auth.AuthN(w, r)
	if err != nil {
		util.ReturnHTTPMessage(w, r, 403, "forbidden", "no access to get scenario steps")
		return
	}

	vars := mux.Vars(r)

	stepId, err := strconv.Atoi(vars["step_id"])
	if err != nil {
		util.ReturnHTTPMessage(w, r, 404, "not found", fmt.Sprintf("scenario %s step %s not found", vars["scenario_id"], vars["step_id"]))
		return
	}
	step, err := s.getPreparedScenarioStepById(vars["scenario_id"], stepId)
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

func (s ScenarioServer) ListScenarioForAccessCodes(w http.ResponseWriter, r *http.Request) {
	user, err := s.auth.AuthN(w, r)
	if err != nil {
		util.ReturnHTTPMessage(w, r, 403, "forbidden", "no access to list scenarios")
		return
	}

	// store a list of scenarios linked to courses for filtering
	//var courseScenarios []string
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
		scenario, err := s.GetScenarioById(scenarioId)
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

func (s ScenarioServer) ListFunc(w http.ResponseWriter, r *http.Request) {
	_, err := s.auth.AuthNAdmin(w, r)
	if err != nil {
		util.ReturnHTTPMessage(w, r, 403, "forbidden", "no access to list scenarios")
		return
	}

	scenarios, err := s.hfClientSet.HobbyfarmV1().Scenarios().List(metav1.ListOptions{})

	if err != nil {
		glog.Errorf("error while retrieving scenarios %v", err)
		util.ReturnHTTPMessage(w, r, 500, "error", "no scenarios found")
		return
	}

	preparedScenarios := []AdminPreparedScenario{}
	for _, s := range scenarios.Items {
		pScenario := AdminPreparedScenario{s.Name, s.Spec}
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

func (s ScenarioServer) PrintFunc(w http.ResponseWriter, r *http.Request) {
	_, err := s.auth.AuthNAdmin(w, r)
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

	scenario, err := s.GetScenarioById(id)

	if err != nil {
		glog.Errorf("error while retrieving scenario %v", err)
		util.ReturnHTTPMessage(w, r, 500, "error", "no scenario found")
		return
	}

	var content string

	name, err := base64.StdEncoding.DecodeString(scenario.Spec.Name)
	if err != nil {
		glog.Errorf("Error decoding title of scenario: %s %v", scenario.Name, err)
	}
	description, err := base64.StdEncoding.DecodeString(scenario.Spec.Description)
	if err != nil {
		glog.Errorf("Error decoding description of scenario: %s %v", scenario.Name, err)
	}

	content = fmt.Sprintf("# %s\n%s\n\n", name, description)

	for i, s := range scenario.Spec.Steps {

		title, err := base64.StdEncoding.DecodeString(s.Title)
		if err != nil {
			glog.Errorf("Error decoding title of scenario: %s step %d: %v", scenario.Name, i, err)
		}

		content = content + fmt.Sprintf("## Step %d: %s\n", i+1, string(title))

		stepContent, err := base64.StdEncoding.DecodeString(s.Content)
		if err != nil {
			glog.Errorf("Error decoding content of scenario: %s step %d: %v", scenario.Name, i, err)
		}

		content = content + fmt.Sprintf("%s\n", string(stepContent))
	}

	util.ReturnHTTPRaw(w, r, content)

	glog.V(2).Infof("retrieved scenario and rendered for printability %s", scenario.Name)
}

func (s ScenarioServer) CreateFunc(w http.ResponseWriter, r *http.Request) {
	_, err := s.auth.AuthNAdmin(w, r)
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

	keepaliveDuration := r.PostFormValue("keepalive_duration")
	// we won't error if no keep alive duration is passed in or if it's blank because we'll default elsewhere

	steps := []hfv1.ScenarioStep{}
	virtualmachines := []map[string]string{}

	rawSteps := r.PostFormValue("steps")
	if rawSteps != "" {
		err = json.Unmarshal([]byte(rawSteps), &steps)
		if err != nil {
			glog.Errorf("error while unmarshaling steps %v", err)
			util.ReturnHTTPMessage(w, r, 500, "internalerror", "error parsing")
			return
		}
	}

	rawVirtualMachines := r.PostFormValue("virtualmachines")
	if rawVirtualMachines != "" {
		err = json.Unmarshal([]byte(rawVirtualMachines), &virtualmachines)
		if err != nil {
			glog.Errorf("error while unmarshaling VMs %v", err)
			util.ReturnHTTPMessage(w, r, 500, "internalerror", "error parsing")
			return
		}
	}

	pauseable := r.PostFormValue("pauseable")
	pauseDuration := r.PostFormValue("pause_duration")

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
	scenario.Spec.KeepAliveDuration = keepaliveDuration

	scenario.Spec.Pauseable = false
	if pauseable != "" {
		if strings.ToLower(pauseable) == "true" {
			scenario.Spec.Pauseable = true
		}
	}

	if pauseDuration != "" {
		scenario.Spec.PauseDuration = pauseDuration
	}

	scenario, err = s.hfClientSet.HobbyfarmV1().Scenarios().Create(scenario)
	if err != nil {
		glog.Errorf("error creating scenario %v", err)
		util.ReturnHTTPMessage(w, r, 500, "internalerror", "error creating scenario")
		return
	}

	util.ReturnHTTPMessage(w, r, 201, "created", scenario.Name)
	return
}

func (s ScenarioServer) UpdateFunc(w http.ResponseWriter, r *http.Request) {
	_, err := s.auth.AuthNAdmin(w, r)
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
		scenario, err := s.hfClientSet.HobbyfarmV1().Scenarios().Get(id, metav1.GetOptions{})
		if err != nil {
			glog.Error(err)
			util.ReturnHTTPMessage(w, r, 400, "badrequest", "no ID found")
			return fmt.Errorf("bad")
		}

		name := r.PostFormValue("name")
		description := r.PostFormValue("description")
		rawSteps := r.PostFormValue("steps")
		pauseable := r.PostFormValue("pauseable")
		pauseDuration := r.PostFormValue("pause_duration")
		keepaliveDuration := r.PostFormValue("keepalive_duration")
		rawVirtualMachines := r.PostFormValue("virtualmachines")

		if name != "" {
			scenario.Spec.Name = name
		}
		if description != "" {
			scenario.Spec.Description = description
		}
		if keepaliveDuration != "" {
			scenario.Spec.KeepAliveDuration = keepaliveDuration
		}

		if pauseable != "" {
			if strings.ToLower(pauseable) == "true" {
				scenario.Spec.Pauseable = true
			} else {
				scenario.Spec.Pauseable = false
			}
		}

		if pauseDuration != "" {
			scenario.Spec.PauseDuration = pauseDuration
		}

		if rawSteps != "" {
			steps := []hfv1.ScenarioStep{}

			err = json.Unmarshal([]byte(rawSteps), &steps)
			if err != nil {
				glog.Errorf("error while unmarshaling steps %v", err)
				return fmt.Errorf("bad")
			}
			scenario.Spec.Steps = steps
		}

		if rawVirtualMachines != "" {
			virtualmachines := []map[string]string{}
			err = json.Unmarshal([]byte(rawVirtualMachines), &virtualmachines)
			if err != nil {
				glog.Errorf("error while unmarshaling VMs %v", err)
				return fmt.Errorf("bad")
			}
			scenario.Spec.VirtualMachines = virtualmachines
		}

		_, updateErr := s.hfClientSet.HobbyfarmV1().Scenarios().Update(scenario)
		return updateErr
	})

	if retryErr != nil {
		util.ReturnHTTPMessage(w, r, 500, "error", "error attempting to update")
		return
	}

	util.ReturnHTTPMessage(w, r, 200, "updated", "")
	return
}

func (s ScenarioServer) GetScenarioById(id string) (hfv1.Scenario, error) {
	if len(id) == 0 {
		return hfv1.Scenario{}, fmt.Errorf("scenario id passed in was blank")
	}
	obj, err := s.scenarioIndexer.ByIndex(idIndex, id)

	if err != nil {
		return hfv1.Scenario{}, fmt.Errorf("error while retrieving scenario by ID %s %v", id, err)
	}

	if len(obj) < 1 {
		return hfv1.Scenario{}, fmt.Errorf("error while retrieving scenario by ID %s", id)
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
