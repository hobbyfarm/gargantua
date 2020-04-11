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
	"k8s.io/client-go/tools/cache"
	"net/http"
)

const (
	idIndex = "courseserver.hobbyfarm.io/id-index"
)

type CourseServer struct {
	auth          *authclient.AuthClient
	hfClientSet   *hfClientset.Clientset
	acClient      *accesscode.AccessCodeClient
	courseIndexer cache.Indexer
}

type PreparedCourse struct {
	Id              string              `json:"id"`
	hfv1.CourseSpec
}

func NewCourseServer(authClient *authclient.AuthClient, acClient *accesscode.AccessCodeClient, hfClientset *hfClientset.Clientset, hfInformerFactory hfInformers.SharedInformerFactory) (*CourseServer, error) {
	course := CourseServer{}

	course.hfClientSet = hfClientset
	course.acClient = acClient
	course.auth = authClient
	inf := hfInformerFactory.Hobbyfarm().V1().Courses().Informer()
	indexers := map[string]cache.IndexFunc{idIndex: idIndexer}

	inf.AddIndexers(indexers)
	course.courseIndexer = inf.GetIndexer()

	return &course, nil
}

func (c CourseServer) SetupRoutes(r *mux.Router) {
	r.HandleFunc("/course/list", c.ListCourse).Methods("GET")
	r.HandleFunc("/course/{course_id}", c.GetCourse).Methods("GET")
}

func (c CourseServer) getPreparedCourseById(id string) (PreparedCourse, error) {
	course, err := c.GetCourseById(id)

	if err != nil {
		return PreparedCourse{}, fmt.Errorf("error while retrieving course %v", err)
	}

	preparedCourse := PreparedCourse{course.Name, course.Spec}

	return preparedCourse, nil
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

func (c CourseServer) ListCourse(w http.ResponseWriter, r *http.Request) {
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

func idIndexer(obj interface{}) ([]string, error) {
	course, ok := obj.(*hfv1.Course)
	if !ok {
		return []string{}, nil
	}
	return []string{course.Spec.Id}, nil
}
