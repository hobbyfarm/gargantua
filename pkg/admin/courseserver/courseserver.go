package courseserver

import (
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
	"strconv"
)

type AdminCourseServer struct {
	auth *authclient.AuthClient
	hfClientSet *hfClientset.Clientset
}

type PreparedCourse struct {
	Id string `json:"id"`
	hfv1.CourseSpec
}

func NewAdminCourseServer(authClient *authclient.AuthClient, hfClientset *hfClientset.Clientset) (*AdminCourseServer, error) {
	s := AdminCourseServer{}

	s.hfClientSet = hfClientset
	s.auth = authClient

	return &s, nil
}

func (a AdminCourseServer) SetupRoutes(r *mux.Router) {
	r.HandleFunc("/a/course/{id}", a.DeleteFunc).Methods("DELETE")
}



func (a AdminCourseServer) DeleteFunc(w http.ResponseWriter, r *http.Request) {
	_, err := a.auth.AuthNAdmin(w, r)
	if err != nil {
		util.ReturnHTTPMessage(w, r, 403, "forbidden", "no access to delete scenarios")
		return
	}

	vars := mux.Vars(r)

	id := vars["id"]
	if id == "" {
		util.ReturnHTTPMessage(w, r, 400, "badrequest", "no id passed in")
		return
	}

	// when can we safely delete a course?
	// 1. when there are no active scheduled events using the course
	// 2. when there are no sessions using the course

	seList, err := a.hfClientSet.HobbyfarmV1().ScheduledEvents().List(metav1.ListOptions{})
	if err != nil {
		glog.Errorf("error retrieving scheduledevent list: %v", err)
		util.ReturnHTTPMessage(w, r, 500, "internalerror", "error while deleting course")
		return
	}

	seInUse := filterScheduledEvents(id, seList)

	sessList, err := a.hfClientSet.HobbyfarmV1().Sessions().List(metav1.ListOptions{})
	if err != nil {
		glog.Errorf("error retrieving session list: %v", err)
		util.ReturnHTTPMessage(w, r, 500, "internalerror", "error while deleting course")
		return
	}

	sessInUse := filterSessions(id, sessList)

	var msg string = ""
	delete := true

	if len(*seInUse) > 0 {
		// cannot delete, in use. alert the user
		msg += "In use by scheduled events:"
		for _, se := range *seInUse {
			msg += " " + se.Name
		}
		delete = false
	}

	if len(*sessInUse) > 0 {
		msg += "In use by sessions:"
		for _, sess := range *sessInUse {
			msg += " " + sess.Name
		}
		delete = false
	}

	if !delete {
		util.ReturnHTTPMessage(w, r, 403, "badrequest", msg)
		return
	}

	err = a.hfClientSet.HobbyfarmV1().Courses().Delete(id, &metav1.DeleteOptions{})
	if err != nil {
		glog.Errorf("error deleting course: %v", err)
		util.ReturnHTTPMessage(w, r, 500, "internalerror", "error while deleting course")
		return
	}

	util.ReturnHTTPMessage(w, r, 204, "deleted", "deleted successfully")
	glog.V(4).Infof("deleted course: %v", id)
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