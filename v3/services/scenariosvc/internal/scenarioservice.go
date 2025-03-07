package scenarioservice

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"slices"
	"strconv"
	"strings"

	hferrors "github.com/hobbyfarm/gargantua/v3/pkg/errors"
	"github.com/hobbyfarm/gargantua/v3/pkg/rbac"
	"github.com/hobbyfarm/gargantua/v3/pkg/util"
	"google.golang.org/protobuf/types/known/wrapperspb"

	accesscodepb "github.com/hobbyfarm/gargantua/v3/protos/accesscode"
	authrpb "github.com/hobbyfarm/gargantua/v3/protos/authr"
	coursepb "github.com/hobbyfarm/gargantua/v3/protos/course"
	generalpb "github.com/hobbyfarm/gargantua/v3/protos/general"
	scenariopb "github.com/hobbyfarm/gargantua/v3/protos/scenario"
	scheduledeventpb "github.com/hobbyfarm/gargantua/v3/protos/scheduledevent"

	"github.com/golang/glog"
	"github.com/gorilla/mux"
)

const (
	idIndex        = "scenarioserver.hobbyfarm.io/id-index"
	resourcePlural = rbac.ResourcePluralScenario
)

type PreparedScenarioStep struct {
	Title   string `json:"title"`
	Content string `json:"content"`
	Quiz    string `json:"quiz"`
}

type PreparedScenario struct {
	Id              string                            `json:"id"`
	Name            string                            `json:"name"`
	Description     string                            `json:"description"`
	StepCount       int                               `json:"stepcount"`
	VirtualMachines []map[string]string               `json:"virtualmachines"`
	Pauseable       bool                              `json:"pauseable"`
	Printable       bool                              `json:"printable"`
	Tasks           []*scenariopb.VirtualMachineTasks `json:"vm_tasks"`
}

type AdminPreparedScenario struct {
	ID                string                            `json:"id"`
	Name              string                            `json:"name"`
	Description       string                            `json:"description"`
	Steps             []*scenariopb.ScenarioStep        `json:"steps"`
	Categories        []string                          `json:"categories"`
	Tags              []string                          `json:"tags"`
	VirtualMachines   []map[string]string               `json:"virtualmachines"`
	KeepAliveDuration string                            `json:"keepalive_duration"`
	PauseDuration     string                            `json:"pause_duration"`
	Pauseable         bool                              `json:"pauseable"`
	Tasks             []*scenariopb.VirtualMachineTasks `json:"vm_tasks"`
}

func (s ScenarioServer) prepareScenario(scenario *scenariopb.Scenario, printable bool) PreparedScenario {
	return PreparedScenario{
		Id:              scenario.GetId(),
		Name:            scenario.GetName(),
		Description:     scenario.GetDescription(),
		VirtualMachines: util.ConvertToStringMapSlice(scenario.GetVms()),
		Pauseable:       scenario.GetPausable(),
		Printable:       printable,
		StepCount:       len(scenario.GetSteps()),
		Tasks:           scenario.GetVmTasks(),
	}
}

func (s ScenarioServer) getPreparedScenarioStepById(ctx context.Context, id string, step int) (PreparedScenarioStep, error) {
	scenario, err := s.internalScenarioServer.GetScenario(ctx, &generalpb.GetRequest{Id: id, LoadFromCache: true})
	if err != nil {
		return PreparedScenarioStep{}, fmt.Errorf("error while retrieving scenario step")
	}

	if step >= 0 && len(scenario.GetSteps()) > step {
		stepContent := scenario.GetSteps()[step]
		return PreparedScenarioStep{stepContent.GetTitle(), stepContent.GetContent(), stepContent.GetQuiz()}, nil
	}

	return PreparedScenarioStep{}, fmt.Errorf("error while retrieving scenario step, most likely doesn't exist in cache")
}

func (s ScenarioServer) getPrintableScenarioIds(ctx context.Context, accessCodes []string) []string {
	var printableScenarioIds []string
	var printableCourseIds []string
	accessCodeList, err := s.acClient.GetAccessCodesWithOTACs(ctx, &accesscodepb.ResourceIds{Ids: accessCodes})
	if err != nil {
		glog.Errorf("error retrieving access codes: %s", hferrors.GetErrorMessage(err))
		return []string{}
	}
	for _, accessCode := range accessCodeList.GetAccessCodes() {
		if !accessCode.GetPrintable() {
			continue
		}
		printableScenarioIds = append(printableScenarioIds, accessCode.GetScenarios()...)
		printableCourseIds = append(printableCourseIds, accessCode.GetCourses()...)
	}
	printableCourseIds = util.UniqueStringSlice(printableCourseIds)

	for _, courseId := range printableCourseIds {
		course, err := s.courseClient.GetCourse(ctx, &generalpb.GetRequest{Id: courseId, LoadFromCache: true})
		if err != nil {
			glog.Errorf("error retrieving course %s", hferrors.GetErrorMessage(err))
			continue
		}
		dynamicScenarios := util.AppendDynamicScenariosByCategories(
			ctx,
			course.GetScenarios(),
			course.GetCategories(),
			s.internalScenarioServer.ListScenario,
		)
		printableScenarioIds = append(printableScenarioIds, dynamicScenarios...)
	}

	printableScenarioIds = util.UniqueStringSlice(printableScenarioIds)
	return printableScenarioIds
}

func (s ScenarioServer) getPreparedScenarioById(ctx context.Context, id string, accessCodes []string) (PreparedScenario, error) {
	scenario, err := s.internalScenarioServer.GetScenario(ctx, &generalpb.GetRequest{Id: id, LoadFromCache: true})
	if err != nil {
		return PreparedScenario{}, fmt.Errorf("error while retrieving scenario: %s", hferrors.GetErrorMessage(err))
	}

	printableScenarioIds := s.getPrintableScenarioIds(ctx, accessCodes)
	printable := slices.Contains(printableScenarioIds, scenario.GetId())

	preparedScenario := s.prepareScenario(scenario, printable)

	return preparedScenario, nil
}

func (s ScenarioServer) GetScenarioFunc(w http.ResponseWriter, r *http.Request) {
	user, err := rbac.AuthenticateRequest(r, s.authnClient)
	if err != nil {
		util.ReturnHTTPMessage(w, r, 403, "forbidden", "no access to get scenarios")
		return
	}

	vars := mux.Vars(r)

	scenario_id := vars["scenario_id"]

	if len(scenario_id) == 0 {
		util.ReturnHTTPMessage(w, r, 400, "badrequest", "no scenario id passed in")
		return
	}

	scenario, err := s.getPreparedScenarioById(r.Context(), scenario_id, user.AccessCodes)
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
	user, err := rbac.AuthenticateRequest(r, s.authnClient)
	if err != nil {
		util.ReturnHTTPMessage(w, r, 401, "unauthorized", "authentication failed")
		return
	}

	impersonatedUserId := user.GetId()
	authrResponse, err := rbac.AuthorizeSimple(r, s.authrClient, impersonatedUserId, rbac.HobbyfarmPermission(resourcePlural, rbac.VerbGet))
	if err != nil || !authrResponse.Success {
		util.ReturnHTTPMessage(w, r, 403, "forbidden", "no access to get Scenario")
		return
	}

	vars := mux.Vars(r)

	id := vars["id"]

	if len(id) == 0 {
		util.ReturnHTTPMessage(w, r, 400, "bad request", "no id passed in")
		return
	}

	scenario, err := s.internalScenarioServer.GetScenario(r.Context(), &generalpb.GetRequest{
		Id:            id,
		LoadFromCache: true,
	})

	if err != nil {
		glog.Errorf("error while retrieving scenario %s", hferrors.GetErrorMessage(err))
		util.ReturnHTTPMessage(w, r, 500, "error", "no scenario found")
		return
	}

	preparedScenario := AdminPreparedScenario{
		ID:                scenario.GetId(),
		Name:              scenario.GetName(),
		Description:       scenario.GetDescription(),
		Steps:             scenario.GetSteps(),
		Categories:        scenario.GetCategories(),
		Tags:              scenario.GetTags(),
		VirtualMachines:   util.ConvertToStringMapSlice(scenario.GetVms()),
		KeepAliveDuration: scenario.GetKeepaliveDuration(),
		PauseDuration:     scenario.GetPauseDuration(),
		Pauseable:         scenario.GetPausable(),
		Tasks:             scenario.GetVmTasks(),
	}

	encodedScenario, err := json.Marshal(preparedScenario)
	if err != nil {
		glog.Error(err)
	}
	util.ReturnHTTPContent(w, r, 200, "success", encodedScenario)

	glog.V(2).Infof("retrieved scenario %s", scenario.Name)
}

func (s ScenarioServer) AdminDeleteFunc(w http.ResponseWriter, r *http.Request) {
	user, err := rbac.AuthenticateRequest(r, s.authnClient)
	if err != nil {
		util.ReturnHTTPMessage(w, r, 401, "unauthorized", "authentication failed")
		return
	}

	impersonatedUserId := user.GetId()
	authrResponse, err := rbac.AuthorizeSimple(r, s.authrClient, impersonatedUserId, rbac.HobbyfarmPermission(resourcePlural, rbac.VerbDelete))
	if err != nil || !authrResponse.Success {
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

	seList, err := s.scheduledEventClient.ListScheduledEvent(r.Context(), &generalpb.ListOptions{})
	if err != nil {
		glog.Errorf("error retrieving scheduledevent list: %s", hferrors.GetErrorMessage(err))
		util.ReturnHTTPMessage(w, r, 500, "internalerror", "error while deleting scenario")
		return
	}

	seInUse := util.FilterScheduledEvents(id, seList, util.FilterByScenario[*scheduledeventpb.ScheduledEvent])

	sessList, err := s.sessionClient.ListSession(r.Context(), &generalpb.ListOptions{})
	if err != nil {
		glog.Errorf("error retrieving session list: %s", hferrors.GetErrorMessage(err))
		util.ReturnHTTPMessage(w, r, 500, "internalerror", "error while deleting scenario")
		return
	}

	sessInUse := util.FilterSessions(id, sessList, util.IsSessionOfScenario)

	courseList, err := s.courseClient.ListCourse(r.Context(), &generalpb.ListOptions{})
	if err != nil {
		glog.Errorf("error retrieving course list: %s", hferrors.GetErrorMessage(err))
		util.ReturnHTTPMessage(w, r, 500, "internalerror", "error while deleting scenario")
		return
	}

	coursesInUse := filterCourses(id, courseList)

	var msg = ""
	toDelete := true

	if len(seInUse) > 0 {
		// cannot toDelete, in use. alert the user
		msg += "In use by scheduled events:"
		for _, se := range seInUse {
			msg += " " + se.GetId()
		}
		toDelete = false
	}

	if len(sessInUse) > 0 {
		msg += "In use by sessions:"
		for _, sess := range sessInUse {
			msg += " " + sess.GetId()
		}
		toDelete = false
	}

	if len(coursesInUse) > 0 {
		msg += "In use by courses:"
		for _, course := range coursesInUse {
			msg += " " + course.GetId()
		}
		toDelete = false
	}

	if !toDelete {
		util.ReturnHTTPMessage(w, r, 403, "badrequest", msg)
		return
	}

	_, err = s.internalScenarioServer.DeleteScenario(r.Context(), &generalpb.ResourceId{Id: id})

	if err != nil {
		glog.Errorf("error while deleting scenario %s", hferrors.GetErrorMessage(err))
		util.ReturnHTTPMessage(w, r, 500, "error", "scenario could not be deleted")
		return
	}
	util.ReturnHTTPMessage(w, r, 200, "success", "scenario deleted")
	glog.V(2).Infof("deleted scenario %s", id)
}

func (s ScenarioServer) GetScenarioStepFunc(w http.ResponseWriter, r *http.Request) {
	_, err := rbac.AuthenticateRequest(r, s.authnClient)
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
	step, err := s.getPreparedScenarioStepById(r.Context(), vars["scenario_id"], stepId)
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
	user, err := rbac.AuthenticateRequest(r, s.authnClient)
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

	if !slices.Contains(user.AccessCodes, accessCode) {

		util.ReturnHTTPMessage(w, r, 403, "forbidden", "no access to list scenarios for this AccessCode")
		return
	}

	// store a list of scenarios linked to courses for filtering
	//var courseScenarios []string
	var scenarioIds []string
	ac, err := s.acClient.GetAccessCodeWithOTACs(r.Context(), &generalpb.ResourceId{Id: accessCode})
	if err != nil {
		glog.Errorf("error retrieving access code %s: %s", accessCode, hferrors.GetErrorMessage(err))
	}
	scenarioIds = append(scenarioIds, ac.GetScenarios()...)

	var scenarios []PreparedScenario
	for _, scenarioId := range scenarioIds {
		scenario, err := s.internalScenarioServer.GetScenario(r.Context(), &generalpb.GetRequest{
			Id:            scenarioId,
			LoadFromCache: true,
		})
		if err != nil {
			glog.Errorf("error retrieving scenario: %s", hferrors.GetErrorMessage(err))
			continue
		}
		pScenario := s.prepareScenario(scenario, ac.GetPrintable())
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
	user, err := rbac.AuthenticateRequest(r, s.authnClient)
	if err != nil {
		util.ReturnHTTPMessage(w, r, 401, "unauthorized", "authentication failed")
		return
	}

	impersonatedUserId := user.GetId()
	authrResponse, err := rbac.AuthorizeSimple(r, s.authrClient, impersonatedUserId, rbac.HobbyfarmPermission(resourcePlural, rbac.VerbList))
	if err != nil || !authrResponse.Success {
		util.ReturnHTTPMessage(w, r, 403, "forbidden", "no access to list scenarios")
		return
	}

	categorySelector := &generalpb.ListOptions{}
	if category != "" {
		categorySelector = &generalpb.ListOptions{
			LabelSelector: fmt.Sprintf("category-%s=true", category),
		}
	}

	scenarioList, err := s.internalScenarioServer.ListScenario(r.Context(), categorySelector)

	if err != nil {
		glog.Errorf("error while retrieving scenarios %s", hferrors.GetErrorMessage(err))
		util.ReturnHTTPMessage(w, r, 500, "error", "no scenarios found")
		return
	}

	preparedScenarios := []AdminPreparedScenario{}
	for _, scenario := range scenarioList.GetScenarios() {
		pScenario := AdminPreparedScenario{
			ID:                scenario.GetId(),
			Name:              scenario.GetName(),
			Description:       scenario.GetDescription(),
			Steps:             nil,
			Categories:        scenario.GetCategories(),
			Tags:              scenario.GetTags(),
			VirtualMachines:   util.ConvertToStringMapSlice(scenario.GetVms()),
			KeepAliveDuration: scenario.GetKeepaliveDuration(),
			PauseDuration:     scenario.GetPauseDuration(),
			Pauseable:         scenario.GetPausable(),
			Tasks:             scenario.GetVmTasks(),
		}
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
	user, err := rbac.AuthenticateRequest(r, s.authnClient)
	if err != nil {
		util.ReturnHTTPMessage(w, r, 401, "unauthorized", "authentication failed")
		return
	}

	impersonatedUserId := user.GetId()
	authrResponse, err := rbac.AuthorizeSimple(r, s.authrClient, impersonatedUserId, rbac.HobbyfarmPermission(resourcePlural, rbac.VerbList))
	if err != nil || !authrResponse.Success {
		util.ReturnHTTPMessage(w, r, 403, "forbidden", "no access to list categories")
		return
	}

	scenarioList, err := s.internalScenarioServer.ListScenario(r.Context(), &generalpb.ListOptions{})

	if err != nil {
		glog.Errorf("error while retrieving scenarios %s", hferrors.GetErrorMessage(err))
		util.ReturnHTTPMessage(w, r, 500, "error", "no scenarios found")
		return
	}

	categorySlice := []string{}

	for _, scenario := range scenarioList.GetScenarios() {
		categories := scenario.GetCategories()
		if len(categories) != 0 {
			categorySlice = append(categorySlice, categories...)
		}
	}

	// Sort + Compact creates a unique sorted slice
	slices.Sort(categorySlice)
	slices.Compact(categorySlice)

	encodedCategories, err := json.Marshal(categorySlice)
	if err != nil {
		glog.Error(err)
	}
	util.ReturnHTTPContent(w, r, 200, "success", encodedCategories)

	glog.V(2).Infof("listed categories")
}

func (s ScenarioServer) AdminPrintFunc(w http.ResponseWriter, r *http.Request) {
	user, err := rbac.AuthenticateRequest(r, s.authnClient)
	if err != nil {
		util.ReturnHTTPMessage(w, r, 401, "unauthorized", "authentication failed")
		return
	}

	impersonatedUserId := user.GetId()
	authrResponse, err := rbac.AuthorizeSimple(r, s.authrClient, impersonatedUserId, rbac.HobbyfarmPermission(resourcePlural, rbac.VerbGet))
	if err != nil || !authrResponse.Success {
		util.ReturnHTTPMessage(w, r, 403, "forbidden", "no access to get Scenario")
		return
	}

	vars := mux.Vars(r)

	id := vars["id"]

	if len(id) == 0 {
		util.ReturnHTTPMessage(w, r, 400, "bad request", "no id passed in")
		return
	}

	scenario, err := s.internalScenarioServer.GetScenario(r.Context(), &generalpb.GetRequest{
		Id:            id,
		LoadFromCache: true,
	})

	if err != nil {
		glog.Errorf("error while retrieving scenario: %s", hferrors.GetErrorMessage(err))
		util.ReturnHTTPMessage(w, r, 500, "error", "no scenario found")
		return
	}

	content := preparePrintableContent(scenario)

	util.ReturnHTTPRaw(w, r, content)

	glog.V(2).Infof("retrieved scenario and rendered for printability %s", id)
}

func (s ScenarioServer) PrintFunc(w http.ResponseWriter, r *http.Request) {
	user, err := rbac.AuthenticateRequest(r, s.authnClient)
	if err != nil {
		util.ReturnHTTPMessage(w, r, 403, "forbidden", "no access to get Scenario")
		return
	}

	vars := mux.Vars(r)

	id := vars["id"]

	if len(id) == 0 {
		util.ReturnHTTPMessage(w, r, 400, "bad request", "no id passed in")
		return
	}

	printableScenarioIds := s.getPrintableScenarioIds(r.Context(), user.GetAccessCodes())

	if !slices.Contains(printableScenarioIds, id) {
		util.ReturnHTTPMessage(w, r, 403, "forbidden", "no access to get this Scenario")
		return
	}

	scenario, err := s.internalScenarioServer.GetScenario(r.Context(), &generalpb.GetRequest{
		Id:            id,
		LoadFromCache: true,
	})

	if err != nil {
		glog.Errorf("error while retrieving scenario %s: %s", id, hferrors.GetErrorMessage(err))
		util.ReturnHTTPMessage(w, r, 500, "error", "no scenario found")
		return
	}

	content := preparePrintableContent(scenario)

	util.ReturnHTTPRaw(w, r, content)

	glog.V(2).Infof("retrieved scenario and rendered for printability %s", id)
}

func (s ScenarioServer) CopyFunc(w http.ResponseWriter, r *http.Request) {
	user, err := rbac.AuthenticateRequest(r, s.authnClient)
	if err != nil {
		util.ReturnHTTPMessage(w, r, 401, "unauthorized", "authentication failed")
		return
	}

	impersonatedUserId := user.GetId()
	authrResponse, err := rbac.Authorize(r, s.authrClient, impersonatedUserId, []*authrpb.Permission{
		rbac.HobbyfarmPermission(resourcePlural, rbac.VerbCreate),
		rbac.HobbyfarmPermission(resourcePlural, rbac.VerbGet),
	}, rbac.OperatorAND)
	if err != nil || !authrResponse.Success {
		util.ReturnHTTPMessage(w, r, 403, "forbidden", "no access to create scenarios")
		return
	}

	vars := mux.Vars(r)

	id := vars["id"]

	if len(id) == 0 {
		util.ReturnHTTPMessage(w, r, 400, "bad request", "no id passed in")
		return
	}

	_, err = s.internalScenarioServer.CopyScenario(r.Context(), &generalpb.ResourceId{Id: id})
	if err != nil {
		glog.Error(hferrors.GetErrorMessage(err))
		if hferrors.IsGrpcNotFound(err) {
			util.ReturnHTTPMessage(w, r, 404, "not found", fmt.Sprintf("error attempting to copy: scenario %s not found", id))
			return
		} else {
			util.ReturnHTTPMessage(w, r, 500, "internalerror", "error attempting to copy")
			return
		}
	}
	util.ReturnHTTPMessage(w, r, 200, "copied scenario", "")
}

func (s ScenarioServer) CreateFunc(w http.ResponseWriter, r *http.Request) {
	user, err := rbac.AuthenticateRequest(r, s.authnClient)
	if err != nil {
		util.ReturnHTTPMessage(w, r, 401, "unauthorized", "authentication failed")
		return
	}

	impersonatedUserId := user.GetId()
	authrResponse, err := rbac.AuthorizeSimple(r, s.authrClient, impersonatedUserId, rbac.HobbyfarmPermission(resourcePlural, rbac.VerbCreate))
	if err != nil || !authrResponse.Success {
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

	rawSteps := r.PostFormValue("steps")
	rawCategories := r.PostFormValue("categories")
	rawTags := r.PostFormValue("tags")
	rawVirtualMachines := r.PostFormValue("virtualmachines")
	rawVMTasks := r.PostFormValue("vm_tasks")

	pauseable := r.PostFormValue("pauseable")
	pauseableBool := false
	if pauseable != "" {
		if strings.ToLower(pauseable) == "true" {
			pauseableBool = true
		}
	}
	pauseDuration := r.PostFormValue("pause_duration")

	scenarioId, err := s.internalScenarioServer.CreateScenario(r.Context(), &scenariopb.CreateScenarioRequest{
		Name:              name,
		Description:       description,
		RawSteps:          rawSteps,
		RawCategories:     rawCategories,
		RawTags:           rawTags,
		RawVms:            rawVirtualMachines,
		RawVmTasks:        rawVMTasks,
		KeepaliveDuration: keepaliveDuration,
		PauseDuration:     pauseDuration,
		Pausable:          pauseableBool,
	})
	if err != nil {
		glog.Errorf("error creating scenario %s", hferrors.GetErrorMessage(err))
		util.ReturnHTTPMessage(w, r, 500, "internalerror", "error creating scenario")
		return
	}

	util.ReturnHTTPMessage(w, r, 201, "created", scenarioId.GetId())
}

func (s ScenarioServer) UpdateFunc(w http.ResponseWriter, r *http.Request) {
	user, err := rbac.AuthenticateRequest(r, s.authnClient)
	if err != nil {
		util.ReturnHTTPMessage(w, r, 401, "unauthorized", "authentication failed")
		return
	}

	impersonatedUserId := user.GetId()
	authrResponse, err := rbac.AuthorizeSimple(r, s.authrClient, impersonatedUserId, rbac.HobbyfarmPermission(resourcePlural, rbac.VerbUpdate))
	if err != nil || !authrResponse.Success {
		util.ReturnHTTPMessage(w, r, 403, "forbidden", "no access to update scenarios")
		return
	}

	vars := mux.Vars(r)

	id := vars["id"]
	if id == "" {
		util.ReturnHTTPMessage(w, r, 400, "badrequest", "no ID passed in")
		return
	}

	name := r.PostFormValue("name")
	description := r.PostFormValue("description")
	rawSteps := r.PostFormValue("steps")
	pauseable := r.PostFormValue("pauseable")
	pauseableBool := false
	if pauseable != "" {
		if strings.ToLower(pauseable) == "true" {
			pauseableBool = true
		}
	}
	pauseDuration := r.PostFormValue("pause_duration")
	keepaliveDuration := r.PostFormValue("keepalive_duration")
	rawVirtualMachines := r.PostFormValue("virtualmachines")
	rawCategories := r.PostFormValue("categories")
	rawTags := r.PostFormValue("tags")
	rawVMTasks := r.PostFormValue("vm_tasks")

	_, err = s.internalScenarioServer.UpdateScenario(r.Context(), &scenariopb.UpdateScenarioRequest{
		Id:                id,
		Name:              name,
		Description:       description,
		RawSteps:          rawSteps,
		RawCategories:     rawCategories,
		RawTags:           rawTags,
		RawVms:            rawVirtualMachines,
		RawVmTasks:        rawVMTasks,
		KeepaliveDuration: wrapperspb.String(keepaliveDuration),
		PauseDuration:     wrapperspb.String(pauseDuration),
		Pausable:          wrapperspb.Bool(pauseableBool),
	})

	if err != nil {
		util.ReturnHTTPMessage(w, r, 500, "error", "error attempting to update")
		return
	}

	util.ReturnHTTPMessage(w, r, 200, "updated", "")
}

func filterCourses(scenario string, courseList *coursepb.ListCoursesResponse) []*coursepb.Course {
	outList := make([]*coursepb.Course, 0)
	for _, course := range courseList.GetCourses() {
		if util.FilterByScenario(course, scenario) {
			outList = append(outList, course)
		}
	}

	return outList
}

func preparePrintableContent(scenario *scenariopb.Scenario) string {
	id := scenario.GetId()
	var content string

	name, err := base64.StdEncoding.DecodeString(scenario.GetName())
	if err != nil {
		glog.Errorf("Error decoding title of scenario %s: %v", id, err)
	}
	description, err := base64.StdEncoding.DecodeString(scenario.GetDescription())
	if err != nil {
		glog.Errorf("Error decoding description of scenario %s: %v", id, err)
	}

	content = fmt.Sprintf("# %s\n%s\n\n", name, description)

	for i, s := range scenario.GetSteps() {

		title, err := base64.StdEncoding.DecodeString(s.GetTitle())
		if err != nil {
			glog.Errorf("Error decoding title of scenario: %s step %d: %v", id, i, err)
		}

		content = content + fmt.Sprintf("## Step %d: %s\n", i+1, string(title))

		stepContent, err := base64.StdEncoding.DecodeString(s.GetContent())
		if err != nil {
			glog.Errorf("Error decoding content of scenario: %s step %d: %v", id, i, err)
		}

		content = content + fmt.Sprintf("%s\n", string(stepContent))
	}
	return content
}
