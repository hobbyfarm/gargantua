package progressservice

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	hfv1 "github.com/hobbyfarm/gargantua/v3/pkg/apis/hobbyfarm.io/v1"
	hferrors "github.com/hobbyfarm/gargantua/v3/pkg/errors"
	hflabels "github.com/hobbyfarm/gargantua/v3/pkg/labels"
	"github.com/hobbyfarm/gargantua/v3/pkg/rbac"
	"github.com/hobbyfarm/gargantua/v3/pkg/util"

	"github.com/golang/glog"
	"github.com/gorilla/mux"
	generalpb "github.com/hobbyfarm/gargantua/v3/protos/general"
	progresspb "github.com/hobbyfarm/gargantua/v3/protos/progress"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/util/retry"
)

const (
	idIndex        = "progressserver.hobbyfarm.io/id-index"
	resourcePlural = rbac.ResourcePluralProgress
)

type AdminPreparedProgress struct {
	ID          string                     `json:"id"`
	Session     string                     `json:"session"`
	CurrentStep uint32                     `json:"current_step"`
	MaxStep     uint32                     `json:"max_step"`
	TotalStep   uint32                     `json:"total_step"`
	Course      string                     `json:"course"`
	Scenario    string                     `json:"scenario"`
	UserId      string                     `json:"user"`
	Started     string                     `json:"started"`
	LastUpdate  string                     `json:"last_update"`
	Finished    string                     `json:"finished"`
	Steps       []*progresspb.ProgressStep `json:"steps"`
}

type AdminPreparedProgressWithScheduledEvent struct {
	AdminPreparedProgress
	ScheduledEvent string `json:"scheduled_event"`
}

type ScheduledEventProgressCount struct {
	CountMap map[string]int `json:"count_map"`
}

/*
List Progress by Scheduled Event

	Vars:
	- id : The scheduled event id
*/
func (s ProgressServer) ListByScheduledEventFunc(w http.ResponseWriter, r *http.Request) {
	user, err := rbac.AuthenticateRequest(r, s.authnClient)
	if err != nil {
		util.ReturnHTTPMessage(w, r, 401, "unauthorized", "authentication failed")
		return
	}

	impersonatedUserId := user.GetId()
	authrResponse, err := rbac.AuthorizeSimple(r, s.authrClient, impersonatedUserId, rbac.HobbyfarmPermission(resourcePlural, rbac.VerbList))
	if err != nil || !authrResponse.Success {
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

	s.ListByLabel(w, r, hflabels.ScheduledEventLabel, id, includeFinished)

	glog.V(2).Infof("listed progress for scheduledevent %s", id)
}

func (s ProgressServer) ListByRangeFunc(w http.ResponseWriter, r *http.Request) {
	user, err := rbac.AuthenticateRequest(r, s.authnClient)
	if err != nil {
		util.ReturnHTTPMessage(w, r, 401, "unauthorized", "authentication failed")
		return
	}

	impersonatedUserId := user.GetId()
	authrResponse, err := rbac.AuthorizeSimple(r, s.authrClient, impersonatedUserId, rbac.HobbyfarmPermission(resourcePlural, rbac.VerbList))
	if err != nil || !authrResponse.Success {
		util.ReturnHTTPMessage(w, r, 403, "forbidden", "no access to list progress")
		return
	}

	fromString := r.URL.Query().Get("from")
	if fromString == "" {
		util.ReturnHTTPMessage(w, r, 500, "error", "no start of range passed in")
		return
	}

	start, err := time.Parse(time.UnixDate, fromString)

	if err != nil {
		util.ReturnHTTPMessage(w, r, 500, "error", "error parsing start time")
		return
	}

	toString := r.URL.Query().Get("to")
	if toString == "" {
		util.ReturnHTTPMessage(w, r, 500, "error", "no end of range passed in")
		return
	}

	end, err := time.Parse(time.UnixDate, toString)

	if err != nil {
		util.ReturnHTTPMessage(w, r, 500, "error", "error parsing end time")
		return
	}

	s.ListByRange(w, r, start, end, true)

	glog.V(2).Info("listed progress for time range")
}

/*
List Progress for the authenticated user
*/
func (s ProgressServer) ListForUserFunc(w http.ResponseWriter, r *http.Request) {
	user, err := rbac.AuthenticateRequest(r, s.authnClient)
	if err != nil {
		util.ReturnHTTPMessage(w, r, 403, "forbidden", "no access to list progress")
		return
	}

	s.ListByLabel(w, r, hflabels.UserLabel, user.GetId(), true)
}

/*
List Progress by User

	Vars:
	- id : The user id
*/
func (s ProgressServer) ListByUserFunc(w http.ResponseWriter, r *http.Request) {
	user, err := rbac.AuthenticateRequest(r, s.authnClient)
	if err != nil {
		util.ReturnHTTPMessage(w, r, 401, "unauthorized", "authentication failed")
		return
	}

	impersonatedUserId := user.GetId()
	authrResponse, err := rbac.AuthorizeSimple(r, s.authrClient, impersonatedUserId, rbac.HobbyfarmPermission(resourcePlural, rbac.VerbList))
	if err != nil || !authrResponse.Success {
		util.ReturnHTTPMessage(w, r, 403, "forbidden", "no access to list progress")
		return
	}

	vars := mux.Vars(r)

	id := vars["id"]

	if len(id) == 0 {
		util.ReturnHTTPMessage(w, r, 500, "error", "no id passed in")
		return
	}

	s.ListByLabel(w, r, hflabels.UserLabel, id, true)

	glog.V(2).Infof("listed progress for user %s", id)
}

func (s ProgressServer) CountByScheduledEvent(w http.ResponseWriter, r *http.Request) {
	user, err := rbac.AuthenticateRequest(r, s.authnClient)
	if err != nil {
		util.ReturnHTTPMessage(w, r, 401, "unauthorized", "authentication failed")
		return
	}

	impersonatedUserId := user.GetId()
	authrResponse, err := rbac.AuthorizeSimple(r, s.authrClient, impersonatedUserId, rbac.HobbyfarmPermission(resourcePlural, rbac.VerbList))
	if err != nil || !authrResponse.Success {
		util.ReturnHTTPMessage(w, r, 403, "forbidden", "no access to list progress")
		return
	}

	progressList, err := s.internalProgressServer.ListProgress(r.Context(), &generalpb.ListOptions{
		LabelSelector: fmt.Sprintf("%s=%s", "finished", "false"),
	})
	if err != nil {
		glog.Errorf("error while retrieving progress: %s", hferrors.GetErrorMessage(err))
		util.ReturnHTTPMessage(w, r, 500, "error", "no progress found")
		return
	}
	countMap := map[string]int{}
	for _, p := range progressList.GetProgresses() {
		se := p.GetLabels()[hflabels.ScheduledEventLabel]
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

func (s ProgressServer) ListByRange(w http.ResponseWriter, r *http.Request, start time.Time, end time.Time, includeFinished bool) {
	includeFinishedFilter := "finished=false" // Default is to only include active (finished=false) progress
	if includeFinished {
		includeFinishedFilter = ""
	}

	progressList, err := s.internalProgressServer.ListProgress(r.Context(), &generalpb.ListOptions{
		LabelSelector: includeFinishedFilter,
	})

	if err != nil {
		glog.Errorf("error while retrieving progress: %s", hferrors.GetErrorMessage(err))
		util.ReturnHTTPMessage(w, r, 500, "error", "no progress found")
		return
	}

	preparedProgress := []AdminPreparedProgressWithScheduledEvent{}
	for _, p := range progressList.GetProgresses() {
		creationTimeStamp := p.GetCreationTimestamp().AsTime()
		//CreationTimestamp of progress is out of range
		if creationTimeStamp.Before(start) || end.Before(creationTimeStamp) {
			continue
		}
		pProgressWithEventId := AdminPreparedProgressWithScheduledEvent{
			AdminPreparedProgress: AdminPreparedProgress{
				ID:          p.GetId(),
				Session:     p.GetLabels()[hflabels.SessionLabel],
				CurrentStep: p.GetCurrentStep(),
				MaxStep:     p.GetMaxStep(),
				TotalStep:   p.GetTotalStep(),
				Course:      p.GetCourse(),
				Scenario:    p.GetScenario(),
				UserId:      p.GetUser(),
				Started:     p.GetStarted(),
				LastUpdate:  p.GetLastUpdate(),
				Finished:    p.GetFinished(),
				Steps:       p.GetSteps(),
			},
			ScheduledEvent: p.GetLabels()[hflabels.ScheduledEventLabel],
		}
		preparedProgress = append(preparedProgress, pProgressWithEventId)
	}

	encodedProgress, err := json.Marshal(preparedProgress)
	if err != nil {
		glog.Error(err)
	}
	util.ReturnHTTPContent(w, r, 200, "success", encodedProgress)
}

func (s ProgressServer) ListByLabel(w http.ResponseWriter, r *http.Request, label string, value string, includeFinished bool) {
	includeFinishedFilter := ",finished=false" // Default is to only include active (finished=false) progress
	if includeFinished {
		includeFinishedFilter = ""
	}
	progressList, err := s.internalProgressServer.ListProgress(r.Context(), &generalpb.ListOptions{
		LabelSelector: fmt.Sprintf("%s=%s%s", label, value, includeFinishedFilter),
	})

	if err != nil {
		glog.Errorf("error while retrieving progress %s", hferrors.GetErrorMessage(err))
		util.ReturnHTTPMessage(w, r, 500, "error", "no progress found")
		return
	}

	preparedProgress := []AdminPreparedProgress{}
	for _, p := range progressList.GetProgresses() {
		pProgress := AdminPreparedProgress{
			ID:          p.GetId(),
			Session:     p.GetLabels()[hflabels.SessionLabel],
			CurrentStep: p.GetCurrentStep(),
			MaxStep:     p.GetMaxStep(),
			TotalStep:   p.GetTotalStep(),
			Course:      p.GetCourse(),
			Scenario:    p.GetScenario(),
			UserId:      p.GetUser(),
			Started:     p.GetStarted(),
			LastUpdate:  p.GetLastUpdate(),
			Finished:    p.GetFinished(),
			Steps:       p.GetSteps(),
		}
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

	user, err := rbac.AuthenticateRequest(r, s.authnClient)
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

	// @TODO: Currently we're using the progress k8s client directly.
	// The grpc service would internally retrieve each progress object again before updating which is less performant.
	// In the future, we should use the internal gRPC service to abstract the underlying database operations.
	// This approach ensures that database interactions are not exposed directly, allowing easier interchangeability and improved modularity.
	progress, err := s.internalProgressServer.progressClient.List(r.Context(), metav1.ListOptions{
		LabelSelector: fmt.Sprintf("%s=%s,%s=%s,finished=false", hflabels.SessionLabel, id, hflabels.UserLabel, user.GetId())})

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

			_, updateErr := s.internalProgressServer.progressClient.Update(r.Context(), &p, metav1.UpdateOptions{})
			glog.V(4).Infof("updated result for progress")

			return updateErr
		})

		if retryErr != nil {
			glog.Errorf("error updating progress %s: %v", p.Name, err)
			util.ReturnHTTPMessage(w, r, 500, "error", "progress could not be updated")
			return
		}
	}

	util.ReturnHTTPMessage(w, r, 200, "success", "Progress was updated")
}
