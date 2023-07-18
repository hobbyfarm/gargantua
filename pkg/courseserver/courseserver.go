package courseserver

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/hobbyfarm/gargantua/v3/pkg/rbacclient"
	"net/http"
	"strconv"
	"strings"

	"github.com/golang/glog"
	"github.com/gorilla/mux"
	"github.com/hobbyfarm/gargantua/v3/pkg/accesscode"
	hfv1 "github.com/hobbyfarm/gargantua/v3/pkg/apis/hobbyfarm.io/v1"
	"github.com/hobbyfarm/gargantua/v3/pkg/authclient"
	hfClientset "github.com/hobbyfarm/gargantua/v3/pkg/client/clientset/versioned"
	hfInformers "github.com/hobbyfarm/gargantua/v3/pkg/client/informers/externalversions"
	"github.com/hobbyfarm/gargantua/v3/pkg/util"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/retry"
)

const (
	idIndex        = "courseserver.hobbyfarm.io/id-index"
	resourcePlural = "courses"
)

type CourseServer struct {
	auth          *authclient.AuthClient
	hfClientSet   hfClientset.Interface
	acClient      *accesscode.AccessCodeClient
	courseIndexer cache.Indexer
	ctx           context.Context
}

type PreparedCourse struct {
	Id string `json:"id"`
	hfv1.CourseSpec
}

func NewCourseServer(authClient *authclient.AuthClient, acClient *accesscode.AccessCodeClient, hfClientset hfClientset.Interface, hfInformerFactory hfInformers.SharedInformerFactory, ctx context.Context) (*CourseServer, error) {
	course := CourseServer{}

	course.hfClientSet = hfClientset
	course.acClient = acClient
	course.auth = authClient
	inf := hfInformerFactory.Hobbyfarm().V1().Courses().Informer()
	indexers := map[string]cache.IndexFunc{idIndex: idIndexer}

	err := inf.AddIndexers(indexers)
	if err != nil {
		glog.Errorf("error adding indexer %s for courses", idIndex)
	}
	course.courseIndexer = inf.GetIndexer()
	course.ctx = ctx

	return &course, nil
}

func (c CourseServer) SetupRoutes(r *mux.Router) {
	r.HandleFunc("/course/list/{access_code}", c.ListCoursesForAccesscode).Methods("GET")
	r.HandleFunc("/course/{course_id}", c.GetCourse).Methods("GET")
	r.HandleFunc("/a/course/list", c.ListFunc).Methods("GET")
	r.HandleFunc("/a/course/new", c.CreateFunc).Methods("POST")
	r.HandleFunc("/a/course/{course_id}", c.GetCourse).Methods("GET")
	r.HandleFunc("/a/course/{id}", c.UpdateFunc).Methods("PUT")
	r.HandleFunc("/a/course/{id}", c.DeleteFunc).Methods("DELETE")
	r.HandleFunc("/a/course/previewDynamicScenarios", c.previewDynamicScenarios).Methods("POST")
}

func (c CourseServer) getPreparedCourseById(id string) (PreparedCourse, error) {
	course, err := c.GetCourseById(id)

	if err != nil {
		return PreparedCourse{}, fmt.Errorf("error while retrieving course %v", err)
	}

	preparedCourse := PreparedCourse{course.Name, course.Spec}

	return preparedCourse, nil
}

func (c CourseServer) ListFunc(w http.ResponseWriter, r *http.Request) {
	_, err := c.auth.AuthGrant(rbacclient.RbacRequest().HobbyfarmPermission(resourcePlural, rbacclient.VerbList), w, r)
	if err != nil {
		util.ReturnHTTPMessage(w, r, 403, "forbidden", "no access to list courses")
		return
	}

	tempCourses, err := c.hfClientSet.HobbyfarmV1().Courses(util.GetReleaseNamespace()).List(c.ctx, metav1.ListOptions{})
	if err != nil {
		glog.Errorf("error listing courses: %v", err)
		util.ReturnHTTPMessage(w, r, 500, "internalerror", "error listing courses")
		return
	}

	var courses []PreparedCourse
	for _, c := range tempCourses.Items {
		courses = append(courses, PreparedCourse{c.Name, c.Spec})
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
	_, err := c.auth.AuthGrant(rbacclient.RbacRequest().HobbyfarmPermission(resourcePlural, rbacclient.VerbGet), w, r)
	if err != nil {
		util.ReturnHTTPMessage(w, r, 403, "forbidden", "no access to courses")
		return
	}

	vars := mux.Vars(r)

	course, err := c.getPreparedCourseById(vars["course_id"])
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
	_, err := c.auth.AuthGrant(rbacclient.RbacRequest().HobbyfarmPermission(resourcePlural, rbacclient.VerbCreate), w, r)
	if err != nil {
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
	scenarioSlice := make([]string, 0)
	if scenarios != "" {
		err = json.Unmarshal([]byte(scenarios), &scenarioSlice)
		if err != nil {
			glog.Errorf("error while unmarshalling scenarios %v", err)
			util.ReturnHTTPMessage(w, r, 500, "internalerror", "error parsing")
			return
		}
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

	rawVirtualMachines := r.PostFormValue("virtualmachines")
	virtualmachines := []map[string]string{} // must be declared this way so as to JSON marshal into [] instead of null
	if rawVirtualMachines != "" {
		err = json.Unmarshal([]byte(rawVirtualMachines), &virtualmachines)
		if err != nil {
			glog.Errorf("error while unmarshaling VMs %v", err)
			util.ReturnHTTPMessage(w, r, 500, "internalerror", "error parsing")
			return
		}
	}

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

	course := &hfv1.Course{}

	generatedName := util.GenerateResourceName("c", name, 10)

	course.Name = generatedName

	course.Spec.Name = name
	course.Spec.Description = description
	course.Spec.VirtualMachines = virtualmachines
	course.Spec.Scenarios = scenarioSlice
	course.Spec.Categories = categoriesSlice
	if keepaliveDuration != "" {
		course.Spec.KeepAliveDuration = keepaliveDuration
	}
	course.Spec.Pauseable = pauseable
	if pauseDuration != "" {
		course.Spec.PauseDuration = pauseDuration
	}
	course.Spec.KeepVM = keepVM

	course, err = c.hfClientSet.HobbyfarmV1().Courses(util.GetReleaseNamespace()).Create(c.ctx, course, metav1.CreateOptions{})
	if err != nil {
		glog.Errorf("error creating course %v", err)
		util.ReturnHTTPMessage(w, r, 500, "internalerror", "error creating course")
		return
	}

	util.ReturnHTTPMessage(w, r, 201, "created", course.Name)
	glog.V(4).Infof("Created course %s", course.Name)
	return
}

func (c CourseServer) UpdateFunc(w http.ResponseWriter, r *http.Request) {
	_, err := c.auth.AuthGrant(rbacclient.RbacRequest().HobbyfarmPermission(resourcePlural, rbacclient.VerbUpdate), w, r)
	if err != nil {
		util.ReturnHTTPMessage(w, r, 403, "forbidden", "no access to update scenarios")
		return
	}

	vars := mux.Vars(r)

	id := vars["id"]
	if id == "" {
		util.ReturnHTTPMessage(w, r, 400, "badrequest", "no id passed in")
		return
	}

	retryErr := retry.RetryOnConflict(retry.DefaultRetry, func() error {
		course, err := c.hfClientSet.HobbyfarmV1().Courses(util.GetReleaseNamespace()).Get(c.ctx, id, metav1.GetOptions{})
		if err != nil {
			glog.Error(err)
			util.ReturnHTTPMessage(w, r, http.StatusNotFound, "badrequest", "no course found with given ID")
			return fmt.Errorf("bad")
		}
		// name, description, scenarios, virtualmachines, keepaliveduration, pauseduration, pauseable

		name := r.PostFormValue("name")
		description := r.PostFormValue("description")
		scenarios := r.PostFormValue("scenarios")
		categories := r.PostFormValue("categories")
		virtualMachinesRaw := r.PostFormValue("virtualmachines")
		keepaliveDuration := r.PostFormValue("keepalive_duration")
		pauseDuration := r.PostFormValue("pause_duration")
		pauseableRaw := r.PostFormValue("pauseable")
		keepVMRaw := r.PostFormValue("keep_vm")

		if name != "" {
			course.Spec.Name = name
		}

		if description != "" {
			course.Spec.Description = description
		}

		if scenarios != "" {
			scenarioSlice := make([]string, 0)
			err = json.Unmarshal([]byte(scenarios), &scenarioSlice)
			if err != nil {
				glog.Errorf("error while unmarshalling scenarios %v", err)
				util.ReturnHTTPMessage(w, r, 500, "internalerror", "error parsing")
				return fmt.Errorf("bad")
			}

			course.Spec.Scenarios = scenarioSlice
		}

		if categories != "" {
			categoriesSlice := make([]string, 0)
			err = json.Unmarshal([]byte(categories), &categoriesSlice)
			if err != nil {
				glog.Errorf("error while unmarshalling categories %v", err)
				util.ReturnHTTPMessage(w, r, 500, "internalerror", "error parsing")
				return fmt.Errorf("bad")
			}
			course.Spec.Categories = categoriesSlice
		}

		if virtualMachinesRaw != "" {
			virtualmachines := []map[string]string{} // must be declared this way so as to JSON marshal into [] instead of null
			err = json.Unmarshal([]byte(virtualMachinesRaw), &virtualmachines)
			if err != nil {
				glog.Errorf("error while unmarshaling VMs %v", err)
				util.ReturnHTTPMessage(w, r, 500, "internalerror", "error parsing")
				return fmt.Errorf("bad")
			}

			course.Spec.VirtualMachines = virtualmachines
		}

		if keepaliveDuration != "" {
			course.Spec.KeepAliveDuration = keepaliveDuration
		}

		if pauseDuration != "" {
			course.Spec.PauseDuration = pauseDuration
		}

		if pauseableRaw != "" {
			pauseable, err := strconv.ParseBool(pauseableRaw)
			if err != nil {
				glog.Errorf("error while parsing bool: %v", err)
				util.ReturnHTTPMessage(w, r, 500, "internalerror", "error parsing")
				return fmt.Errorf("bad")
			}

			course.Spec.Pauseable = pauseable
		}

		if keepVMRaw != "" {
			keepVM, err := strconv.ParseBool(keepVMRaw)
			if err != nil {
				glog.Errorf("error while parsing bool: %v", err)
				util.ReturnHTTPMessage(w, r, 500, "internalerror", "error parsing")
				return fmt.Errorf("bad")
			}

			course.Spec.KeepVM = keepVM
		}

		_, updateErr := c.hfClientSet.HobbyfarmV1().Courses(util.GetReleaseNamespace()).Update(c.ctx, course, metav1.UpdateOptions{})
		return updateErr
	})

	if retryErr != nil {
		util.ReturnHTTPMessage(w, r, 500, "internalerror", "error attempting to update")
		return
	}

	util.ReturnHTTPMessage(w, r, 200, "updated", "")
	glog.V(4).Infof("Updated course %s", id)
	return
}

func (c CourseServer) DeleteFunc(w http.ResponseWriter, r *http.Request) {
	_, err := c.auth.AuthGrant(rbacclient.RbacRequest().HobbyfarmPermission(resourcePlural, rbacclient.VerbDelete), w, r)
	if err != nil {
		util.ReturnHTTPMessage(w, r, 403, "forbidden", "no access to toDelete scenarios")
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

	seList, err := c.hfClientSet.HobbyfarmV1().ScheduledEvents(util.GetReleaseNamespace()).List(c.ctx, metav1.ListOptions{})
	if err != nil {
		glog.Errorf("error retrieving scheduledevent list: %v", err)
		util.ReturnHTTPMessage(w, r, 500, "internalerror", "error while deleting course")
		return
	}

	seInUse := filterScheduledEvents(id, seList)

	sessList, err := c.hfClientSet.HobbyfarmV1().Sessions(util.GetReleaseNamespace()).List(c.ctx, metav1.ListOptions{})
	if err != nil {
		glog.Errorf("error retrieving session list: %v", err)
		util.ReturnHTTPMessage(w, r, 500, "internalerror", "error while deleting course")
		return
	}

	sessInUse := filterSessions(id, sessList)

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

	if !toDelete {
		util.ReturnHTTPMessage(w, r, 403, "badrequest", msg)
		return
	}

	err = c.hfClientSet.HobbyfarmV1().Courses(util.GetReleaseNamespace()).Delete(c.ctx, id, metav1.DeleteOptions{})
	if err != nil {
		glog.Errorf("error deleting course: %v", err)
		util.ReturnHTTPMessage(w, r, 500, "internalerror", "error while deleting course")
		return
	}

	util.ReturnHTTPMessage(w, r, 204, "deleted", "deleted successfully")
	glog.V(4).Infof("deleted course: %v", id)
}

func (c CourseServer) ListCoursesForAccesscode(w http.ResponseWriter, r *http.Request) {
	user, err := c.auth.AuthN(w, r)
	if err != nil {
		util.ReturnHTTPMessage(w, r, 403, "forbidden", "no access to list courses")
		return
	}

	vars := mux.Vars(r)
	accessCode := vars["access_code"]

	if accessCode == "" {
		util.ReturnHTTPMessage(w, r, 400, "badrequest", "access_code is missing")
		return
	}

	contains := false
	for _, acc := range user.Spec.AccessCodes {
		if acc == accessCode {
			contains = true
			break
		}
	}

	if !contains {
		util.ReturnHTTPMessage(w, r, 403, "forbidden", "no access to list scenarios for this AccessCode")
		return
	}

	var courseIds []string
	tempCourseIds, err := c.acClient.GetCourseIds(accessCode)
	if err != nil {
		glog.Errorf("error retrieving course ids for access code: %s %v", accessCode, err)
	} else {
		courseIds = append(courseIds, tempCourseIds...)
	}

	courseIds = util.UniqueStringSlice(courseIds)

	var courses []PreparedCourse
	for _, courseId := range courseIds {
		course, err := c.GetCourseById(courseId)
		if err != nil {
			glog.Errorf("error retrieving course %v", err)
		} else {
			course.Spec.Scenarios = c.AppendDynamicScenariosByCategories(course.Spec.Scenarios, course.Spec.Categories)

			pCourse := PreparedCourse{course.Name, course.Spec}
			courses = append(courses, pCourse)
		}
	}

	encodedCourses, err := json.Marshal(courses)
	if err != nil {
		glog.Error(err)
	}
	util.ReturnHTTPContent(w, r, 200, "success", encodedCourses)
}

func (c CourseServer) previewDynamicScenarios(w http.ResponseWriter, r *http.Request) {
	_, err := c.auth.AuthGrant(rbacclient.RbacRequest().HobbyfarmPermission("scenarios", rbacclient.VerbList), w, r)
	if err != nil {
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

	scenarios = c.AppendDynamicScenariosByCategories(scenarios, categoriesSlice)

	encodedScenarios, err := json.Marshal(scenarios)
	if err != nil {
		glog.Error(err)
	}
	util.ReturnHTTPContent(w, r, 200, "success", encodedScenarios)
}

func (c CourseServer) AppendDynamicScenariosByCategories(scenariosList []string, categories []string) []string {
	categorySelector := metav1.ListOptions{}
	for _, categoryQuery := range categories {
		categorySelectors := []string{}
		categoryQueryParts := strings.Split(categoryQuery, "&")
		for _, categoryQueryPart := range categoryQueryParts {
			operator := "in"
			if strings.HasPrefix(categoryQueryPart, "!") {
				operator = "notin"
				categoryQueryPart = categoryQueryPart[1:]
			}
			categorySelectors = append(categorySelectors, fmt.Sprintf("category-%s %s (true)", categoryQueryPart, operator))
		}
		categorySelectorString := strings.Join(categorySelectors, ",")
		glog.Errorf("query scenarios by query: %s", categorySelectorString)
		categorySelector = metav1.ListOptions{
			LabelSelector: fmt.Sprintf("%s", categorySelectorString),
		}
		scenarios, err := c.hfClientSet.HobbyfarmV1().Scenarios(util.GetReleaseNamespace()).List(c.ctx, categorySelector)

		if err != nil {
			glog.Errorf("error while retrieving scenarios %v", err)
			continue
		}
		for _, scenario := range scenarios.Items {
			scenariosList = append(scenariosList, scenario.Name)
		}
	}

	scenariosList = util.UniqueStringSlice(scenariosList)
	return scenariosList
}

func (c CourseServer) GetCourseById(id string) (hfv1.Course, error) {
	if len(id) == 0 {
		return hfv1.Course{}, fmt.Errorf("course id passed in was blank")
	}
	obj, err := c.courseIndexer.ByIndex(idIndex, id)

	if err != nil {
		return hfv1.Course{}, fmt.Errorf("error while retrieving course by ID %s %v", id, err)
	}

	if len(obj) < 1 {
		return hfv1.Course{}, fmt.Errorf("error while retrieving course by ID %s", id)
	}

	course, ok := obj[0].(*hfv1.Course)

	if !ok {
		return hfv1.Course{}, fmt.Errorf("error while retrieving course by ID %s %v", id, ok)
	}

	return *course, nil
}

// Filter a ScheduledEventList to find SEs that are a) active and b) using the course specified
func filterScheduledEvents(course string, seList *hfv1.ScheduledEventList) *[]hfv1.ScheduledEvent {
	outList := make([]hfv1.ScheduledEvent, 0)
	for _, se := range seList.Items {
		if se.Status.Finished == true {
			continue
		}

		for _, c := range se.Spec.Courses {
			if c == course {
				outList = append(outList, se)
				break
			}
		}
	}

	return &outList
}

func filterSessions(course string, list *hfv1.SessionList) *[]hfv1.Session {
	outList := make([]hfv1.Session, 0)
	for _, sess := range list.Items {
		if sess.Spec.CourseId == course {
			outList = append(outList, sess)
		}
	}

	return &outList
}

func idIndexer(obj interface{}) ([]string, error) {
	course, ok := obj.(*hfv1.Course)
	if !ok {
		return []string{}, nil
	}
	return []string{course.Name}, nil
}
