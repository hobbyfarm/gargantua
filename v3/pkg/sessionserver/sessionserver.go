package sessionserver

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/hobbyfarm/gargantua/v3/pkg/accesscode"
	hfv1 "github.com/hobbyfarm/gargantua/v3/pkg/apis/hobbyfarm.io/v1"
	hfClientset "github.com/hobbyfarm/gargantua/v3/pkg/client/clientset/versioned"
	hfInformers "github.com/hobbyfarm/gargantua/v3/pkg/client/informers/externalversions"
	"github.com/hobbyfarm/gargantua/v3/pkg/courseclient"
	hflabels "github.com/hobbyfarm/gargantua/v3/pkg/labels"
	rbac2 "github.com/hobbyfarm/gargantua/v3/pkg/rbac"
	"github.com/hobbyfarm/gargantua/v3/pkg/scenarioclient"
	util2 "github.com/hobbyfarm/gargantua/v3/pkg/util"

	authnpb "github.com/hobbyfarm/gargantua/v3/protos/authn"
	authrpb "github.com/hobbyfarm/gargantua/v3/protos/authr"

	"github.com/golang/glog"
	"github.com/gorilla/mux"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/retry"
)

const (
	ssIndex            = "sss.hobbyfarm.io/session-id-index"
	newSSTimeout       = "5m"
	keepaliveSSTimeout = "5m"
	pauseSSTimeout     = "2h"
	resourcePlural     = rbac2.ResourcePluralSession
)

type SessionServer struct {
	authnClient      authnpb.AuthNClient
	authrClient      authrpb.AuthRClient
	hfClientSet      hfClientset.Interface
	courseClient     *courseclient.CourseClient
	scenarioClient   *scenarioclient.ScenarioClient
	accessCodeClient *accesscode.AccessCodeClient
	ssIndexer        cache.Indexer
	ctx              context.Context
}

type preparedSession struct {
	ID string `json:"id"`
	hfv1.SessionSpec
}

func NewSessionServer(authnClient authnpb.AuthNClient, authrClient authrpb.AuthRClient, accessCodeClient *accesscode.AccessCodeClient, scenarioClient *scenarioclient.ScenarioClient, courseClient *courseclient.CourseClient, hfClientSet hfClientset.Interface, hfInformerFactory hfInformers.SharedInformerFactory, ctx context.Context) (*SessionServer, error) {
	a := SessionServer{}
	a.authnClient = authnClient
	a.authrClient = authrClient
	a.hfClientSet = hfClientSet
	a.courseClient = courseClient
	a.scenarioClient = scenarioClient
	a.accessCodeClient = accessCodeClient
	inf := hfInformerFactory.Hobbyfarm().V1().Sessions().Informer()
	indexers := map[string]cache.IndexFunc{ssIndex: ssIdIndexer}
	inf.AddIndexers(indexers)
	a.ssIndexer = inf.GetIndexer()
	a.ctx = ctx

	return &a, nil
}

func (sss SessionServer) SetupRoutes(r *mux.Router) {
	r.HandleFunc("/session/new", sss.NewSessionFunc).Methods("POST")
	r.HandleFunc("/session/{session_id}", sss.GetSessionFunc).Methods("GET")
	r.HandleFunc("/session/{session_id}/finished", sss.FinishedSessionFunc).Methods("PUT")
	r.HandleFunc("/session/{session_id}/keepalive", sss.KeepAliveSessionFunc).Methods("PUT")
	r.HandleFunc("/session/{session_id}/pause", sss.PauseSessionFunc).Methods("PUT")
	r.HandleFunc("/session/{session_id}/resume", sss.ResumeSessionFunc).Methods("PUT")
	glog.V(2).Infof("set up routes for session server")
}

func (sss SessionServer) NewSessionFunc(w http.ResponseWriter, r *http.Request) {
	user, err := rbac2.AuthenticateRequest(r, sss.authnClient)
	if err != nil {
		util2.ReturnHTTPMessage(w, r, 403, "forbidden", "no access to create sessions")
		return
	}

	courseid := r.PostFormValue("course")
	scenarioid := r.PostFormValue("scenario")

	if courseid == "" && scenarioid == "" {
		util2.ReturnHTTPMessage(w, r, 500, "error", "no course/scenario id passed in")
		return
	}

	accessCode := r.PostFormValue("access_code")

	restrictedBind := false
	restrictedBindVal := ""

	if accessCode == "" {
		util2.ReturnHTTPMessage(w, r, 400, "error", "An accesscode has to be given in order so start a session")
		return
	}

	// we should validate the user can use this access code
	// let's figure out the restricted bind value
	accessCodeObj, err := sss.accessCodeClient.GetAccessCodeWithOTACs(accessCode)
	if err != nil {
		util2.ReturnHTTPMessage(w, r, 500, "error", "could not retrieve access code")
		return
	}
	if accessCodeObj.Spec.RestrictedBind {
		restrictedBind = accessCodeObj.Spec.RestrictedBind
		restrictedBindVal = accessCodeObj.Spec.RestrictedBindValue
	}

	random := util2.RandStringRunes(10)
	var course hfv1.Course
	var scenario hfv1.Scenario

	// get the course and/or scenario objects
	if courseid != "" {
		course, err = sss.courseClient.GetCourseById(courseid)
		if err != nil {
			glog.Errorf("course not found %v", err)
			util2.ReturnHTTPMessage(w, r, 500, "error", "no course found")
			return
		}
	}
	if scenarioid != "" {
		scenario, err = sss.scenarioClient.GetScenarioById(scenarioid)
		if err != nil {
			glog.Errorf("scenario not found %v", err)
			util2.ReturnHTTPMessage(w, r, 500, "error", "no scenario found")
			return
		}
	}

	// now we should check for existing sessions for the user
	sessions, err := sss.hfClientSet.HobbyfarmV1().Sessions(util2.GetReleaseNamespace()).List(sss.ctx, metav1.ListOptions{
		LabelSelector: fmt.Sprintf("%s=%s", hflabels.UserLabel, user.GetId()),
	})

	if err != nil {
		glog.Error(err)
	}
	now := time.Now()

	// should we check the sessions list for the restricted bind value and match if one is passed in? probably...
	for _, v := range sessions.Items {
		expires, err := time.Parse(time.UnixDate, v.Status.ExpirationTime)
		if err != nil {
			continue
		}
		if v.Spec.UserId == user.GetId() &&
			(v.Spec.CourseId == course.Name || v.Spec.ScenarioId == scenario.Name) &&
			!v.Status.Finished &&
			v.Status.Active && expires.After(now) {
			// we should just return this session...

			// if this is a course, return the same scenario id that was given to us
			// i.e., reuse the course id and give them the scenario they asked for
			if v.Spec.CourseId != "" {
				v.Spec.ScenarioId = scenarioid

				retryErr := retry.RetryOnConflict(retry.DefaultRetry, func() error {
					result, getErr := sss.hfClientSet.HobbyfarmV1().Sessions(util2.GetReleaseNamespace()).Get(sss.ctx, v.Name, metav1.GetOptions{})
					if getErr != nil {
						return fmt.Errorf("error retrieving latest version of session %s: %v", v.Name, getErr)
					}

					result.Spec.ScenarioId = scenarioid

					_, updateErr := sss.hfClientSet.HobbyfarmV1().Sessions(util2.GetReleaseNamespace()).Update(sss.ctx, result, metav1.UpdateOptions{})
					glog.V(4).Infof("updated session for new scenario")

					//finish old progress & create new progress for the new scenario
					sss.FinishProgress(result.Name, user.GetId())
					sss.CreateProgress(result.Name, accessCodeObj.Labels[hflabels.ScheduledEventLabel], scenario.Name, course.Name, user.GetId(), len(scenario.Spec.Steps))

					return updateErr
				})

				if retryErr != nil {
					glog.Errorf("error updating session %v", err)
					util2.ReturnHTTPMessage(w, r, 500, "error", "something happened")
					return
				}

			}

			preparedSession := preparedSession{v.Name, v.Spec}
			encodedSS, err := json.Marshal(preparedSession)
			if err != nil {
				glog.Error(err)
			}
			util2.ReturnHTTPContent(w, r, 200, "exists", encodedSS)
			return
		}

	}

	sessionName := util2.GenerateResourceName("ss", random, 10)
	session := hfv1.Session{}

	session.Name = sessionName
	session.Spec.CourseId = course.Name
	session.Spec.ScenarioId = scenario.Name
	session.Spec.UserId = user.GetId()
	session.Spec.KeepCourseVM = course.Spec.KeepVM
	session.Spec.AccessCode = accessCode // accessCode can be an OTAC or a normal AccessCode
	labels := make(map[string]string)
	labels[hflabels.AccessCodeLabel] = accessCodeObj.Name // map accesscode to session, this has to be the SE AccessCode in order for session cleanup to work upon SE deletion
	labels[hflabels.UserLabel] = user.GetId()             // map user to session
	session.Labels = labels
	var vms []map[string]string
	if course.Spec.VirtualMachines != nil {
		vms = course.Spec.VirtualMachines
	} else {
		vms = scenario.Spec.VirtualMachines
	}

	// find bindMode by quering the scheduledEvent
	owners := accessCodeObj.GetOwnerReferences()
	if len(owners) != 1 {
		util2.ReturnHTTPMessage(w, r, 500, "error", "access code has multiple owners.. invalid request")
		return
	}

	schedEvent, err := sss.hfClientSet.HobbyfarmV1().ScheduledEvents(util2.GetReleaseNamespace()).Get(sss.ctx, owners[0].Name, metav1.GetOptions{})
	if err != nil {
		util2.ReturnHTTPMessage(w, r, 500, "error", "unable to find scheduledEvent")
		return
	}

	var bindMode string
	var baseName string
	if schedEvent.Spec.OnDemand {
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

	session.Spec.VmClaimSet = make([]string, len(vms))
	for index, vmset := range vms {
		virtualMachineClaim := hfv1.VirtualMachineClaim{}
		vmcId := util2.GenerateResourceName(baseName, util2.RandStringRunes(10), 10)
		labels := make(map[string]string)
		labels[hflabels.SessionLabel] = session.Name // map vmc to session
		labels[hflabels.UserLabel] = user.GetId()    // map session to user in a way that is searchable
		labels[hflabels.AccessCodeLabel] = session.Labels[hflabels.AccessCodeLabel]
		labels[hflabels.ScheduledEventLabel] = schedEvent.Name
		virtualMachineClaim.Labels = labels
		virtualMachineClaim.Spec.BaseName = vmcId
		virtualMachineClaim.Name = vmcId
		virtualMachineClaim.Spec.VirtualMachines = make(map[string]hfv1.VirtualMachineClaimVM)
		for vmName, vmTemplateName := range vmset {
			virtualMachineClaim.Spec.VirtualMachines[vmName] = hfv1.VirtualMachineClaimVM{Template: vmTemplateName, VirtualMachineId: ""}
			// also label this vmc so we can query against it later
			labels[fmt.Sprintf("virtualmachinetemplate.hobbyfarm.io/%s", vmTemplateName)] = "true"
		}
		virtualMachineClaim.Spec.UserId = user.GetId()

		virtualMachineClaim.Spec.DynamicCapable = true

		if restrictedBind {
			virtualMachineClaim.Spec.RestrictedBind = restrictedBind
			virtualMachineClaim.Spec.RestrictedBindValue = restrictedBindVal
		} else {
			virtualMachineClaim.Spec.RestrictedBind = false
		}

		createdVmClaim, err := sss.hfClientSet.HobbyfarmV1().VirtualMachineClaims(util2.GetReleaseNamespace()).Create(sss.ctx, &virtualMachineClaim, metav1.CreateOptions{})
		if err != nil {
			glog.Errorf("error creating vm claim %v", err)
			util2.ReturnHTTPMessage(w, r, 500, "error", "something happened")
			return
		}

		createdVmClaim.Status.Bound = false
		createdVmClaim.Status.Ready = false
		createdVmClaim.Status.BindMode = bindMode

		_, err = sss.hfClientSet.HobbyfarmV1().VirtualMachineClaims(util2.GetReleaseNamespace()).UpdateStatus(sss.ctx, createdVmClaim, metav1.UpdateOptions{})
		if err != nil {
			glog.Errorf("error updating vm claim status %v", err)
			util2.ReturnHTTPMessage(w, r, 500, "error", "something happened")
			return
		}

		session.Spec.VmClaimSet[index] = createdVmClaim.Name
	}

	var ssTimeout string

	if course.Spec.KeepAliveDuration != "" {
		ssTimeout = course.Spec.KeepAliveDuration
	} else if scenario.Spec.KeepAliveDuration != "" {
		ssTimeout = scenario.Spec.KeepAliveDuration
	} else {
		ssTimeout = newSSTimeout
	}

	createdSession, err := sss.hfClientSet.HobbyfarmV1().Sessions(util2.GetReleaseNamespace()).Create(sss.ctx, &session, metav1.CreateOptions{})

	if err != nil {
		glog.Errorf("error creating session %v", err)
		util2.ReturnHTTPMessage(w, r, 500, "error", "something happened")
		return
	}

	createdSession.Status.StartTime = now.Format(time.UnixDate)
	duration, _ := time.ParseDuration(ssTimeout)

	createdSession.Status.ExpirationTime = now.Add(duration).Format(time.UnixDate)
	createdSession.Status.Active = true
	createdSession.Status.Finished = false

	_, err = sss.hfClientSet.HobbyfarmV1().Sessions(util2.GetReleaseNamespace()).UpdateStatus(sss.ctx, createdSession, metav1.UpdateOptions{})

	if err != nil {
		glog.Errorf("error creating session %v", err)
		util2.ReturnHTTPMessage(w, r, 500, "error", "something happened")
		return
	}

	glog.V(2).Infof("created session ID %s", createdSession.Name)

	sss.CreateProgress(createdSession.Name, accessCodeObj.Labels[hflabels.ScheduledEventLabel], scenario.Name, course.Name, user.GetId(), len(scenario.Spec.Steps))

	preparedSession := preparedSession{createdSession.Name, createdSession.Spec}
	encodedSS, err := json.Marshal(preparedSession)
	if err != nil {
		glog.Error(err)
	}
	util2.ReturnHTTPContent(w, r, 201, "created", encodedSS)
}

func (sss SessionServer) CreateProgress(sessionId string, scheduledEventId string, scenarioId string, courseId string, userId string, totalSteps int) {
	random := util2.RandStringRunes(16)
	now := time.Now()

	progressName := util2.GenerateResourceName("progress", random, 16)
	progress := hfv1.Progress{}

	progress.Name = progressName
	progress.Spec.Course = courseId
	progress.Spec.Scenario = scenarioId
	progress.Spec.UserId = userId
	progress.Spec.Started = now.Format(time.UnixDate)
	progress.Spec.LastUpdate = now.Format(time.UnixDate)
	progress.Spec.Finished = "false"
	progress.Spec.TotalStep = totalSteps
	progress.Spec.MaxStep = 0
	progress.Spec.CurrentStep = 0

	steps := []hfv1.ProgressStep{}
	step := hfv1.ProgressStep{Step: 0, Timestamp: now.Format(time.UnixDate)}
	steps = append(steps, step)
	progress.Spec.Steps = steps

	labels := make(map[string]string)
	labels[hflabels.SessionLabel] = sessionId               // map to session
	labels[hflabels.ScheduledEventLabel] = scheduledEventId // map to scheduledevent
	labels[hflabels.UserLabel] = userId                     // map to scheduledevent
	labels["finished"] = "false"                            // default is in progress, finished = false
	progress.Labels = labels

	createdProgress, err := sss.hfClientSet.HobbyfarmV1().Progresses(util2.GetReleaseNamespace()).Create(sss.ctx, &progress, metav1.CreateOptions{})

	if err != nil {
		glog.Errorf("error creating progress %v", err)
		return
	}

	glog.V(2).Infof("created progress with ID %s", createdProgress.Name)
}

func (sss SessionServer) FinishProgress(sessionId string, userId string) {
	now := time.Now()

	progress, err := sss.hfClientSet.HobbyfarmV1().Progresses(util2.GetReleaseNamespace()).List(sss.ctx, metav1.ListOptions{
		LabelSelector: fmt.Sprintf("%s=%s,%s=%s,finished=false", hflabels.SessionLabel, sessionId, hflabels.UserLabel, userId)})

	if err != nil {
		glog.Errorf("error while retrieving progress %v", err)
		return
	}

	for _, p := range progress.Items {
		retryErr := retry.RetryOnConflict(retry.DefaultRetry, func() error {
			p.Labels["finished"] = "true"
			p.Spec.LastUpdate = now.Format(time.UnixDate)
			p.Spec.Finished = "true"

			_, updateErr := sss.hfClientSet.HobbyfarmV1().Progresses(util2.GetReleaseNamespace()).Update(sss.ctx, &p, metav1.UpdateOptions{})
			glog.V(4).Infof("updated progress with ID %s", p.Name)

			return updateErr
		})
		if retryErr != nil {
			glog.Errorf("error finishing progress %v", err)
			return
		}
	}
}

func (sss SessionServer) FinishedSessionFunc(w http.ResponseWriter, r *http.Request) {
	user, err := rbac2.AuthenticateRequest(r, sss.authnClient)
	if err != nil {
		util2.ReturnHTTPMessage(w, r, 403, "forbidden", "no access to finish sessions")
		return
	}

	vars := mux.Vars(r)

	sessionId := vars["session_id"]
	if len(sessionId) == 0 {
		util2.ReturnHTTPMessage(w, r, 500, "error", "no session id passed in")
		return
	}

	ss, err := sss.GetSessionById(sessionId)
	if ss.Spec.UserId != user.Id {
		// check if the user has access to write sessions
		impersonatedUserId := user.GetId()
		authrResponse, err := rbac2.AuthorizeSimple(r, sss.authrClient, impersonatedUserId, rbac2.HobbyfarmPermission(resourcePlural, rbac2.VerbUpdate))
		if err != nil || !authrResponse.Success {
			util2.ReturnHTTPMessage(w, r, 403, "forbidden", "access denied to update session")
			return
		}
	}

	now := time.Now().Format(time.UnixDate)

	retryErr := retry.RetryOnConflict(retry.DefaultRetry, func() error {
		result, getErr := sss.hfClientSet.HobbyfarmV1().Sessions(util2.GetReleaseNamespace()).Get(sss.ctx, sessionId, metav1.GetOptions{})
		if getErr != nil {
			return fmt.Errorf("error retrieving latest version of session %s: %v", sessionId, getErr)
		}

		// Change the expiration time to now, the sessionController will clean up the session
		result.Status.ExpirationTime = now
		result.Status.Active = false

		_, updateErr := sss.hfClientSet.HobbyfarmV1().Sessions(util2.GetReleaseNamespace()).UpdateStatus(sss.ctx, result, metav1.UpdateOptions{})
		glog.V(4).Infof("updated result for environment")

		return updateErr
	})

	if retryErr != nil {
		glog.Errorf("error deleting session %v", err)
		util2.ReturnHTTPMessage(w, r, 500, "error", "something happened")
		return
	}

	util2.ReturnHTTPMessage(w, r, 200, "updated", "updated session")
}

func (sss SessionServer) KeepAliveSessionFunc(w http.ResponseWriter, r *http.Request) {
	user, err := rbac2.AuthenticateRequest(r, sss.authnClient)
	if err != nil {
		util2.ReturnHTTPMessage(w, r, 403, "forbidden", "no access to sessions")
		return
	}

	vars := mux.Vars(r)

	sessionId := vars["session_id"]
	if len(sessionId) == 0 {
		util2.ReturnHTTPMessage(w, r, 500, "error", "no session id passed in")
		return
	}

	ss, err := sss.GetSessionById(sessionId)
	if ss.Spec.UserId != user.Id {
		util2.ReturnHTTPMessage(w, r, 403, "forbidden", "no session found that matches this user")
		return
	}

	if ss.Status.Finished {
		util2.ReturnHTTPMessage(w, r, 404, "notfound", "session was finished")
		return
	}

	if ss.Status.Paused {
		glog.V(4).Infof("session %s was paused, returning paused", ss.Name)

		now := time.Now()
		pauseExpiration, err := time.Parse(time.UnixDate, ss.Status.PausedTime)

		if err != nil {
			glog.Error(err)
			util2.ReturnHTTPMessage(w, r, 304, "paused", "session is paused")
			return
		}

		timeUntilExpiration := pauseExpiration.Sub(now)

		util2.ReturnHTTPMessage(w, r, 202, "paused", timeUntilExpiration.String())
		return
	}

	if ss.Spec.AccessCode != "" {
		// If we receive an AccessCodeObj from the accessCode Client the AC from this session is still valid, if we find no AccessCode it was deleted or time ran out.
		// We will gracefully end this session by just not increasing the Session ExpirationTime anymore. This will end the session at maximum <KeepAliveDuration> after the AC expired
		_, err := sss.accessCodeClient.GetAccessCodeWithOTACs(ss.Spec.AccessCode)
		if err != nil {
			util2.ReturnHTTPMessage(w, r, 400, "error", "Session is overdue, can not increase duration")
			return
		}
	}

	var scenario hfv1.Scenario
	var course hfv1.Course

	if ss.Spec.ScenarioId != "" {
		scenario, err = sss.scenarioClient.GetScenarioById(ss.Spec.ScenarioId)
		if err != nil {
			glog.Errorf("error retrieving scenario %v", err)
			util2.ReturnHTTPMessage(w, r, 500, "error", "error getting scenario")
			return
		}
	}
	if ss.Spec.CourseId != "" {
		course, err = sss.courseClient.GetCourseById(ss.Spec.CourseId)
		if err != nil {
			glog.Errorf("error retrieving course %v", err)
			util2.ReturnHTTPMessage(w, r, 500, "error", "error getting course")
			return
		}
	}

	var ssTimeout string

	if course.Spec.KeepAliveDuration != "" {
		ssTimeout = course.Spec.KeepAliveDuration
	} else if scenario.Spec.KeepAliveDuration != "" {
		ssTimeout = scenario.Spec.KeepAliveDuration
	} else {
		ssTimeout = newSSTimeout
	}

	now := time.Now()
	duration, _ := time.ParseDuration(ssTimeout)

	expiration := now.Add(duration).Format(time.UnixDate)

	retryErr := retry.RetryOnConflict(retry.DefaultRetry, func() error {
		result, getErr := sss.hfClientSet.HobbyfarmV1().Sessions(util2.GetReleaseNamespace()).Get(sss.ctx, sessionId, metav1.GetOptions{})
		if getErr != nil {
			return fmt.Errorf("error retrieving latest version of session %s: %v", sessionId, getErr)
		}

		result.Status.ExpirationTime = expiration

		_, updateErr := sss.hfClientSet.HobbyfarmV1().Sessions(util2.GetReleaseNamespace()).UpdateStatus(sss.ctx, result, metav1.UpdateOptions{})
		glog.V(4).Infof("updated expiration time for session")

		return updateErr
	})

	if retryErr != nil {
		glog.Errorf("error creating session %v", err)
		util2.ReturnHTTPMessage(w, r, 500, "error", "something happened")
		return
	}

	util2.ReturnHTTPMessage(w, r, 202, "keepalived", "keepalive successful")
}

func (sss SessionServer) PauseSessionFunc(w http.ResponseWriter, r *http.Request) {
	user, err := rbac2.AuthenticateRequest(r, sss.authnClient)
	if err != nil {
		util2.ReturnHTTPMessage(w, r, 403, "forbidden", "no access to pause sessions")
		return
	}

	vars := mux.Vars(r)

	sessionId := vars["session_id"]
	if len(sessionId) == 0 {
		util2.ReturnHTTPMessage(w, r, 500, "error", "no session id passed in")
		return
	}

	ss, err := sss.GetSessionById(sessionId)
	if ss.Spec.UserId != user.Id {
		util2.ReturnHTTPMessage(w, r, 403, "forbidden", "no session found that matches this user")
		return
	}

	var course hfv1.Course
	var scenario hfv1.Scenario

	if ss.Spec.CourseId != "" {
		course, err = sss.courseClient.GetCourseById(ss.Spec.CourseId)
		if err != nil {
			glog.Errorf("error retrieving course %v", err)
			util2.ReturnHTTPMessage(w, r, 500, "error", "error getting course")
			return
		}
	}
	if ss.Spec.ScenarioId != "" {
		scenario, err = sss.scenarioClient.GetScenarioById(ss.Spec.ScenarioId)
		if err != nil {
			glog.Errorf("error retrieving scenario %v", err)
			util2.ReturnHTTPMessage(w, r, 500, "error", "error getting scenario")
			return
		}
	}

	if !course.Spec.Pauseable && !scenario.Spec.Pauseable {
		glog.Errorf("session is not pauseable %s", course.Name)
		util2.ReturnHTTPMessage(w, r, 500, "error", "not pauseable")
		return
	}

	var ssTimeout string

	if course.Spec.PauseDuration != "" {
		ssTimeout = course.Spec.PauseDuration
	} else if scenario.Spec.PauseDuration != "" {
		ssTimeout = scenario.Spec.PauseDuration
	} else {
		ssTimeout = pauseSSTimeout
	}

	now := time.Now()
	duration, _ := time.ParseDuration(ssTimeout)

	pauseExpiration := now.Add(duration).Format(time.UnixDate)

	retryErr := retry.RetryOnConflict(retry.DefaultRetry, func() error {
		result, getErr := sss.hfClientSet.HobbyfarmV1().Sessions(util2.GetReleaseNamespace()).Get(sss.ctx, sessionId, metav1.GetOptions{})
		if getErr != nil {
			return fmt.Errorf("error retrieving latest version of session %s: %v", sessionId, getErr)
		}

		result.Status.PausedTime = pauseExpiration
		result.Status.Paused = true

		_, updateErr := sss.hfClientSet.HobbyfarmV1().Sessions(util2.GetReleaseNamespace()).UpdateStatus(sss.ctx, result, metav1.UpdateOptions{})
		glog.V(4).Infof("updated result for course session")

		return updateErr
	})

	if retryErr != nil {
		glog.Errorf("error creating session %v", err)
		util2.ReturnHTTPMessage(w, r, 500, "error", "something happened")
		return
	}

	util2.ReturnHTTPMessage(w, r, 204, "updated", "updated session")
}

func (sss SessionServer) ResumeSessionFunc(w http.ResponseWriter, r *http.Request) {
	user, err := rbac2.AuthenticateRequest(r, sss.authnClient)
	if err != nil {
		util2.ReturnHTTPMessage(w, r, 403, "forbidden", "no access to resume sessions")
		return
	}

	vars := mux.Vars(r)

	sessionId := vars["session_id"]
	if len(sessionId) == 0 {
		util2.ReturnHTTPMessage(w, r, 500, "error", "no session id passed in")
		return
	}

	ss, err := sss.GetSessionById(sessionId)
	if ss.Spec.UserId != user.Id {
		util2.ReturnHTTPMessage(w, r, 403, "forbidden", "no session found that matches this user")
		return
	}

	var course hfv1.Course
	var scenario hfv1.Scenario

	if ss.Spec.CourseId != "" {
		course, err = sss.courseClient.GetCourseById(ss.Spec.CourseId)
		if err != nil {
			glog.Errorf("error retrieving course %v", err)
			util2.ReturnHTTPMessage(w, r, 500, "error", "error getting course")
			return
		}
	}
	if ss.Spec.ScenarioId != "" {
		scenario, err = sss.scenarioClient.GetScenarioById(ss.Spec.ScenarioId)
		if err != nil {
			glog.Errorf("error retrieving scenario %v", err)
			util2.ReturnHTTPMessage(w, r, 500, "error", "error getting scenario")
			return
		}
	}

	var ssTimeout string

	if course.Spec.KeepAliveDuration != "" {
		ssTimeout = course.Spec.KeepAliveDuration
	} else if scenario.Spec.KeepAliveDuration != "" {
		ssTimeout = scenario.Spec.KeepAliveDuration
	} else {
		ssTimeout = keepaliveSSTimeout
	}

	now := time.Now()
	duration, _ := time.ParseDuration(ssTimeout)

	newExpiration := now.Add(duration).Format(time.UnixDate)

	retryErr := retry.RetryOnConflict(retry.DefaultRetry, func() error {
		result, getErr := sss.hfClientSet.HobbyfarmV1().Sessions(util2.GetReleaseNamespace()).Get(sss.ctx, sessionId, metav1.GetOptions{})
		if getErr != nil {
			return fmt.Errorf("error retrieving latest version of session %s: %v", sessionId, getErr)
		}

		result.Status.PausedTime = ""
		result.Status.ExpirationTime = newExpiration
		result.Status.Paused = false

		_, updateErr := sss.hfClientSet.HobbyfarmV1().Sessions(util2.GetReleaseNamespace()).UpdateStatus(sss.ctx, result, metav1.UpdateOptions{})
		glog.V(4).Infof("updated result for session")

		return updateErr
	})

	if retryErr != nil {
		glog.Errorf("error creating session %v", err)
		util2.ReturnHTTPMessage(w, r, 500, "error", "something happened")
		return
	}

	util2.ReturnHTTPMessage(w, r, 204, "updated", "resumed session")
}

func (sss SessionServer) GetSessionFunc(w http.ResponseWriter, r *http.Request) {
	user, err := rbac2.AuthenticateRequest(r, sss.authnClient)
	if err != nil {
		util2.ReturnHTTPMessage(w, r, 403, "forbidden", "no access to get sessions")
		return
	}

	vars := mux.Vars(r)

	sessionId := vars["session_id"]
	if len(sessionId) == 0 {
		util2.ReturnHTTPMessage(w, r, 500, "error", "no session id passed in")
		return
	}

	ss, err := sss.GetSessionById(sessionId)

	if err != nil {
		glog.Errorf("did not find a coressponding session with the given ID")
		util2.ReturnHTTPMessage(w, r, http.StatusNotFound, "error", "no session found")
		return
	}

	if ss.Spec.UserId != user.Id {
		impersonatedUserId := user.GetId()
		authrResponse, err := rbac2.AuthorizeSimple(r, sss.authrClient, impersonatedUserId, rbac2.HobbyfarmPermission(resourcePlural, rbac2.VerbGet))
		if err != nil || !authrResponse.Success {
			util2.ReturnHTTPMessage(w, r, 403, "forbidden", "no session found that matches for this user")
			return
		}
	}

	preparedSession := preparedSession{ss.Name, ss.Spec}
	encodedSS, err := json.Marshal(preparedSession)
	if err != nil {
		glog.Error(err)
	}
	util2.ReturnHTTPContent(w, r, 200, "success", encodedSS)

	glog.V(2).Infof("retrieved session %s", ss.Name)
}

func ssIdIndexer(obj interface{}) ([]string, error) {
	ss, ok := obj.(*hfv1.Session)
	if !ok {
		return []string{}, nil
	}
	return []string{ss.Name}, nil
}

func (sss SessionServer) GetSessionById(id string) (hfv1.Session, error) {
	if len(id) == 0 {
		return hfv1.Session{}, fmt.Errorf("id passed in was empty")
	}

	obj, err := sss.ssIndexer.ByIndex(ssIndex, id)
	if err != nil {
		return hfv1.Session{}, fmt.Errorf("error while retrieving session by id: %s with error: %v", id, err)
	}

	if len(obj) < 1 {
		return hfv1.Session{}, fmt.Errorf("session server not found by id: %s", id)
	}

	Session, ok := obj[0].(*hfv1.Session)

	if !ok {
		return hfv1.Session{}, fmt.Errorf("error while converting session found by id to object: %s", id)
	}

	return *Session, nil
}
