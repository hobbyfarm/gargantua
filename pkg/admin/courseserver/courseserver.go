package courseserver

import (
	"crypto/sha256"
	"encoding/base32"
	"encoding/base64"
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

type AdminCourseServer struct {
	auth        *authclient.AuthClient
	hfClientSet *hfClientset.Clientset
}

func NewAdminCourseServer(authClient *authclient.AuthClient, hfClientset *hfClientset.Clientset) (*AdminCourseServer, error) {
	s := AdminCourseServer{}

	s.hfClientSet = hfClientset
	s.auth = authClient

	return &s, nil
}

func (a AdminCourseServer) getCourse(id string) (hfv1.Course, error) {

	empty := hfv1.Course{}

	if len(id) == 0 {
		return empty, fmt.Errorf("course id passed in was empty")
	}

	obj, err := a.hfClientSet.HobbyfarmV1().Courses().Get(id, metav1.GetOptions{})
	if err != nil {
		return empty, fmt.Errorf("error while retrieving course by id: %s with error: %v", id, err)
	}

	return *obj, nil

}

func (a AdminCourseServer) SetupRoutes(r *mux.Router) {
	r.HandleFunc("/a/course/new", a.CreateFunc).Methods("POST")
	r.HandleFunc("/a/course/list", a.ListFunc).Methods("GET")
	r.HandleFunc("/a/course/{id}", a.GetFunc).Methods("GET")
	r.HandleFunc("/a/course/{id}/printable", a.PrintFunc).Methods("GET")
	r.HandleFunc("/a/course/{id}", a.UpdateFunc).Methods("PUT")
	glog.V(2).Infof("set up routes for course server")
}

type PreparedCourse struct {
	ID string `json:"id"`
	hfv1.CourseSpec
}

func (a AdminCourseServer) GetFunc(w http.ResponseWriter, r *http.Request) {
	_, err := a.auth.AuthNAdmin(w, r)
	if err != nil {
		util.ReturnHTTPMessage(w, r, 403, "forbidden", "no access to get course")
		return
	}

	vars := mux.Vars(r)

	id := vars["id"]

	if len(id) == 0 {
		util.ReturnHTTPMessage(w, r, 500, "error", "no id passed in")
		return
	}

	course, err := a.getCourse(id)

	if err != nil {
		glog.Errorf("error while retrieving course %v", err)
		util.ReturnHTTPMessage(w, r, 500, "error", "no course found")
		return
	}

	preparedCourse := PreparedCourse{course.Name, course.Spec}

	encodedCourse, err := json.Marshal(preparedCourse)
	if err != nil {
		glog.Error(err)
	}
	util.ReturnHTTPContent(w, r, 200, "success", encodedCourse)

	glog.V(2).Infof("retrieved course %s", course.Name)
}

func (a AdminCourseServer) PrintFunc(w http.ResponseWriter, r *http.Request) {
	_, err := a.auth.AuthNAdmin(w, r)
	if err != nil {
		util.ReturnHTTPMessage(w, r, 403, "forbidden", "no access to get course")
		return
	}

	vars := mux.Vars(r)

	id := vars["id"]

	if len(id) == 0 {
		util.ReturnHTTPMessage(w, r, 500, "error", "no id passed in")
		return
	}

	course, err := a.getCourse(id)

	if err != nil {
		glog.Errorf("error while retrieving course %v", err)
		util.ReturnHTTPMessage(w, r, 500, "error", "no course found")
		return
	}

	var content string

	name, err := base64.StdEncoding.DecodeString(course.Spec.Name)
	if err != nil {
		glog.Errorf("Error decoding title of course: %s %v", course.Name, err)
	}
	description, err := base64.StdEncoding.DecodeString(course.Spec.Description)
	if err != nil {
		glog.Errorf("Error decoding description of course: %s %v", course.Name, err)
	}

	content = fmt.Sprintf("# %s\n%s\n\n", name, description)

	for _, s := range course.Spec.Scenarios {

		name, err := base64.StdEncoding.DecodeString(s)
		if err != nil {
			glog.Errorf("Error decoding name of course: %s scenario %d: %v", course.Name, name, err)
		}

		content = content + fmt.Sprintf("%s\n", string(name))
	}

	util.ReturnHTTPRaw(w, r, content)

	glog.V(2).Infof("retrieved course and rendered for printability %s", course.Name)
}

func (a AdminCourseServer) ListFunc(w http.ResponseWriter, r *http.Request) {
	_, err := a.auth.AuthNAdmin(w, r)
	if err != nil {
		util.ReturnHTTPMessage(w, r, 403, "forbidden", "no access to list courses")
		return
	}

	courses, err := a.hfClientSet.HobbyfarmV1().Courses().List(metav1.ListOptions{})

	if err != nil {
		glog.Errorf("error while retrieving courses %v", err)
		util.ReturnHTTPMessage(w, r, 500, "error", "no courses found")
		return
	}

	preparedCourses := []PreparedCourse{}
	for _, s := range courses.Items {
		pCourse := PreparedCourse{s.Name, s.Spec}
		preparedCourses = append(preparedCourses, pCourse)
	}

	encodedCourses, err := json.Marshal(preparedCourses)
	if err != nil {
		glog.Error(err)
	}
	util.ReturnHTTPContent(w, r, 200, "success", encodedCourses)

	glog.V(2).Infof("listed courses")
}

func (a AdminCourseServer) CreateFunc(w http.ResponseWriter, r *http.Request) {
	_, err := a.auth.AuthNAdmin(w, r)
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
	// we won't error if no keep alive duration is passed in or if it's blank because we'll default elsewhere

	scenarios := []string{}
	virtualmachines := []map[string]string{}

	rawScenarios := r.PostFormValue("scenarios")
	if rawScenarios != "" {
		err = json.Unmarshal([]byte(rawScenarios), &scenarios)
		if err != nil {
			glog.Errorf("error while unmarshaling scenarios %v", err)
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
	pause_duration := r.PostFormValue("pause_duration")

	course := &hfv1.Course{}

	hasher := sha256.New()
	hasher.Write([]byte(name))
	sha := base32.StdEncoding.WithPadding(-1).EncodeToString(hasher.Sum(nil))[:10]
	course.Name = "s-" + strings.ToLower(sha)
	course.Spec.Id = "s-" + strings.ToLower(sha) // LEGACY!!!!

	course.Spec.Name = name
	course.Spec.Description = description
	course.Spec.VirtualMachines = virtualmachines
	course.Spec.Scenarios = scenarios
	course.Spec.KeepAliveDuration = keepaliveDuration

	course.Spec.Pauseable = false
	if pauseable != "" {
		if strings.ToLower(pauseable) == "true" {
			course.Spec.Pauseable = true
		}
	}

	if pause_duration != "" {
		course.Spec.PauseDuration = pause_duration
	}

	course, err = a.hfClientSet.HobbyfarmV1().Courses().Create(course)
	if err != nil {
		glog.Errorf("error creating course %v", err)
		util.ReturnHTTPMessage(w, r, 500, "internalerror", "error creating course")
		return
	}

	util.ReturnHTTPMessage(w, r, 201, "created", course.Name)
	return
}

func (a AdminCourseServer) UpdateFunc(w http.ResponseWriter, r *http.Request) {
	_, err := a.auth.AuthNAdmin(w, r)
	if err != nil {
		util.ReturnHTTPMessage(w, r, 403, "forbidden", "no access to update courses")
		return
	}

	vars := mux.Vars(r)

	id := vars["id"]
	if id == "" {
		util.ReturnHTTPMessage(w, r, 400, "badrequest", "no ID passed in")
		return
	}

	retryErr := retry.RetryOnConflict(retry.DefaultRetry, func() error {
		course, err := a.hfClientSet.HobbyfarmV1().Courses().Get(id, metav1.GetOptions{})
		if err != nil {
			glog.Error(err)
			util.ReturnHTTPMessage(w, r, 400, "badrequest", "no ID found")
			return fmt.Errorf("bad")
		}

		name := r.PostFormValue("name")
		description := r.PostFormValue("description")
		rawScenarios := r.PostFormValue("scenarios")
		pauseable := r.PostFormValue("pauseable")
		pause_duration := r.PostFormValue("pause_duration")
		keepaliveDuration := r.PostFormValue("keepalive_duration")
		rawVirtualMachines := r.PostFormValue("virtualmachines")

		if name != "" {
			course.Spec.Name = name
		}
		if description != "" {
			course.Spec.Description = description
		}
		if keepaliveDuration != "" {
			course.Spec.KeepAliveDuration = keepaliveDuration
		}

		if pauseable != "" {
			if strings.ToLower(pauseable) == "true" {
				course.Spec.Pauseable = true
			} else {
				course.Spec.Pauseable = false
			}
		}

		if pause_duration != "" {
			course.Spec.PauseDuration = pause_duration
		}

		if rawScenarios != "" {
			scenarios := []string{}

			err = json.Unmarshal([]byte(rawScenarios), &scenarios)
			if err != nil {
				glog.Errorf("error while unmarshaling scenarios %v", err)
				return fmt.Errorf("bad")
			}
			course.Spec.Scenarios = scenarios
		}

		if rawVirtualMachines != "" {
			virtualmachines := []map[string]string{}
			err = json.Unmarshal([]byte(rawVirtualMachines), &virtualmachines)
			if err != nil {
				glog.Errorf("error while unmarshaling VMs %v", err)
				return fmt.Errorf("bad")
			}
			course.Spec.VirtualMachines = virtualmachines
		}

		_, updateErr := a.hfClientSet.HobbyfarmV1().Courses().Update(course)
		return updateErr
	})

	if retryErr != nil {
		util.ReturnHTTPMessage(w, r, 500, "error", "error attempting to update")
		return
	}

	util.ReturnHTTPMessage(w, r, 200, "updated", "")
	return
}
