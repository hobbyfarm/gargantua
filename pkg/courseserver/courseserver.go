package courseserver

import (
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
)

const (
	idIndex = "courseserver.hobbyfarm.io/id-index"
)

type CourseServer struct {
	auth          *authclient.AuthClient
	hfClientSet   hfClientset.Interface
	acClient      *accesscode.AccessCodeClient
	courseIndexer cache.Indexer
}

type PreparedCourse struct {
	Id string `json:"id"`
	hfv1.CourseSpec
}

func NewCourseServer(authClient *authclient.AuthClient, acClient *accesscode.AccessCodeClient, hfClientset hfClientset.Interface, hfInformerFactory hfInformers.SharedInformerFactory) (*CourseServer, error) {
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

	return &course, nil
}

func (c CourseServer) SetupRoutes(r *mux.Router) {
	r.HandleFunc("/course/list", c.ListCoursesForAccesscode).Methods("GET")
	r.HandleFunc("/course/{course_id}", c.GetCourse).Methods("GET")
	r.HandleFunc("/a/course/list", c.ListFunc).Methods("GET")
	r.HandleFunc("/a/course/new", c.CreateFunc).Methods("POST")
	r.HandleFunc("/a/course/{id}", c.GetCourse).Methods("GET")
	r.HandleFunc("/a/course/{id}", c.UpdateFunc).Methods("PUT")
	r.HandleFunc("/a/course/{id}", c.DeleteFunc).Methods("DELETE")
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
	_, err := c.auth.AuthNAdmin(w, r)
	if err != nil {
		util.ReturnHTTPMessage(w, r, 403, "forbidden", "no access to list courses")
		return
	}

	tempCourses, err := c.hfClientSet.HobbyfarmV1().Courses().List(metav1.ListOptions{})
	if err != nil {
		glog.Errorf("error listing courses: %v", err)
		util.ReturnHTTPMessage(w, r, 500, "internalerror", "error listing courses")
		return
	}

	var courses []PreparedCourse
	for _, c := range tempCourses.Items {
		courses = append(courses, PreparedCourse{ c.Name, c.Spec})
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
	_, err := c.auth.AuthN(w, r)
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
	_, err := c.auth.AuthNAdmin(w, r)
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

	course := &hfv1.Course{}

	generatedName := util.GenerateResourceName("c", name, 10)

	course.Name = generatedName
	course.Spec.Id = generatedName

	course.Spec.Name = name
	course.Spec.Description = description
	course.Spec.VirtualMachines = virtualmachines
	course.Spec.Scenarios = scenarioSlice
	if keepaliveDuration != "" {
		course.Spec.KeepAliveDuration = keepaliveDuration
	}
	course.Spec.Pauseable = pauseable
	if pauseDuration != "" {
		course.Spec.PauseDuration = pauseDuration
	}

	course, err = c.hfClientSet.HobbyfarmV1().Courses().Create(course)
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
	_, err := c.auth.AuthNAdmin(w, r)
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
		course, err := c.hfClientSet.HobbyfarmV1().Courses().Get(id, metav1.GetOptions{})
		if err != nil {
			glog.Error(err)
			util.ReturnHTTPMessage(w, r, 400, "badrequest", "no id found")
			return fmt.Errorf("bad")
		}
		// name, description, scenarios, virtualmachines, keepaliveduration, pauseduration, pauseable

		name := r.PostFormValue("name")
		description := r.PostFormValue("description")
		scenarios := r.PostFormValue("scenarios")
		virtualMachinesRaw := r.PostFormValue("virtualmachines")
		keepaliveDuration := r.PostFormValue("keepalive_duration")
		pauseDuration := r.PostFormValue("pause_duration")
		pauseableRaw := r.PostFormValue("pauseable")

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

		_, updateErr := c.hfClientSet.HobbyfarmV1().Courses().Update(course)
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
	_, err := c.auth.AuthNAdmin(w, r)
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

	seList, err := c.hfClientSet.HobbyfarmV1().ScheduledEvents().List(metav1.ListOptions{})
	if err != nil {
		glog.Errorf("error retrieving scheduledevent list: %v", err)
		util.ReturnHTTPMessage(w, r, 500, "internalerror", "error while deleting course")
		return
	}

	seInUse := filterScheduledEvents(id, seList)

	sessList, err := c.hfClientSet.HobbyfarmV1().Sessions().List(metav1.ListOptions{})
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

	err = c.hfClientSet.HobbyfarmV1().Courses().Delete(id, &metav1.DeleteOptions{})
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

	var courseIds []string
	for _, ac := range user.Spec.AccessCodes {
		tempCourseIds, err := c.acClient.GetCourseIds(ac)
		if err != nil {
			glog.Errorf("error retrieving course ids for access code: %s %v", ac, err)
		} else {
			courseIds = append(courseIds, tempCourseIds...)
		}
	}

	courseIds = util.UniqueStringSlice(courseIds)

	var courses []PreparedCourse
	for _, courseId := range courseIds {
		course, err := c.GetCourseById(courseId)
		if err != nil {
			glog.Errorf("error retrieving course %v", err)
		} else {
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
	return []string{course.Spec.Id}, nil
}
