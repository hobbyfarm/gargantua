package sessionservice

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"

	hferrors "github.com/hobbyfarm/gargantua/v3/pkg/errors"
	hflabels "github.com/hobbyfarm/gargantua/v3/pkg/labels"
	"github.com/hobbyfarm/gargantua/v3/pkg/rbac"
	"github.com/hobbyfarm/gargantua/v3/pkg/util"
	coursepb "github.com/hobbyfarm/gargantua/v3/protos/course"
	generalpb "github.com/hobbyfarm/gargantua/v3/protos/general"
	progresspb "github.com/hobbyfarm/gargantua/v3/protos/progress"
	scenariopb "github.com/hobbyfarm/gargantua/v3/protos/scenario"
	sessionpb "github.com/hobbyfarm/gargantua/v3/protos/session"
	vmclaimpb "github.com/hobbyfarm/gargantua/v3/protos/vmclaim"
	"google.golang.org/protobuf/types/known/wrapperspb"

	"github.com/golang/glog"
	"github.com/gorilla/mux"
)

const (
	ssIndex            = "sss.hobbyfarm.io/session-id-index"
	newSSTimeout       = "5m"
	keepaliveSSTimeout = "5m"
	pauseSSTimeout     = "2h"
	resourcePlural     = rbac.ResourcePluralSession
)

type preparedSession struct {
	Id           string   `json:"id"`
	ScenarioId   string   `json:"scenario"`
	CourseId     string   `json:"course"`
	KeepCourseVM bool     `json:"keep_course_vm"`
	UserId       string   `json:"user"`
	VmClaimSet   []string `json:"vm_claim"`
	AccessCode   string   `json:"access_code"`
}

func (sss SessionServer) NewSessionFunc(w http.ResponseWriter, r *http.Request) {
	user, err := rbac.AuthenticateRequest(r, sss.authnClient)
	if err != nil {
		util.ReturnHTTPMessage(w, r, 403, "forbidden", "no access to create sessions")
		return
	}

	courseid := r.PostFormValue("course")
	scenarioid := r.PostFormValue("scenario")

	if courseid == "" && scenarioid == "" {
		util.ReturnHTTPMessage(w, r, 400, "bad request", "no course/scenario id passed in")
		return
	}

	accessCodeId := r.PostFormValue("access_code")

	restrictedBind := false
	restrictedBindVal := ""

	if accessCodeId == "" {
		util.ReturnHTTPMessage(w, r, 400, "bad request", "An accesscode has to be given in order so start a session")
		return
	}

	// we should validate the user can use this access code
	// let's figure out the restricted bind value
	accessCodeObj, err := sss.acClient.GetAccessCodeWithOTACs(r.Context(), &generalpb.ResourceId{Id: accessCodeId})
	if err != nil {
		util.ReturnHTTPMessage(w, r, 500, "error", "could not retrieve access code")
		return
	}
	if accessCodeObj.GetRestrictedBind() {
		restrictedBind = accessCodeObj.GetRestrictedBind()
		restrictedBindVal = accessCodeObj.GetRestrictedBindValue()
	}
	scheduledEventId := accessCodeObj.Labels[hflabels.ScheduledEventLabel]

	course, scenario, ok := sss.getCourseAndScenarioFromCache(w, r, courseid, scenarioid)
	if !ok {
		// we encountered an error and already returned an HTTPMessage in getCourseAndScenarioFromCache()
		return
	}

	// now we should check for existing sessions for the user
	sessionList, err := sss.internalSessionServer.ListSession(r.Context(), &generalpb.ListOptions{
		LabelSelector: fmt.Sprintf("%s=%s", hflabels.UserLabel, user.GetId()),
	})
	if err != nil {
		glog.Error(hferrors.GetErrorMessage(err))
	}
	now := time.Now()

	// should we check the sessions list for the restricted bind value and match if one is passed in? probably...
	for _, sess := range sessionList.GetSessions() {
		expires, err := time.Parse(time.UnixDate, sess.GetStatus().GetExpirationTime())
		if err != nil {
			continue
		}
		if sess.GetUser() == user.GetId() &&
			(sess.GetCourse() == courseid || sess.GetScenario() == scenarioid) &&
			!sess.GetStatus().GetFinished() &&
			sess.GetStatus().GetActive() && expires.After(now) {
			// we should just return this session...

			// if this is a course, return the same scenario id that was given to us
			// i.e., reuse the course id and give them the scenario they asked for
			if sess.GetCourse() != "" {
				_, err = sss.internalSessionServer.UpdateSession(r.Context(), &sessionpb.UpdateSessionRequest{
					Id:       sess.GetId(),
					Scenario: scenarioid,
				})
				if err != nil {
					glog.Errorf("error updating session %s", hferrors.GetErrorMessage(err))
					errMsg := fmt.Sprintf("error retrieving new session: could not update session to new scenario %s", scenarioid)
					util.ReturnHTTPMessage(w, r, 500, "error", errMsg)
					return
				}

				err = sss.FinishProgress(r.Context(), sess.GetId(), user.GetId())
				if err != nil {
					glog.Errorf("error finishing progress %s", hferrors.GetErrorMessage(err))
					util.ReturnHTTPMessage(w, r, 500, "error", "error retrieving new session: unable to clean up progress data")
				}
				createdProgress, err := sss.progressClient.CreateProgress(r.Context(), &progresspb.CreateProgressRequest{
					CurrentStep: 0,
					MaxStep:     0,
					TotalStep:   uint32(len(scenario.GetSteps())),
					Scenario:    scenarioid,
					Course:      courseid,
					User:        user.GetId(),
					Labels: map[string]string{
						hflabels.SessionLabel:        sess.GetId(),     // map to session
						hflabels.ScheduledEventLabel: scheduledEventId, // map to scheduledevent
						hflabels.UserLabel:           user.GetId(),     // map to user
						"finished":                   "false",          // default is in progress, finished = false
					},
				})
				if err != nil {
					glog.Errorf("error creating progress %s", hferrors.GetErrorMessage(err))
					util.ReturnHTTPMessage(w, r, 500, "error", "error retrieving new session: unable to recreate progress")
				}
				glog.V(2).Infof("created progress with ID %s", createdProgress.GetId())
			}

			preparedSession := preparedSession{
				Id:           sess.GetId(),
				ScenarioId:   scenarioid,
				CourseId:     sess.GetCourse(),
				KeepCourseVM: sess.GetKeepCourseVm(),
				UserId:       sess.GetUser(),
				VmClaimSet:   sess.GetVmClaim(),
				AccessCode:   sess.GetAccessCode(),
			}
			encodedSS, err := json.Marshal(preparedSession)
			if err != nil {
				glog.Error(err)
			}
			util.ReturnHTTPContent(w, r, 200, "exists", encodedSS)
			return
		}
	}
	schedEvent, err := sss.scheduledEventClient.GetScheduledEvent(r.Context(), &generalpb.GetRequest{Id: scheduledEventId})
	if err != nil {
		if hferrors.IsGrpcNotFound(err) {
			errMsg := fmt.Sprintf("error retrieving new session: scheduled event %s not found", scheduledEventId)
			util.ReturnHTTPMessage(w, r, http.StatusNotFound, "not found", errMsg)
			return
		}
		errMsg := fmt.Sprintf("error retrieving new session: failed to retrieve scheduled event %s", scheduledEventId)
		util.ReturnHTTPMessage(w, r, http.StatusInternalServerError, "internalerror", errMsg)
	}

	var bindMode string
	var baseName string
	if schedEvent.GetOnDemand() {
		bindMode = "dynamic"
		bndp := os.Getenv("HF_BASENAME_DYNAMIC_PREFIX")
		if bndp == "" {
			baseName = "vmc"
		} else {
			baseName = bndp
		}
	} else {
		bindMode = "static"
		bnsp := os.Getenv("HF_BASENAME_SCHEDULED_PREFIX")
		if bnsp == "" {
			baseName = "scheduled"
		} else {
			baseName = bnsp
		}
	}

	wrappedVms := course.GetVms()
	if len(wrappedVms) == 0 {
		wrappedVms = scenario.GetVms()
	}
	vms := util.ConvertToStringMapSlice(wrappedVms)

	lenVms := len(vms)
	sessionVmClaimSet := make([]string, 0, lenVms)
	vmClaimRequests := make([]*vmclaimpb.CreateVMClaimRequest, 0, lenVms)
	for _, vmset := range vms {
		vmClaimId := util.GenerateResourceName(baseName, util.RandStringRunes(10), 10)
		sessionVmClaimSet = append(sessionVmClaimSet, vmClaimId)
		vmClaimRequests = append(vmClaimRequests, &vmclaimpb.CreateVMClaimRequest{
			Id:                  vmClaimId,
			UserName:            user.GetId(),
			Vmset:               vmset,
			RestrictedBind:      restrictedBind,
			RestrictedBindValue: restrictedBindVal,
			DynamicCapable:      true,
			Labels: map[string]string{
				hflabels.SessionLabel:        "",
				hflabels.UserLabel:           user.GetId(),
				hflabels.AccessCodeLabel:     accessCodeObj.GetId(),
				hflabels.ScheduledEventLabel: scheduledEventId,
			},
		})
	}
	keepVm := course.GetKeepVm()

	createdSessionId, err := sss.internalSessionServer.CreateSession(r.Context(), &sessionpb.CreateSessionRequest{
		Scenario:     scenarioid,
		Course:       courseid,
		KeepCourseVm: keepVm,
		User:         user.GetId(),
		VmClaim:      sessionVmClaimSet,
		AccessCode:   accessCodeId,
		Labels: map[string]string{
			hflabels.AccessCodeLabel: accessCodeObj.GetId(),
			hflabels.UserLabel:       user.GetId(),
		},
	})
	if err != nil {
		glog.Errorf("error creating session %s", hferrors.GetErrorMessage(err))
		util.ReturnHTTPMessage(w, r, 500, "error", "error creating new session")
		return
	}
	sessionId := createdSessionId.GetId()
	expirationTime, err := calculateExpiration(newSSTimeout, course.GetKeepaliveDuration(), scenario.GetKeepaliveDuration())
	if err != nil {
		glog.Errorf("Unable to calculate session expiration: %v", err)
		errMsg := "Failed to calculate session expiration. Please check your course/scenario keepalive settings."
		util.ReturnHTTPMessage(w, r, 500, "error", errMsg)
	}
	_, err = sss.internalSessionServer.UpdateSessionStatus(r.Context(), &sessionpb.UpdateSessionStatusRequest{
		Id:             sessionId,
		Active:         wrapperspb.Bool(true),
		Finished:       wrapperspb.Bool(false),
		StartTime:      now.Format(time.UnixDate),
		ExpirationTime: expirationTime,
	})

	if err != nil {
		glog.Errorf("error updating session status: %s", hferrors.GetErrorMessage(err))
		util.ReturnHTTPMessage(w, r, 500, "error", "error while creating session: could not update session status")
		return
	}
	glog.V(2).Infof("created session %s", sessionId)

	for _, vmClaimCreateReq := range vmClaimRequests {
		vmClaimId := vmClaimCreateReq.GetId()
		vmClaimCreateReq.Labels[hflabels.SessionLabel] = sessionId
		_, err = sss.vmclaimClient.CreateVMClaim(r.Context(), vmClaimCreateReq)
		if err != nil {
			glog.Errorf("error creating vm claim: %s", hferrors.GetErrorMessage(err))
			errMsg := fmt.Sprintf("error creating vm claim %s", vmClaimId)
			util.ReturnHTTPMessage(w, r, 500, "error", errMsg)
			return
		}
		_, err = sss.vmclaimClient.UpdateVMClaimStatus(r.Context(), &vmclaimpb.UpdateVMClaimStatusRequest{
			Id:       vmClaimId,
			BindMode: bindMode,
			Bound:    wrapperspb.Bool(false),
			Ready:    wrapperspb.Bool(false),
		})
		if err != nil {
			glog.Errorf("error updating vm claim status %s", hferrors.GetErrorMessage(err))
			util.ReturnHTTPMessage(w, r, 500, "error", "error while creating vm claim: could not update vm claim status")
			return
		}
	}

	createdProgress, err := sss.progressClient.CreateProgress(r.Context(), &progresspb.CreateProgressRequest{
		CurrentStep: 0,
		MaxStep:     0,
		TotalStep:   uint32(len(scenario.GetSteps())),
		Scenario:    scenarioid,
		Course:      courseid,
		User:        user.GetId(),
		Labels: map[string]string{
			hflabels.SessionLabel:        sessionId,        // map to session
			hflabels.ScheduledEventLabel: scheduledEventId, // map to scheduledevent
			hflabels.UserLabel:           user.GetId(),     // map to user
			"finished":                   "false",          // default is in progress, finished = false
		},
	})
	if err != nil {
		glog.Errorf("error creating progress %s", hferrors.GetErrorMessage(err))
		return
	}
	glog.V(2).Infof("created progress with ID %s", createdProgress.GetId())

	preparedSession := preparedSession{
		Id:           sessionId,
		ScenarioId:   scenarioid,
		CourseId:     courseid,
		KeepCourseVM: keepVm,
		UserId:       user.GetId(),
		VmClaimSet:   sessionVmClaimSet,
		AccessCode:   accessCodeId,
	}
	encodedSS, err := json.Marshal(preparedSession)
	if err != nil {
		glog.Error(err)
	}
	util.ReturnHTTPContent(w, r, 201, "created", encodedSS)
}

func (sss SessionServer) FinishProgress(ctx context.Context, sessionId string, userId string) error {
	now := time.Now()

	_, err := sss.progressClient.UpdateCollectionProgress(ctx, &progresspb.UpdateCollectionProgressRequest{
		Labelselector: fmt.Sprintf("%s=%s,%s=%s,finished=false", hflabels.SessionLabel, sessionId, hflabels.UserLabel, userId),
		Finished:      "true",
		LastUpdate:    now.Format(time.UnixDate),
	})
	// If the error is a NotFoundError, let's ignore it. There simply did not exist any progress for this session.
	if err != nil && !hferrors.IsGrpcNotFound(err) {
		glog.Errorf("error while finishing progress %s", hferrors.GetErrorMessage(err))
		return err
	}
	return nil
}

func (sss SessionServer) FinishedSessionFunc(w http.ResponseWriter, r *http.Request) {
	user, err := rbac.AuthenticateRequest(r, sss.authnClient)
	if err != nil {
		util.ReturnHTTPMessage(w, r, 403, "forbidden", "no access to finish sessions")
		return
	}
	impersonatedUserId := user.GetId()

	vars := mux.Vars(r)

	sessionId := vars["session_id"]
	if len(sessionId) == 0 {
		util.ReturnHTTPMessage(w, r, 500, "error", "no session id passed in")
		return
	}

	ss, err := sss.internalSessionServer.GetSession(r.Context(), &generalpb.GetRequest{Id: sessionId, LoadFromCache: true})
	if err != nil {
		util.ReturnHTTPMessage(w, r, 500, "error", "error retrieving session")
		return
	}
	if ss.GetUser() != impersonatedUserId {
		// check if the user has access to write sessions
		authrResponse, err := rbac.AuthorizeSimple(r, sss.authrClient, impersonatedUserId, rbac.HobbyfarmPermission(resourcePlural, rbac.VerbUpdate))
		if err != nil || !authrResponse.Success {
			util.ReturnHTTPMessage(w, r, 403, "forbidden", "access denied to update session")
			return
		}
	}

	now := time.Now().Format(time.UnixDate)

	_, err = sss.internalSessionServer.UpdateSessionStatus(r.Context(), &sessionpb.UpdateSessionStatusRequest{
		Id:             sessionId,
		Active:         wrapperspb.Bool(false),
		ExpirationTime: now,
	})

	if err != nil {
		glog.Errorf("error marking session as expired: %s", hferrors.GetErrorMessage(err))
		util.ReturnHTTPMessage(w, r, 500, "error", "error terminating session")
		return
	}

	util.ReturnHTTPMessage(w, r, 200, "updated", "updated session")
}

func (sss SessionServer) KeepAliveSessionFunc(w http.ResponseWriter, r *http.Request) {
	user, err := rbac.AuthenticateRequest(r, sss.authnClient)
	if err != nil {
		util.ReturnHTTPMessage(w, r, 403, "forbidden", "no access to sessions")
		return
	}

	vars := mux.Vars(r)

	sessionId := vars["session_id"]
	if len(sessionId) == 0 {
		util.ReturnHTTPMessage(w, r, 400, "bad request", "no session id passed in")
		return
	}

	ss, err := sss.internalSessionServer.GetSession(r.Context(), &generalpb.GetRequest{Id: sessionId, LoadFromCache: true})
	if err != nil {
		util.ReturnHTTPMessage(w, r, 500, "error", "error retrieving session")
		return
	}
	if ss.GetUser() != user.GetId() {
		util.ReturnHTTPMessage(w, r, 403, "forbidden", "no session found that matches this user")
		return
	}

	if ss.GetStatus().GetFinished() {
		util.ReturnHTTPMessage(w, r, 404, "notfound", "session was finished")
		return
	}

	if ss.GetStatus().GetPaused() {
		glog.V(4).Infof("session %s was paused, returning paused", ss.GetId())

		now := time.Now()
		pauseExpiration, err := time.Parse(time.UnixDate, ss.GetStatus().GetPausedTime())

		if err != nil {
			glog.Error(err)
			util.ReturnHTTPMessage(w, r, 304, "paused", "session is paused")
			return
		}

		timeUntilExpiration := pauseExpiration.Sub(now)

		util.ReturnHTTPMessage(w, r, 202, "paused", timeUntilExpiration.String())
		return
	}

	sessionAc := ss.GetAccessCode() // the session's access code
	if sessionAc != "" {
		// If we receive an AccessCodeObj from the accessCode Client the AC from this session is still valid, if we find no AccessCode it was deleted or time ran out.
		// We will gracefully end this session by just not increasing the Session ExpirationTime anymore. This will end the session at maximum <KeepAliveDuration> after the AC expired
		_, err := sss.acClient.GetAccessCodeWithOTACs(r.Context(), &generalpb.ResourceId{Id: sessionAc})
		if err != nil {
			util.ReturnHTTPMessage(w, r, 400, "error", "Session is overdue, can not increase duration")
			return
		}
	}

	scenarioId := ss.GetScenario() // the session's scenario ID
	courseId := ss.GetCourse()     // the session's course ID
	course, scenario, ok := sss.getCourseAndScenarioFromCache(w, r, courseId, scenarioId)
	if !ok {
		// we encountered an error and already returned an HTTPMessage in getCourseAndScenarioFromCache()
		return
	}

	expiration, err := calculateExpiration(newSSTimeout, course.GetKeepaliveDuration(), scenario.GetKeepaliveDuration())
	if err != nil {
		glog.Errorf("Unable to calculate session expiration: %v", err)
		errMsg := "Failed to calculate session expiration. Please check your course/scenario keepalive settings."
		util.ReturnHTTPMessage(w, r, 500, "error", errMsg)
	}

	_, err = sss.internalSessionServer.UpdateSessionStatus(r.Context(), &sessionpb.UpdateSessionStatusRequest{
		Id:             sessionId,
		ExpirationTime: expiration,
	})

	if err != nil {
		glog.Errorf("error updating expiration of session status: %s", hferrors.GetErrorMessage(err))
		util.ReturnHTTPMessage(w, r, 500, "error", "Error: Unable to extend session lifetime.")
		return
	}

	util.ReturnHTTPMessage(w, r, 202, "keepalived", "keepalive successful")
}

func (sss SessionServer) PauseSessionFunc(w http.ResponseWriter, r *http.Request) {
	user, err := rbac.AuthenticateRequest(r, sss.authnClient)
	if err != nil {
		util.ReturnHTTPMessage(w, r, 403, "forbidden", "no access to pause sessions")
		return
	}

	vars := mux.Vars(r)

	sessionId := vars["session_id"]
	if len(sessionId) == 0 {
		util.ReturnHTTPMessage(w, r, 500, "bad request", "no session id passed in")
		return
	}

	ss, err := sss.internalSessionServer.GetSession(r.Context(), &generalpb.GetRequest{Id: sessionId, LoadFromCache: true})
	if err != nil {
		util.ReturnHTTPMessage(w, r, 500, "error", "error retrieving session")
		return
	}
	if ss.GetUser() != user.GetId() {
		util.ReturnHTTPMessage(w, r, 403, "forbidden", "no session found that matches this user")
		return
	}

	scenarioId := ss.GetScenario() // the session's scenario ID
	courseId := ss.GetCourse()     // the session's course ID

	course, scenario, ok := sss.getCourseAndScenarioFromCache(w, r, courseId, scenarioId)
	if !ok {
		// we encountered an error and already returned an HTTPMessage in getCourseAndScenarioFromCache()
		return
	}

	// We always use the Getter functions to retrieve struct fields since they provide a fallback if the object is nil
	if !course.GetPausable() && !scenario.GetPausable() {
		// either course and scenario are not pausable or the session does not contain any course/scenario
		glog.Error("session is not pauseable")
		util.ReturnHTTPMessage(w, r, 500, "error", "not pauseable")
		return
	}

	pauseExpiration, err := calculateExpiration(pauseSSTimeout, course.GetPauseDuration(), scenario.GetPauseDuration())
	if err != nil {
		glog.Errorf("Unable to calculate pause expiration: %v", err)
		errMsg := "Failed to calculate pause expiration. Please check your course/scenario pause duration settings."
		util.ReturnHTTPMessage(w, r, 500, "error", errMsg)
	}

	_, err = sss.internalSessionServer.UpdateSessionStatus(r.Context(), &sessionpb.UpdateSessionStatusRequest{
		Id:         sessionId,
		Paused:     wrapperspb.Bool(true),
		PausedTime: wrapperspb.String(pauseExpiration),
	})

	if err != nil {
		glog.Errorf("error updating pause duration of session status %s", hferrors.GetErrorMessage(err))
		util.ReturnHTTPMessage(w, r, 500, "error", "Error: Unable to extend pause duration")
		return
	}

	util.ReturnHTTPMessage(w, r, 204, "updated", "updated session")
}

func (sss SessionServer) ResumeSessionFunc(w http.ResponseWriter, r *http.Request) {
	user, err := rbac.AuthenticateRequest(r, sss.authnClient)
	if err != nil {
		util.ReturnHTTPMessage(w, r, 403, "forbidden", "no access to resume sessions")
		return
	}

	vars := mux.Vars(r)

	sessionId := vars["session_id"]
	if len(sessionId) == 0 {
		util.ReturnHTTPMessage(w, r, 500, "error", "no session id passed in")
		return
	}

	ss, err := sss.internalSessionServer.GetSession(r.Context(), &generalpb.GetRequest{Id: sessionId, LoadFromCache: true})
	if err != nil {
		util.ReturnHTTPMessage(w, r, 500, "error", "error retrieving session")
		return
	}
	if ss.GetUser() != user.GetId() {
		util.ReturnHTTPMessage(w, r, 403, "forbidden", "no session found that matches this user")
		return
	}

	course, scenario, ok := sss.getCourseAndScenarioFromCache(w, r, ss.GetScenario(), ss.GetCourse())
	if !ok {
		// we encountered an error and already returned an HTTPMessage in getCourseAndScenarioFromCache()
		return
	}

	newExpiration, err := calculateExpiration(keepaliveSSTimeout, course.GetKeepaliveDuration(), scenario.GetKeepaliveDuration())
	if err != nil {
		glog.Errorf("Unable to calculate session expiration: %v", err)
		errMsg := "Failed to calculate session expiration. Please check your course/scenario keepalive settings."
		util.ReturnHTTPMessage(w, r, 500, "error", errMsg)
	}

	_, err = sss.internalSessionServer.UpdateSessionStatus(r.Context(), &sessionpb.UpdateSessionStatusRequest{
		Id:             sessionId,
		Paused:         wrapperspb.Bool(false),
		PausedTime:     wrapperspb.String(""),
		ExpirationTime: newExpiration,
	})

	if err != nil {
		glog.Errorf("error updating session status: %s", hferrors.GetErrorMessage(err))
		util.ReturnHTTPMessage(w, r, 500, "error", "Error: Failed to resume session, unable to update session expiration!")
		return
	}

	util.ReturnHTTPMessage(w, r, 204, "updated", "resumed session")
}

func (sss SessionServer) GetSessionFunc(w http.ResponseWriter, r *http.Request) {
	user, err := rbac.AuthenticateRequest(r, sss.authnClient)
	if err != nil {
		util.ReturnHTTPMessage(w, r, 403, "forbidden", "no access to get sessions")
		return
	}
	impersonatedUserId := user.GetId()

	vars := mux.Vars(r)

	sessionId := vars["session_id"]
	if len(sessionId) == 0 {
		util.ReturnHTTPMessage(w, r, 500, "error", "no session id passed in")
		return
	}

	ss, err := sss.internalSessionServer.GetSession(r.Context(), &generalpb.GetRequest{Id: sessionId, LoadFromCache: true})

	if err != nil {
		glog.Errorf("did not find a coressponding session with the given ID")
		util.ReturnHTTPMessage(w, r, http.StatusNotFound, "error", "no session found")
		return
	}

	if ss.GetUser() != impersonatedUserId {
		authrResponse, err := rbac.AuthorizeSimple(r, sss.authrClient, impersonatedUserId, rbac.HobbyfarmPermission(resourcePlural, rbac.VerbGet))
		if err != nil || !authrResponse.Success {
			util.ReturnHTTPMessage(w, r, 403, "forbidden", "no session found that matches this user")
			return
		}
	}

	preparedSession := preparedSession{
		Id:           sessionId,
		ScenarioId:   ss.GetScenario(),
		CourseId:     ss.GetCourse(),
		KeepCourseVM: ss.GetKeepCourseVm(),
		UserId:       ss.GetUser(),
		VmClaimSet:   ss.GetVmClaim(),
		AccessCode:   ss.GetAccessCode(),
	}
	encodedSS, err := json.Marshal(preparedSession)
	if err != nil {
		glog.Error(err)
	}
	util.ReturnHTTPContent(w, r, 200, "success", encodedSS)

	glog.V(2).Infof("retrieved session %s", ss.GetId())
}

// returns (courseObj, scenarioObj, ok)
func (sss SessionServer) getCourseAndScenarioFromCache(
	w http.ResponseWriter,
	r *http.Request,
	courseId string,
	scenarioId string,
) (*coursepb.Course, *scenariopb.Scenario, bool) {
	var scenario *scenariopb.Scenario
	var course *coursepb.Course
	var err error

	if scenarioId != "" {
		scenario, err = sss.scenarioClient.GetScenario(r.Context(), &generalpb.GetRequest{
			Id:            scenarioId,
			LoadFromCache: true,
		})
		if err != nil {
			glog.Errorf("error retrieving scenario: %s", hferrors.GetErrorMessage(err))
			util.ReturnHTTPMessage(w, r, 500, "error", "error getting scenario")
			return nil, nil, false
		}
	}
	if courseId != "" {
		course, err = sss.courseClient.GetCourse(r.Context(), &generalpb.GetRequest{
			Id:            courseId,
			LoadFromCache: true,
		})
		if err != nil {
			glog.Errorf("error retrieving course: %s", hferrors.GetErrorMessage(err))
			util.ReturnHTTPMessage(w, r, 500, "error", "error getting course")
			return nil, nil, false
		}
	}
	return course, scenario, true
}

// this function can be used to either calculate the session expiration or the pause expiration
func calculateExpiration(defaultVal string, courseDuration string, scenarioDuration string) (string, error) {
	ssTimeout := keepaliveSSTimeout // the default value if course/scenario keepalive is not set

	courseKeepAlive := courseDuration // course keepalive/pause duration takes precedence over scenario's keepalive/pause duration
	scenarioKeepAlive := scenarioDuration
	if courseKeepAlive != "" {
		ssTimeout = courseKeepAlive
	} else if scenarioKeepAlive != "" {
		ssTimeout = scenarioKeepAlive
	}

	now := time.Now()
	duration, err := time.ParseDuration(ssTimeout)
	if err != nil {
		return "", err
	}

	newExpiration := now.Add(duration).Format(time.UnixDate)
	return newExpiration, nil
}
