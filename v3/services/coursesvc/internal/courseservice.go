package courseservice

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	hferrors "github.com/hobbyfarm/gargantua/v3/pkg/errors"
	"github.com/hobbyfarm/gargantua/v3/pkg/rbac"
	"github.com/hobbyfarm/gargantua/v3/pkg/util"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/wrapperspb"

	"github.com/golang/glog"
	"github.com/gorilla/mux"
	coursepb "github.com/hobbyfarm/gargantua/v3/protos/course"
	generalpb "github.com/hobbyfarm/gargantua/v3/protos/general"
	scenariopb "github.com/hobbyfarm/gargantua/v3/protos/scenario"
)

const (
	idIndex        = "courseserver.hobbyfarm.io/id-index"
	resourcePlural = rbac.ResourcePluralCourse
)

type PreparedCourse struct {
	Id                string              `json:"id"`
	Name              string              `json:"name"`
	Description       string              `json:"description"`
	Scenarios         []string            `json:"scenarios"`
	Categories        []string            `json:"categories"`
	VirtualMachines   []map[string]string `json:"virtualmachines"`
	KeepAliveDuration string              `json:"keepalive_duration"`
	PauseDuration     string              `json:"pause_duration"`
	Pauseable         bool                `json:"pauseable"`
	KeepVM            bool                `json:"keep_vm"`
	IsLearnpath       bool                `json:"is_learnpath"`
	IsLearnPathStrict bool                `json:"is_learnpath_strict"`
	DisplayInCatalog  bool                `json:"in_catalog"`
	HeaderImagePath   string              `json:"header_image_path"`
}

func convertToPreparedCourse(course *coursepb.Course) PreparedCourse {
	return PreparedCourse{
		Id:                course.GetId(),
		Name:              course.GetName(),
		Description:       course.GetDescription(),
		Scenarios:         course.GetScenarios(),
		Categories:        course.GetCategories(),
		VirtualMachines:   util.ConvertToStringMapSlice(course.GetVms()),
		KeepAliveDuration: course.GetKeepaliveDuration(),
		PauseDuration:     course.GetPauseDuration(),
		Pauseable:         course.GetPausable(),
		KeepVM:            course.GetKeepVm(),
		IsLearnpath:       course.GetIsLearnpath(),
		IsLearnPathStrict: course.GetIsLearnpathStrict(),
		DisplayInCatalog:  course.GetInCatalog(),
		HeaderImagePath:   course.GetHeaderImagePath(),
	}
}

func (c CourseServer) getPreparedCourseById(ctx context.Context, id string) (PreparedCourse, error) {
	// load course from cache
	course, err := c.internalCourseServer.GetCourse(ctx, &generalpb.GetRequest{Id: id, LoadFromCache: true})
	if err != nil {
		return PreparedCourse{}, fmt.Errorf("error while retrieving course %s", hferrors.GetErrorMessage(err))
	}

	return convertToPreparedCourse(course), nil
}

func (c CourseServer) ListFunc(w http.ResponseWriter, r *http.Request) {
	user, err := rbac.AuthenticateRequest(r, c.authnClient)
	if err != nil {
		util.ReturnHTTPMessage(w, r, 401, "unauthorized", "authentication failed")
		return
	}

	impersonatedUserId := user.GetId()
	authrResponse, err := rbac.AuthorizeSimple(r, c.authrClient, impersonatedUserId, rbac.HobbyfarmPermission(resourcePlural, rbac.VerbList))
	if err != nil || !authrResponse.Success {
		glog.Infof("Authr error: %s", err.Error())
		util.ReturnHTTPMessage(w, r, 403, "forbidden", "no access to list courses")
		return
	}

	tempCoursList, err := c.internalCourseServer.ListCourse(r.Context(), &generalpb.ListOptions{})
	if err != nil {
		glog.Errorf("error listing courses: %v", err)
		util.ReturnHTTPMessage(w, r, 500, "internalerror", "error listing courses")
		return
	}
	tempCourses := tempCoursList.GetCourses()

	courses := make([]PreparedCourse, 0, len(tempCourses))
	for _, c := range tempCourses {
		courses = append(courses, convertToPreparedCourse(c))
	}

	encodedCourses, err := json.Marshal(courses)
	if err != nil {
		glog.Errorf("error marshalling prepared courses: %v", err)
		util.ReturnHTTPMessage(w, r, 500, "internalerror", "error listing courses")
		return
	}

	util.ReturnHTTPContent(w, r, 200, "success", encodedCourses)

	glog.V(4).Infof("listed courses")
}

func (c CourseServer) GetCourse(w http.ResponseWriter, r *http.Request) {
	user, err := rbac.AuthenticateRequest(r, c.authnClient)
	if err != nil {
		util.ReturnHTTPMessage(w, r, 401, "unauthorized", "authentication failed")
		return
	}

	impersonatedUserId := user.GetId()
	authrResponse, err := rbac.AuthorizeSimple(r, c.authrClient, impersonatedUserId, rbac.HobbyfarmPermission(resourcePlural, rbac.VerbGet))
	if err != nil || !authrResponse.Success {
		util.ReturnHTTPMessage(w, r, 403, "forbidden", "no access to courses")
		return
	}

	vars := mux.Vars(r)

	courseId := vars["course_id"]
	if len(courseId) == 0 {
		util.ReturnHTTPMessage(w, r, 400, "badrequest", "no course id passed in")
		return
	}

	course, err := c.getPreparedCourseById(r.Context(), courseId)
	if err != nil {
		util.ReturnHTTPMessage(w, r, 404, "not found", fmt.Sprintf("error retrieving course: %v", err))
		return
	}

	encodedCourse, err := json.Marshal(course)
	if err != nil {
		glog.Error(err)
		util.ReturnHTTPMessage(w, r, 500, "internalerror", "error preparing course")
		return
	}

	util.ReturnHTTPContent(w, r, 200, "success", encodedCourse)
}

func (c CourseServer) CreateFunc(w http.ResponseWriter, r *http.Request) {
	user, err := rbac.AuthenticateRequest(r, c.authnClient)
	if err != nil {
		util.ReturnHTTPMessage(w, r, 401, "unauthorized", "authentication failed")
		return
	}

	impersonatedUserId := user.GetId()
	authrResponse, err := rbac.AuthorizeSimple(r, c.authrClient, impersonatedUserId, rbac.HobbyfarmPermission(resourcePlural, rbac.VerbCreate))
	if err != nil || !authrResponse.Success {
		util.ReturnHTTPMessage(w, r, 403, "forbidden", "no access to create courses")
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
	// keepaliveDuration is optional

	scenarios := r.PostFormValue("scenarios")
	// scenarios are optional

	categories := r.PostFormValue("categories")
	// categories are optional

	rawVirtualMachines := r.PostFormValue("virtualmachines")
	// virtualmachines are optional

	pauseableRaw := r.PostFormValue("pauseable")
	pauseable, err := strconv.ParseBool(pauseableRaw)
	if err != nil {
		glog.Errorf("error while parsing bool %v", err)
		util.ReturnHTTPMessage(w, r, 500, "internalerror", "error parsing")
		return
	}
	pauseDuration := r.PostFormValue("pause_duration")

	keepVMRaw := r.PostFormValue("keep_vm")
	keepVM, err := strconv.ParseBool(keepVMRaw)
	if err != nil {
		glog.Errorf("error while parsing bool: %v", err)
		util.ReturnHTTPMessage(w, r, 500, "internalerror", "error parsing")
		return
	}

	isLearnPathRaw := r.PostFormValue("is_learnpath")
	isLearnpath, err := strconv.ParseBool(isLearnPathRaw)
	if err != nil {
		glog.Errorf("error while parsing bool: %v", err)
		util.ReturnHTTPMessage(w, r, 500, "internalerror", "error parsing")
		return
	}

	isLearnPathStrictRaw := r.PostFormValue("is_learnpath_strict")
	isLearnpathStrict, err := strconv.ParseBool(isLearnPathStrictRaw)
	if err != nil {
		glog.Errorf("error while parsing bool: %v", err)
		util.ReturnHTTPMessage(w, r, 500, "internalerror", "error parsing")
		return
	}

	inCatalogRaw := r.PostFormValue("in_catalog")
	inCatalog, err := strconv.ParseBool(inCatalogRaw)
	if err != nil {
		glog.Errorf("error while parsing bool: %v", err)
		util.ReturnHTTPMessage(w, r, 500, "internalerror", "error parsing")
		return
	}

	headerImagePath := r.PostFormValue("header_image_path")

	courseId, err := c.internalCourseServer.CreateCourse(r.Context(), &coursepb.CreateCourseRequest{
		Name:              name,
		Description:       description,
		RawScenarios:      scenarios,
		RawCategories:     categories,
		RawVms:            rawVirtualMachines,
		KeepaliveDuration: keepaliveDuration,
		PauseDuration:     pauseDuration,
		Pausable:          pauseable,
		KeepVm:            keepVM,
		IsLearnpath:       isLearnpath,
		IsLearnpathStrict: isLearnpathStrict,
		InCatalog:         inCatalog,
		HeaderImagePath:   headerImagePath,
	})
	if err != nil {
		statusErr := status.Convert(err)
		if hferrors.IsGrpcParsingError(err) {
			glog.Errorf("error while parsing: %s", statusErr.Message())
			util.ReturnHTTPMessage(w, r, 500, "internalerror", "error parsing")
			return
		}
		glog.Errorf("error creating course %s", hferrors.GetErrorMessage(err))
		util.ReturnHTTPMessage(w, r, 500, "internalerror", "error creating course")
		return
	}

	util.ReturnHTTPMessage(w, r, 201, "created", courseId.GetId())
	glog.V(4).Infof("Created course %s", courseId.GetId())
}

func (c CourseServer) UpdateFunc(w http.ResponseWriter, r *http.Request) {
	user, err := rbac.AuthenticateRequest(r, c.authnClient)
	if err != nil {
		util.ReturnHTTPMessage(w, r, 401, "unauthorized", "authentication failed")
		return
	}

	impersonatedUserId := user.GetId()
	authrResponse, err := rbac.AuthorizeSimple(r, c.authrClient, impersonatedUserId, rbac.HobbyfarmPermission(resourcePlural, rbac.VerbUpdate))
	if err != nil || !authrResponse.Success {
		util.ReturnHTTPMessage(w, r, 403, "forbidden", "no access to update courses")
		return
	}

	vars := mux.Vars(r)

	id := vars["id"]
	if id == "" {
		util.ReturnHTTPMessage(w, r, 400, "badrequest", "no id passed in")
		return
	}

	name := r.PostFormValue("name")
	description := r.PostFormValue("description")
	scenarios := r.PostFormValue("scenarios")
	categories := r.PostFormValue("categories")
	virtualMachinesRaw := r.PostFormValue("virtualmachines")
	keepaliveDuration := r.PostFormValue("keepalive_duration")
	pauseDuration := r.PostFormValue("pause_duration")
	pauseableRaw := r.PostFormValue("pauseable")
	keepVMRaw := r.PostFormValue("keep_vm")
	isLearnPathRaw := r.PostFormValue("is_learnpath")
	isLearnPathStrictRaw := r.PostFormValue("is_learnpath_strict")
	inCatalogRaw := r.PostFormValue("in_catalog")
	headerImagePath := r.PostFormValue("header_image_path")

	var keepaliveWrapper *wrapperspb.StringValue
	if keepaliveDuration != "" {
		keepaliveWrapper = wrapperspb.String(keepaliveDuration)
	}

	var pauseDurationWrapper *wrapperspb.StringValue
	if pauseDuration != "" {
		pauseDurationWrapper = wrapperspb.String(pauseDuration)
	}

	var pauseable bool
	if pauseableRaw != "" {
		pauseable, err = strconv.ParseBool(pauseableRaw)
		if err != nil {
			glog.Errorf("error while parsing bool: %v", err)
			util.ReturnHTTPMessage(w, r, 500, "internalerror", "error parsing")
			return
		}
	}

	var keepVM bool
	if keepVMRaw != "" {
		keepVM, err = strconv.ParseBool(keepVMRaw)
		if err != nil {
			glog.Errorf("error while parsing bool: %v", err)
			util.ReturnHTTPMessage(w, r, 500, "internalerror", "error parsing")
			return
		}
	}

	var isLearnPath bool
	if isLearnPathRaw != "" {
		isLearnPath, err = strconv.ParseBool(isLearnPathRaw)
		if err != nil {
			glog.Errorf("error while parsing bool: %v", err)
			util.ReturnHTTPMessage(w, r, 500, "internalerror", "error parsing")
			return
		}
	}

	var isLearnPathStrict bool
	if isLearnPathStrictRaw != "" {
		isLearnPath, err = strconv.ParseBool(isLearnPathStrictRaw)
		if err != nil {
			glog.Errorf("error while parsing bool: %v", err)
			util.ReturnHTTPMessage(w, r, 500, "internalerror", "error parsing")
			return
		}
	}

	var inCatalog bool
	if inCatalogRaw != "" {
		isLearnPath, err = strconv.ParseBool(inCatalogRaw)
		if err != nil {
			glog.Errorf("error while parsing bool: %v", err)
			util.ReturnHTTPMessage(w, r, 500, "internalerror", "error parsing")
			return
		}
	}

	var headerImagePathWrapper *wrapperspb.StringValue
	if headerImagePath != "" {
		headerImagePathWrapper = wrapperspb.String(headerImagePath)
	}

	_, err = c.internalCourseServer.UpdateCourse(r.Context(), &coursepb.UpdateCourseRequest{
		Id:                id,
		Name:              name,
		Description:       description,
		RawScenarios:      scenarios,
		RawCategories:     categories,
		RawVms:            virtualMachinesRaw,
		KeepaliveDuration: keepaliveWrapper,
		PauseDuration:     pauseDurationWrapper,
		Pausable:          wrapperspb.Bool(pauseable),
		KeepVm:            wrapperspb.Bool(keepVM),
		IsLearnpath:       wrapperspb.Bool(isLearnPath),
		IsLearnpathStrict: wrapperspb.Bool(isLearnPathStrict),
		InCatalog:         wrapperspb.Bool(inCatalog),
		HeaderImagePath:   headerImagePathWrapper,
	})

	if err != nil {
		util.ReturnHTTPMessage(w, r, 500, "internalerror", "error attempting to update")
		return
	}

	util.ReturnHTTPMessage(w, r, 200, "updated", "")
	glog.V(4).Infof("Updated course %s", id)
}

func (c CourseServer) DeleteFunc(w http.ResponseWriter, r *http.Request) {
	user, err := rbac.AuthenticateRequest(r, c.authnClient)
	if err != nil {
		util.ReturnHTTPMessage(w, r, 401, "unauthorized", "authentication failed")
		return
	}

	impersonatedUserId := user.GetId()
	authrResponse, err := rbac.AuthorizeSimple(r, c.authrClient, impersonatedUserId, rbac.HobbyfarmPermission(resourcePlural, rbac.VerbDelete))
	if err != nil || !authrResponse.Success {
		util.ReturnHTTPMessage(w, r, 403, "forbidden", "no access to to delete courses")
		return
	}

	vars := mux.Vars(r)

	id := vars["id"]
	if id == "" {
		util.ReturnHTTPMessage(w, r, 400, "badrequest", "no id passed in")
		return
	}

	// when can we safely toDelete c course?
	// 1. when there are no active scheduled events using the course
	// 2. when there are no sessions using the course

	seList, err := c.scheduledEventClient.ListScheduledEvent(r.Context(), &generalpb.ListOptions{})
	if err != nil {
		glog.Errorf("error retrieving scheduledevent list: %s", hferrors.GetErrorMessage(err))
		util.ReturnHTTPMessage(w, r, 500, "internalerror", "error while deleting course")
		return
	}

	seInUse := util.FilterScheduledEvents(id, seList, util.FilterByCourse)

	sessList, err := c.sessionClient.ListSession(r.Context(), &generalpb.ListOptions{})
	if err != nil {
		glog.Errorf("error retrieving session list: %s", hferrors.GetErrorMessage(err))
		util.ReturnHTTPMessage(w, r, 500, "internalerror", "error while deleting course")
		return
	}

	sessInUse := util.FilterSessions(id, sessList, util.IsSessionOfCourse)

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

	if !toDelete {
		util.ReturnHTTPMessage(w, r, 403, "badrequest", msg)
		return
	}

	_, err = c.internalCourseServer.DeleteCourse(r.Context(), &generalpb.ResourceId{Id: id})
	if err != nil {
		glog.Errorf("error deleting course: %s", hferrors.GetErrorMessage(err))
		util.ReturnHTTPMessage(w, r, 500, "internalerror", "error while deleting course")
		return
	}

	util.ReturnHTTPMessage(w, r, 204, "deleted", "deleted successfully")
	glog.V(4).Infof("deleted course: %s", id)
}

func (c CourseServer) ListCoursesForAccesscode(w http.ResponseWriter, r *http.Request) {
	user, err := rbac.AuthenticateRequest(r, c.authnClient)
	if err != nil {
		util.ReturnHTTPMessage(w, r, 401, "unauthorized", "authentication failed")
		return
	}

	vars := mux.Vars(r)
	accessCode := vars["access_code"]

	if accessCode == "" {
		util.ReturnHTTPMessage(w, r, 400, "badrequest", "access_code is missing")
		return
	}

	contains := false
	for _, acc := range user.GetAccessCodes() {
		if acc == accessCode {
			contains = true
			break
		}
	}

	if !contains {
		util.ReturnHTTPMessage(w, r, 403, "forbidden", "no access to list scenarios for this AccessCode")
		return
	}

	tmpAccesscode, err := c.acClient.GetAccessCodeWithOTACs(r.Context(), &generalpb.ResourceId{Id: accessCode})
	if err != nil {
		glog.Errorf("error retrieving course ids for access code: %s %v", accessCode, err)
	}
	courseIds := util.UniqueStringSlice(tmpAccesscode.GetCourses())

	var courses []PreparedCourse
	for _, courseId := range courseIds {
		course, err := c.getPreparedCourseById(r.Context(), courseId)
		if err != nil {
			glog.Errorf("error retrieving course %s", hferrors.GetErrorMessage(err))
		} else {
			course.Scenarios = util.AppendDynamicScenariosByCategories(
				r.Context(),
				course.Scenarios,
				course.Categories,
				c.listScenarios,
			)
			courses = append(courses, course)
		}
	}

	encodedCourses, err := json.Marshal(courses)
	if err != nil {
		glog.Error(err)
	}
	util.ReturnHTTPContent(w, r, 200, "success", encodedCourses)
}

func (c CourseServer) ListCourseCatalog(w http.ResponseWriter, r *http.Request) {
	_, err := rbac.AuthenticateRequest(r, c.authnClient)
	if err != nil {
		util.ReturnHTTPMessage(w, r, 401, "unauthorized", "authentication failed")
		return
	}

	tempCoursList, err := c.internalCourseServer.ListCourse(r.Context(), &generalpb.ListOptions{})
	if err != nil {
		glog.Errorf("error listing courses: %v", err)
		util.ReturnHTTPMessage(w, r, 500, "internalerror", "error listing courses")
		return
	}
	tempCourses := tempCoursList.GetCourses()

	courses := make([]PreparedCourse, 0, len(tempCourses))
	for _, course := range tempCourses {
		if course.InCatalog {
			course.Scenarios = util.AppendDynamicScenariosByCategories(
				r.Context(),
				course.Scenarios,
				course.Categories,
				c.listScenarios,
			)
			courses = append(courses, convertToPreparedCourse(course))
		}
	}

	encodedCourses, err := json.Marshal(courses)
	if err != nil {
		glog.Errorf("error marshalling prepared courses: %v", err)
		util.ReturnHTTPMessage(w, r, 500, "internalerror", "error listing courses")
		return
	}

	util.ReturnHTTPContent(w, r, 200, "success", encodedCourses)

	glog.V(4).Infof("listed courses")
}

func (c CourseServer) previewDynamicScenarios(w http.ResponseWriter, r *http.Request) {
	user, err := rbac.AuthenticateRequest(r, c.authnClient)
	if err != nil {
		util.ReturnHTTPMessage(w, r, 401, "unauthorized", "authentication failed")
		return
	}

	impersonatedUserId := user.GetId()
	authrResponse, err := rbac.AuthorizeSimple(r, c.authrClient, impersonatedUserId, rbac.HobbyfarmPermission(rbac.ResourcePluralScenario, rbac.VerbList))
	if err != nil || !authrResponse.Success {
		util.ReturnHTTPMessage(w, r, 403, "forbidden", "no access to preview dynamic scenarios")
		return
	}

	categories := r.PostFormValue("categories")
	categoriesSlice := make([]string, 0)
	if categories != "" {
		err = json.Unmarshal([]byte(categories), &categoriesSlice)
		if err != nil {
			glog.Errorf("error while unmarshalling categories %v", err)
			util.ReturnHTTPMessage(w, r, 500, "internalerror", "error parsing")
			return
		}
	}

	scenarios := []string{}

	scenarios = util.AppendDynamicScenariosByCategories(r.Context(), scenarios, categoriesSlice, c.listScenarios)

	encodedScenarios, err := json.Marshal(scenarios)
	if err != nil {
		glog.Error(err)
	}
	util.ReturnHTTPContent(w, r, 200, "success", encodedScenarios)
}

// We need this helper function because util.AppendDynamicScenariosByCategories expects a list function without grpc call options
func (c CourseServer) listScenarios(ctx context.Context, listOptions *generalpb.ListOptions) (*scenariopb.ListScenariosResponse, error) {
	return c.scenarioClient.ListScenario(ctx, listOptions)
}
