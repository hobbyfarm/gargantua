package scenarioserver

import (
	"context"
	"crypto/sha256"
	"encoding/base32"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/hobbyfarm/gargantua/v3/pkg/rbacclient"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"slices"

	"github.com/golang/glog"
	"github.com/gorilla/mux"
	"github.com/hobbyfarm/gargantua/v3/pkg/accesscode"
	hfv1 "github.com/hobbyfarm/gargantua/v3/pkg/apis/hobbyfarm.io/v1"
	"github.com/hobbyfarm/gargantua/v3/pkg/authclient"
	hfClientset "github.com/hobbyfarm/gargantua/v3/pkg/client/clientset/versioned"
	hfInformers "github.com/hobbyfarm/gargantua/v3/pkg/client/informers/externalversions"
	"github.com/hobbyfarm/gargantua/v3/pkg/courseclient"
	"github.com/hobbyfarm/gargantua/v3/pkg/util"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/retry"
)

const (
	idIndex        = "scenarioserver.hobbyfarm.io/id-index"
	resourcePlural = "scenarios"
)

type ScenarioServer struct {
	auth            *authclient.AuthClient
	hfClientSet     hfClientset.Interface
	acClient        *accesscode.AccessCodeClient
	scenarioIndexer cache.Indexer
	ctx             context.Context
	courseClient    *courseclient.CourseClient
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
	Printable       bool                `json:"printable"`
}

type AdminPreparedScenario struct {
	ID string `json:"id"`
	hfv1.ScenarioSpec
}

func NewScenarioServer(authClient *authclient.AuthClient, acClient *accesscode.AccessCodeClient, hfClientset hfClientset.Interface, hfInformerFactory hfInformers.SharedInformerFactory, ctx context.Context, courseClient *courseclient.CourseClient) (*ScenarioServer, error) {
	scenario := ScenarioServer{}

	scenario.hfClientSet = hfClientset
	scenario.acClient = acClient
	scenario.courseClient = courseClient
	scenario.auth = authClient
	inf := hfInformerFactory.Hobbyfarm().V1().Scenarios().Informer()
	indexers := map[string]cache.IndexFunc{idIndex: idIndexer}
	err := inf.AddIndexers(indexers)
	if err != nil {
		glog.Errorf("error adding scenario indexer %s", idIndex)
	}
	scenario.scenarioIndexer = inf.GetIndexer()
	scenario.ctx = ctx
	return &scenario, nil
}

func (s ScenarioServer) SetupRoutes(r *mux.Router) {
	r.HandleFunc("/scenario/list/{access_code}", s.ListScenariosForAccessCode).Methods("GET")
	r.HandleFunc("/a/scenario/categories", s.ListCategories).Methods("GET")
	r.HandleFunc("/a/scenario/list/{category}", s.ListByCategoryFunc).Methods("GET")
	r.HandleFunc("/a/scenario/list", s.ListAllFunc).Methods("GET")
	r.HandleFunc("/a/scenario/{id}", s.AdminGetFunc).Methods("GET")
	r.HandleFunc("/a/scenario/{id}", s.AdminDeleteFunc).Methods("DELETE")
	r.HandleFunc("/scenario/{scenario_id}", s.GetScenarioFunc).Methods("GET")
	r.HandleFunc("/scenario/{id}/printable", s.PrintFunc).Methods("GET")
	r.HandleFunc("/a/scenario/{id}/printable", s.AdminPrintFunc).Methods("GET")
	r.HandleFunc("/a/scenario/new", s.CreateFunc).Methods("POST")
	r.HandleFunc("/a/scenario/copy/{id}", s.CopyFunc).Methods("POST")
	r.HandleFunc("/a/scenario/{id}", s.UpdateFunc).Methods("PUT")
	r.HandleFunc("/scenario/{scenario_id}/step/{step_id:[0-9]+}", s.GetScenarioStepFunc).Methods("GET")
	glog.V(2).Infof("set up route")
}

func (s ScenarioServer) prepareScenario(scenario hfv1.Scenario, printable bool) (PreparedScenario, error) {
	ps := PreparedScenario{}

	ps.Id = scenario.Name
	ps.Name = scenario.Spec.Name
	ps.Description = scenario.Spec.Description
	ps.VirtualMachines = scenario.Spec.VirtualMachines
	ps.Pauseable = scenario.Spec.Pauseable
	ps.Printable = printable
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

func (s ScenarioServer) getPrintableScenarioIds(accessCodes []string) []string {
	var printableScenarioIds []string
	var printableCourseIds []string
	accessCodeObjs, err := s.acClient.GetAccessCodesWithOTACs(accessCodes)
	if err != nil {
		glog.Errorf("error retrieving access codes %v", err)
		return []string{}
	}
	for _, accessCode  := range accessCodeObjs {
		if !accessCode.Spec.Printable {
			continue
		}
		printableScenarioIds = append(printableScenarioIds, accessCode.Spec.Scenarios...)
		printableCourseIds = append(printableCourseIds, accessCode.Spec.Courses...)
	}
	printableCourseIds = util.UniqueStringSlice(printableCourseIds)

	for _, courseId := range printableCourseIds {
		course, err := s.courseClient.GetCourseById(courseId)
		if err != nil {
			glog.Errorf("error retrieving course %v", err)
			continue
		}
		printableScenarioIds = append(printableScenarioIds, s.courseClient.AppendDynamicScenariosByCategories(course.Spec.Scenarios, course.Spec.Categories)...)
	}

	printableScenarioIds = util.UniqueStringSlice(printableScenarioIds)
	return printableScenarioIds
}

func (s ScenarioServer) getPreparedScenarioById(id string, accessCodes []string) (PreparedScenario, error) {
	scenario, err := s.GetScenarioById(id)

	if err != nil {
		return PreparedScenario{}, fmt.Errorf("error while retrieving scenario %v", err)
	}

	printableScenarioIds := s.getPrintableScenarioIds(accessCodes)
	printable := slices.Contains(printableScenarioIds, scenario.Name)

	preparedScenario, err := s.prepareScenario(scenario, printable)

	if err != nil {
		return PreparedScenario{}, fmt.Errorf("error while preparing scenario %v", err)
	}

	return preparedScenario, nil
}

func (s ScenarioServer) GetScenarioFunc(w http.ResponseWriter, r *http.Request) {
	user, err := s.auth.AuthN(w, r)
	if err != nil {
		util.ReturnHTTPMessage(w, r, 403, "forbidden", "no access to get scenarios")
		return
	}

	vars := mux.Vars(r)

	scenario_id := vars["scenario_id"]

	if len(scenario_id) == 0 {
		util.ReturnHTTPMessage(w, r, 500, "error", "no scenario id passed in")
		return
	}

	scenario, err := s.getPreparedScenarioById(scenario_id, user.Spec.AccessCodes)
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
	_, err := s.auth.AuthGrant(rbacclient.RbacRequest().HobbyfarmPermission(resourcePlural, rbacclient.VerbGet), w, r)
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

func (s ScenarioServer) AdminDeleteFunc(w http.ResponseWriter, r *http.Request) {
	_, err := s.auth.AuthGrant(rbacclient.RbacRequest().HobbyfarmPermission(resourcePlural, rbacclient.VerbDelete), w, r)
	if err != nil {
		util.ReturnHTTPMessage(w, r, 403, "forbidden", "no access to delete Scenario")
		return
	}

	vars := mux.Vars(r)

	id := vars["id"]

	if len(id) == 0 {
		util.ReturnHTTPMessage(w, r, 400, "error", "no id passed in")
		return
	}

	// when can we safely a scenario?
	// 1. when there are no active scheduled events using the scenario
	// 2. when there are no sessions using the scenario
	// 3. when there is no course using the scenario

	seList, err := s.hfClientSet.HobbyfarmV1().ScheduledEvents(util.GetReleaseNamespace()).List(s.ctx, metav1.ListOptions{})
	if err != nil {
		glog.Errorf("error retrieving scheduledevent list: %v", err)
		util.ReturnHTTPMessage(w, r, 500, "internalerror", "error while deleting scenario")
		return
	}

	seInUse := filterScheduledEvents(id, seList)

	sessList, err := s.hfClientSet.HobbyfarmV1().Sessions(util.GetReleaseNamespace()).List(s.ctx, metav1.ListOptions{})
	if err != nil {
		glog.Errorf("error retrieving session list: %v", err)
		util.ReturnHTTPMessage(w, r, 500, "internalerror", "error while deleting scenario")
		return
	}

	sessInUse := filterSessions(id, sessList)

	courseList, err := s.hfClientSet.HobbyfarmV1().Courses(util.GetReleaseNamespace()).List(s.ctx, metav1.ListOptions{})
	if err != nil {
		glog.Errorf("error retrieving course list: %v", err)
		util.ReturnHTTPMessage(w, r, 500, "internalerror", "error while deleting scenario")
		return
	}

	coursesInUse := filterCourses(id, courseList)

	var msg = ""
	toDelete := true

	if len(*seInUse) > 0 {
		// cannot toDelete, in use. alert the user
		msg += "In use by scheduled events:"
		for _, se := range *seInUse {
			msg += " " + se.Name
		}
		toDelete = false
	}

	if len(*sessInUse) > 0 {
		msg += "In use by sessions:"
		for _, sess := range *sessInUse {
			msg += " " + sess.Name
		}
		toDelete = false
	}

	if len(*coursesInUse) > 0 {
		msg += "In use by courses:"
		for _, course := range *coursesInUse {
			msg += " " + course.Name
		}
		toDelete = false
	}

	if !toDelete {
		util.ReturnHTTPMessage(w, r, 403, "badrequest", msg)
		return
	}

	err = s.hfClientSet.HobbyfarmV1().Scenarios(util.GetReleaseNamespace()).Delete(s.ctx, id, metav1.DeleteOptions{})

	if err != nil {
		glog.Errorf("error while deleting scenario %v", err)
		util.ReturnHTTPMessage(w, r, 500, "error", "scenario could not be deleted")
		return
	}
	util.ReturnHTTPMessage(w, r, 200, "success", "scenario deleted")
	glog.V(2).Infof("deleted scenario %s", id)
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

func (s ScenarioServer) ListScenariosForAccessCode(w http.ResponseWriter, r *http.Request) {
	user, err := s.auth.AuthN(w, r)
	if err != nil {
		util.ReturnHTTPMessage(w, r, 403, "forbidden", "no access to list scenarios")
		return
	}

	vars := mux.Vars(r)
	accessCode := vars["access_code"]

	if accessCode == "" {
		util.ReturnHTTPMessage(w, r, 400, "badrequest", "access_code is missing")
		return
	}

	if !slices.Contains(user.Spec.AccessCodes, accessCode) {
		util.ReturnHTTPMessage(w, r, 403, "forbidden", "no access to list scenarios for this AccessCode")
		return
	}

	// store a list of scenarios linked to courses for filtering
	//var courseScenarios []string
	var scenarioIds []string
	ac, err := s.acClient.GetAccessCodeWithOTACs(accessCode)
	if err != nil {
		glog.Errorf("error retrieving access code: %s %v", accessCode, err)
	}
	scenarioIds = append(scenarioIds, ac.Spec.Scenarios...)

	var scenarios []PreparedScenario
	for _, scenarioId := range scenarioIds {
		scenario, err := s.GetScenarioById(scenarioId)
		if err != nil {
			glog.Errorf("error retrieving scenario %v", err)
			continue
		}
		pScenario, err := s.prepareScenario(scenario, ac.Spec.Printable)
		if err != nil {
			glog.Errorf("error preparing scenario %v", err)
			continue
		}
		scenarios = append(scenarios, pScenario)
	}

	encodedScenarios, err := json.Marshal(scenarios)
	if err != nil {
		glog.Error(err)
	}
	util.ReturnHTTPContent(w, r, 200, "success", encodedScenarios)
}

func (s ScenarioServer) ListAllFunc(w http.ResponseWriter, r *http.Request) {
	s.ListFunc(w, r, "")
}

func (s ScenarioServer) ListByCategoryFunc(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)

	category := vars["category"]

	if len(category) == 0 {
		util.ReturnHTTPMessage(w, r, 500, "error", "no category passed in")
		return
	}

	s.ListFunc(w, r, category)
}

func (s ScenarioServer) ListFunc(w http.ResponseWriter, r *http.Request, category string) {
	_, err := s.auth.AuthGrant(rbacclient.RbacRequest().HobbyfarmPermission(resourcePlural, rbacclient.VerbList), w, r)
	if err != nil {
		util.ReturnHTTPMessage(w, r, 403, "forbidden", "no access to list scenarios")
		return
	}

	categorySelector := metav1.ListOptions{}
	if category != "" {
		categorySelector = metav1.ListOptions{
			LabelSelector: fmt.Sprintf("category-%s=true", category),
		}
	}

	scenarios, err := s.hfClientSet.HobbyfarmV1().Scenarios(util.GetReleaseNamespace()).List(s.ctx, categorySelector)

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

func (s ScenarioServer) ListCategories(w http.ResponseWriter, r *http.Request) {
	_, err := s.auth.AuthGrant(rbacclient.RbacRequest().HobbyfarmPermission(resourcePlural, rbacclient.VerbList), w, r)
	if err != nil {
		util.ReturnHTTPMessage(w, r, 403, "forbidden", "no access to list categories")
		return
	}

	scenarios, err := s.hfClientSet.HobbyfarmV1().Scenarios(util.GetReleaseNamespace()).List(s.ctx, metav1.ListOptions{})

	if err != nil {
		glog.Errorf("error while retrieving scenarios %v", err)
		util.ReturnHTTPMessage(w, r, 500, "error", "no scenarios found")
		return
	}

	categories := []string{}

	for _, s := range scenarios.Items {
		if len(s.Spec.Categories) != 0 {
			categories = append(categories, s.Spec.Categories...)
		}
	}

	categories = util.UniqueStringSlice(categories)
	sort.Strings(categories)

	encodedCategories, err := json.Marshal(categories)
	if err != nil {
		glog.Error(err)
	}
	util.ReturnHTTPContent(w, r, 200, "success", encodedCategories)

	glog.V(2).Infof("listed categories")
}

func (s ScenarioServer) AdminPrintFunc(w http.ResponseWriter, r *http.Request) {
	_, err := s.auth.AuthGrant(rbacclient.RbacRequest().HobbyfarmPermission(resourcePlural, rbacclient.VerbGet), w, r)
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

func (s ScenarioServer) PrintFunc(w http.ResponseWriter, r *http.Request) {
	user, err := s.auth.AuthN(w, r)
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

	printableScenarioIds := s.getPrintableScenarioIds(user.Spec.AccessCodes)

	if !slices.Contains(printableScenarioIds, id) {
		util.ReturnHTTPMessage(w, r, 403, "forbidden", "no access to get this Scenario")
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

		content = content + fmt.Sprintf("\n## Step %d: %s\n", i+1, string(title))

		stepContent, err := base64.StdEncoding.DecodeString(s.Content)
		if err != nil {
			glog.Errorf("Error decoding content of scenario: %s step %d: %v", scenario.Name, i, err)
		}

		content = content + fmt.Sprintf("%s\n", string(stepContent))
	}

	util.ReturnHTTPRaw(w, r, content)

	glog.V(2).Infof("retrieved scenario and rendered for printability %s", scenario.Name)
}

func (s ScenarioServer) CopyFunc(w http.ResponseWriter, r *http.Request) {
	_, err := s.auth.AuthGrant(
		rbacclient.RbacRequest().
			HobbyfarmPermission(resourcePlural, rbacclient.VerbCreate).
			HobbyfarmPermission(resourcePlural, rbacclient.VerbGet),
		w, r)
	if err != nil {
		util.ReturnHTTPMessage(w, r, 403, "forbidden", "no access to create scenarios")
		return
	}

	vars := mux.Vars(r)

	id := vars["id"]

	if len(id) == 0 {
		util.ReturnHTTPMessage(w, r, 500, "error", "no id passed in")
		return
	}

	retryErr := retry.RetryOnConflict(retry.DefaultRetry, func() error {
		scenario, err := s.hfClientSet.HobbyfarmV1().Scenarios(util.GetReleaseNamespace()).Get(s.ctx, id, metav1.GetOptions{})
		if err != nil {
			glog.Error(err)
			util.ReturnHTTPMessage(w, r, http.StatusNotFound, "badrequest", "no scenario found with given ID")
			return fmt.Errorf("bad")
		}

		name, err := base64.StdEncoding.DecodeString(scenario.Spec.Name)
		if err != nil {
			glog.Errorf("Error decoding title of scenario to copy: %s %v", scenario.Name, err)
		}
		copyName := string(name) + " - Copy"
		copyName = base64.StdEncoding.EncodeToString([]byte(copyName))
		hasher := sha256.New()
		hasher.Write([]byte(copyName))
		sha := base32.StdEncoding.WithPadding(-1).EncodeToString(hasher.Sum(nil))[:10]

		scenario.Spec.Name = copyName
		scenario.Name = "s-" + strings.ToLower(sha)

		_, updateErr := s.hfClientSet.HobbyfarmV1().Scenarios(util.GetReleaseNamespace()).Create(s.ctx, scenario, metav1.CreateOptions{})
		return updateErr
	})

	if retryErr != nil {
		util.ReturnHTTPMessage(w, r, 500, "error", "error attempting to copy")
		return
	}

	util.ReturnHTTPMessage(w, r, 200, "copied scenario", "")
	return
}

func (s ScenarioServer) CreateFunc(w http.ResponseWriter, r *http.Request) {
	_, err := s.auth.AuthGrant(rbacclient.RbacRequest().HobbyfarmPermission(resourcePlural, rbacclient.VerbCreate), w, r)
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
	categories := []string{}
	tags := []string{}

	rawSteps := r.PostFormValue("steps")
	if rawSteps != "" {
		err = json.Unmarshal([]byte(rawSteps), &steps)
		if err != nil {
			glog.Errorf("error while unmarshaling steps %v", err)
			util.ReturnHTTPMessage(w, r, 500, "internalerror", "error parsing")
			return
		}
	}

	rawCategories := r.PostFormValue("categories")
	if rawCategories != "" {
		err = json.Unmarshal([]byte(rawCategories), &categories)
		if err != nil {
			glog.Errorf("error while unmarshaling categories %v", err)
			util.ReturnHTTPMessage(w, r, 500, "internalerror", "error parsing")
			return
		}
	}

	rawTags := r.PostFormValue("tags")
	if rawTags != "" {
		err = json.Unmarshal([]byte(rawTags), &tags)
		if err != nil {
			glog.Errorf("error while unmarshaling tags %v", err)
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

	scenario.Spec.Name = name
	scenario.Spec.Description = description
	scenario.Spec.VirtualMachines = virtualmachines
	scenario.Spec.Steps = steps
	scenario.Spec.Categories = categories
	scenario.Spec.Tags = tags
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

	scenario, err = s.hfClientSet.HobbyfarmV1().Scenarios(util.GetReleaseNamespace()).Create(s.ctx, scenario, metav1.CreateOptions{})
	if err != nil {
		glog.Errorf("error creating scenario %v", err)
		util.ReturnHTTPMessage(w, r, 500, "internalerror", "error creating scenario")
		return
	}

	util.ReturnHTTPMessage(w, r, 201, "created", scenario.Name)
	return
}

func (s ScenarioServer) UpdateFunc(w http.ResponseWriter, r *http.Request) {
	_, err := s.auth.AuthGrant(rbacclient.RbacRequest().HobbyfarmPermission(resourcePlural, rbacclient.VerbUpdate), w, r)
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
		scenario, err := s.hfClientSet.HobbyfarmV1().Scenarios(util.GetReleaseNamespace()).Get(s.ctx, id, metav1.GetOptions{})
		if err != nil {
			glog.Error(err)
			util.ReturnHTTPMessage(w, r, http.StatusNotFound, "badrequest", "no scenario found with given ID")
			return fmt.Errorf("bad")
		}

		name := r.PostFormValue("name")
		description := r.PostFormValue("description")
		rawSteps := r.PostFormValue("steps")
		pauseable := r.PostFormValue("pauseable")
		pauseDuration := r.PostFormValue("pause_duration")
		keepaliveDuration := r.PostFormValue("keepalive_duration")
		rawVirtualMachines := r.PostFormValue("virtualmachines")
		rawCategories := r.PostFormValue("categories")
		rawTags := r.PostFormValue("tags")

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

		if rawCategories != "" {
			oldCategories := []string{}
			if len(scenario.Spec.Categories) != 0 {
				oldCategories = scenario.Spec.Categories
			}

			if scenario.ObjectMeta.Labels == nil {
				scenario.ObjectMeta.Labels = make(map[string]string)
			}

			for _, category := range oldCategories {
				scenario.ObjectMeta.Labels["category-"+category] = "false"
			}
			newCategoriesSlice := make([]string, 0)
			err = json.Unmarshal([]byte(rawCategories), &newCategoriesSlice)
			if err != nil {
				glog.Errorf("error while unmarshaling categories %v", err)
				util.ReturnHTTPMessage(w, r, 500, "internalerror", "error parsing")
				return fmt.Errorf("bad")
			}
			for _, category := range newCategoriesSlice {
				scenario.ObjectMeta.Labels["category-"+category] = "true"
			}
			scenario.Spec.Categories = newCategoriesSlice
		}

		if rawTags != "" {
			tagsSlice := make([]string, 0)
			err = json.Unmarshal([]byte(rawTags), &tagsSlice)
			if err != nil {
				glog.Errorf("error while unmarshaling tags %v", err)
				util.ReturnHTTPMessage(w, r, 500, "internalerror", "error parsing")
				return fmt.Errorf("bad")
			}
			scenario.Spec.Tags = tagsSlice
		}

		_, updateErr := s.hfClientSet.HobbyfarmV1().Scenarios(util.GetReleaseNamespace()).Update(s.ctx, scenario, metav1.UpdateOptions{})
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

// Filter a ScheduledEventList to find SEs that are a) active and b) using the course specified
func filterScheduledEvents(scenario string, seList *hfv1.ScheduledEventList) *[]hfv1.ScheduledEvent {
	outList := make([]hfv1.ScheduledEvent, 0)
	for _, se := range seList.Items {
		if se.Status.Finished == true {
			continue
		}

		for _, s := range se.Spec.Scenarios {
			if s == scenario {
				outList = append(outList, se)
				break
			}
		}
	}

	return &outList
}

func filterSessions(scenario string, list *hfv1.SessionList) *[]hfv1.Session {
	outList := make([]hfv1.Session, 0)
	for _, sess := range list.Items {
		if sess.Spec.ScenarioId == scenario {
			outList = append(outList, sess)
		}
	}

	return &outList
}

func filterCourses(scenario string, list *hfv1.CourseList) *[]hfv1.Course {
	outList := make([]hfv1.Course, 0)
	for _, course := range list.Items {
		for _, s := range course.Spec.Scenarios {
			if s == scenario {
				outList = append(outList, course)
				break
			}
		}
	}

	return &outList
}

func idIndexer(obj interface{}) ([]string, error) {
	scenario, ok := obj.(*hfv1.Scenario)
	if !ok {
		return []string{}, nil
	}
	return []string{scenario.Name}, nil
}
