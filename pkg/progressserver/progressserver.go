package progressserver

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/golang/glog"
	"github.com/gorilla/mux"
	hfv1 "github.com/hobbyfarm/gargantua/pkg/apis/hobbyfarm.io/v1"
	"github.com/hobbyfarm/gargantua/pkg/authclient"
	hfClientset "github.com/hobbyfarm/gargantua/pkg/client/clientset/versioned"
	"github.com/hobbyfarm/gargantua/pkg/rbacclient"
	"github.com/hobbyfarm/gargantua/pkg/util"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/util/retry"
	"net/http"
	"strconv"
	"time"
)

const (
	idIndex             = "progressserver.hobbyfarm.io/id-index"
	ScheduledEventLabel = "hobbyfarm.io/scheduledevent"
	SessionLabel        = "hobbyfarm.io/session"
	UserLabel           = "hobbyfarm.io/user"
	resourcePlural		= "progresses"
)

type ProgressServer struct {
	auth        *authclient.AuthClient
	hfClientSet hfClientset.Interface
	ctx         context.Context
}

type AdminPreparedProgress struct {
	Session string `json:"session"`
	hfv1.ProgressSpec
}

type ScheduledEventProgressCount struct {
	CountMap map[string]int `json:"count_map"`
}

func NewProgressServer(authClient *authclient.AuthClient, hfClientset hfClientset.Interface, ctx context.Context) (*ProgressServer, error) {
	progress := ProgressServer{}

	progress.hfClientSet = hfClientset
	progress.auth = authClient
	progress.ctx = ctx
	return &progress, nil
}

func (s ProgressServer) SetupRoutes(r *mux.Router) {
	r.HandleFunc("/a/progress/scheduledevent/{id}", s.ListByScheduledEventFunc).Methods("GET")
	r.HandleFunc("/a/progress/user/{id}", s.ListByUserFunc).Methods("GET")
	r.HandleFunc("/a/progress/count", s.CountByScheduledEvent).Methods("GET")
	r.HandleFunc("/progress/update/{id}", s.Update).Methods("POST")
	r.HandleFunc("/progress/list", s.ListForUserFunc).Methods("GET")
	glog.V(2).Infof("set up routes for ProgressServer")
}

/*
	List Progress by Scheduled Event
		Vars:
		- id : The scheduled event id
*/
func (s ProgressServer) ListByScheduledEventFunc(w http.ResponseWriter, r *http.Request) {
	_, err := s.auth.AuthGrant(rbacclient.RbacRequest().HobbyfarmPermission(resourcePlural, rbacclient.VerbList), w, r)
	if err != nil {
		util.ReturnHTTPMessage(w, r, 403, "forbidden", "no access to list progress")
		return
	}

	vars := mux.Vars(r)

	id := vars["id"]

	if len(id) == 0 {
		util.ReturnHTTPMessage(w, r, 500, "error", "no id passed in")
		return
	}

	includeFinished := false
	includeFinishedParam := r.URL.Query().Get("includeFinished")
	if includeFinishedParam != "" && includeFinishedParam != "false" {
		includeFinished = true
	}

	s.ListByLabel(w, r, ScheduledEventLabel, id, includeFinished)

	glog.V(2).Infof("listed progress for scheduledevent %s", id)
}

/*
	List Progress for the authenticated user
*/
func (s ProgressServer) ListForUserFunc(w http.ResponseWriter, r *http.Request) {
	user, err := s.auth.AuthN(w, r)
	if err != nil {
		util.ReturnHTTPMessage(w, r, 403, "forbidden", "no access to list progress")
		return
	}

	s.ListByLabel(w, r, UserLabel, user.Spec.Id, true)
}

/*
	List Progress by User
		Vars:
		- id : The user id
*/
func (s ProgressServer) ListByUserFunc(w http.ResponseWriter, r *http.Request) {
	_, err := s.auth.AuthGrant(rbacclient.RbacRequest().HobbyfarmPermission(resourcePlural, rbacclient.VerbList), w, r)
	if err != nil {
		util.ReturnHTTPMessage(w, r, 403, "forbidden", "no access to list progress")
		return
	}

	vars := mux.Vars(r)

	id := vars["id"]

	if len(id) == 0 {
		util.ReturnHTTPMessage(w, r, 500, "error", "no id passed in")
		return
	}

	s.ListByLabel(w, r, UserLabel, id, true)

	glog.V(2).Infof("listed progress for user %s", id)
}

func (s ProgressServer) CountByScheduledEvent(w http.ResponseWriter, r *http.Request) {
	_, err := s.auth.AuthGrant(rbacclient.RbacRequest().HobbyfarmPermission(resourcePlural, rbacclient.VerbList), w, r)
	if err != nil {
		util.ReturnHTTPMessage(w, r, 403, "forbidden", "no access to list progress")
		return
	}

	progress, err := s.hfClientSet.HobbyfarmV1().Progresses(util.GetReleaseNamespace()).List(s.ctx, metav1.ListOptions{
		LabelSelector: fmt.Sprintf("%s=%s", "finished", "false")})

	if err != nil {
		glog.Errorf("error while retrieving progress %v", err)
		util.ReturnHTTPMessage(w, r, 500, "error", "no progress found")
		return
	}
	countMap := map[string]int{}
	for _, p := range progress.Items {
		se := p.Labels[ScheduledEventLabel]
		if _, ok := countMap[se]; ok {
			countMap[se] = countMap[se] + 1
		} else {
			countMap[se] = 1
		}
	}

	encodedMap, err := json.Marshal(countMap)
	if err != nil {
		glog.Error(err)
	}
	util.ReturnHTTPContent(w, r, 200, "success", encodedMap)
}

func (s ProgressServer) ListByLabel(w http.ResponseWriter, r *http.Request, label string, value string, includeFinished bool) {
	includeFinishedFilter := ",finished=false" // Default is to only include active (finished=false) progress
	if includeFinished {
		includeFinishedFilter = ""
	}
	progress, err := s.hfClientSet.HobbyfarmV1().Progresses(util.GetReleaseNamespace()).List(s.ctx, metav1.ListOptions{
		LabelSelector: fmt.Sprintf("%s=%s%s", label, value, includeFinishedFilter)})

	if err != nil {
		glog.Errorf("error while retrieving progress %v", err)
		util.ReturnHTTPMessage(w, r, 500, "error", "no progress found")
		return
	}

	preparedProgress := []AdminPreparedProgress{}
	for _, p := range progress.Items {
		pProgress := AdminPreparedProgress{p.Labels[SessionLabel], p.Spec}
		preparedProgress = append(preparedProgress, pProgress)
	}

	encodedProgress, err := json.Marshal(preparedProgress)
	if err != nil {
		glog.Error(err)
	}
	util.ReturnHTTPContent(w, r, 200, "success", encodedProgress)
}

/*
	Update Progress
		Vars:
		- id : Session linked to the progress resource
*/
func (s ProgressServer) Update(w http.ResponseWriter, r *http.Request) {
	now := time.Now()

	user, err := s.auth.AuthN(w, r)
	if err != nil {
		util.ReturnHTTPMessage(w, r, 403, "forbidden", "no access to update progress")
		return
	}

	vars := mux.Vars(r)

	id := vars["id"]

	if len(id) == 0 {
		util.ReturnHTTPMessage(w, r, 500, "error", "no id passed in")
		return
	}

	stepRaw := r.PostFormValue("step")
	if stepRaw == "" {
		util.ReturnHTTPMessage(w, r, 400, "badrequest", "no step was passed in")
		return
	}

	step, err := strconv.Atoi(stepRaw)
	if err != nil {
		glog.Errorf("error while converting step %v", err)
		util.ReturnHTTPMessage(w, r, 500, "error", "provided step was invalid")
		return
	}

	progress, err := s.hfClientSet.HobbyfarmV1().Progresses(util.GetReleaseNamespace()).List(s.ctx, metav1.ListOptions{
		LabelSelector: fmt.Sprintf("%s=%s,%s=%s,finished=false", SessionLabel, id, UserLabel, user.Spec.Id)})

	if err != nil {
		glog.Errorf("error while retrieving progress %v", err)
		util.ReturnHTTPMessage(w, r, 500, "error", "no active progress for this session found")
		return
	}

	if len(progress.Items) < 1 {
		util.ReturnHTTPMessage(w, r, 404, "error", "no active progress for this session found")
		return
	}

	for _, p := range progress.Items {
		retryErr := retry.RetryOnConflict(retry.DefaultRetry, func() error {
			if step > p.Spec.MaxStep {
				p.Spec.MaxStep = step
			}
			p.Spec.CurrentStep = step
			p.Spec.LastUpdate = now.Format(time.UnixDate)

			steps := p.Spec.Steps
			newStep := hfv1.ProgressStep{Step: step, Timestamp: now.Format(time.UnixDate)}
			steps = append(steps, newStep)
			p.Spec.Steps = steps

			_, updateErr := s.hfClientSet.HobbyfarmV1().Progresses(util.GetReleaseNamespace()).Update(s.ctx, &p, metav1.UpdateOptions{})
			glog.V(4).Infof("updated result for environment")

			return updateErr
		})

		if retryErr != nil {
			glog.Errorf("error updating progress %s: %v", p.Spec.Id, err)
			util.ReturnHTTPMessage(w, r, 500, "error", "progress could not be updated")
			return
		}
	}

	util.ReturnHTTPMessage(w, r, 200, "success", "Progress was updated")
}
