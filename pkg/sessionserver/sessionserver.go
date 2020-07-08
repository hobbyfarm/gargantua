package sessionserver

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/golang/glog"
	"github.com/gorilla/mux"
	"github.com/hobbyfarm/gargantua/pkg/accesscode"
	hfv1 "github.com/hobbyfarm/gargantua/pkg/apis/hobbyfarm.io/v1"
	"github.com/hobbyfarm/gargantua/pkg/authclient"
	hfClientset "github.com/hobbyfarm/gargantua/pkg/client/clientset/versioned"
	hfInformers "github.com/hobbyfarm/gargantua/pkg/client/informers/externalversions"
	"github.com/hobbyfarm/gargantua/pkg/courseclient"
	"github.com/hobbyfarm/gargantua/pkg/scenarioclient"
	"github.com/hobbyfarm/gargantua/pkg/util"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/retry"
)

const (
	ssIndex            = "sss.hobbyfarm.io/session-id-index"
	newSSTimeout       = "5m"
	keepaliveSSTimeout = "5m"
	pauseSSTimeout     = "2h"
	vmcSessionLabel    = "hobbyfarm.io/session"
)

type SessionServer struct {
	hfClientSet      *hfClientset.Clientset
	courseClient     *courseclient.CourseClient
	scenarioClient   *scenarioclient.ScenarioClient
	accessCodeClient *accesscode.AccessCodeClient
	auth             *authclient.AuthClient
	ssIndexer        cache.Indexer
}

func NewSessionServer(authClient *authclient.AuthClient, accessCodeClient *accesscode.AccessCodeClient, scenarioClient *scenarioclient.ScenarioClient, courseClient *courseclient.CourseClient, hfClientSet *hfClientset.Clientset, hfInformerFactory hfInformers.SharedInformerFactory) (*SessionServer, error) {
	a := SessionServer{}
	a.hfClientSet = hfClientSet
	a.courseClient = courseClient
	a.scenarioClient = scenarioClient
	a.auth = authClient
	a.accessCodeClient = accessCodeClient
	inf := hfInformerFactory.Hobbyfarm().V1().Sessions().Informer()
	indexers := map[string]cache.IndexFunc{ssIndex: ssIdIndexer}
	inf.AddIndexers(indexers)
	a.ssIndexer = inf.GetIndexer()
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
	user, err := sss.auth.AuthN(w, r)
	if err != nil {
		util.ReturnHTTPMessage(w, r, 403, "forbidden", "no access to create sessions")
		return
	}

	var id string
	courseid := r.PostFormValue("course")
	scenarioid := r.PostFormValue("scenario")
	if courseid != "" {
		id = courseid
	} else {
		id = scenarioid
	}

	accessCode := r.PostFormValue("access_code")
	glog.V(4).Infof("access code passed in was %s", accessCode)

	restrictedBind := false
	restrictedBindVal := ""

	if accessCode == "" {
		accessCode, err = sss.accessCodeClient.GetClosestAccessCode(user.Spec.Id, id)
		if err != nil {
			glog.Error(err)
			util.ReturnHTTPMessage(w, r, 500, "error", "error retrieving access code applicable to course/scenario")
			return
		}
	}

	// we should validate the user can use this access code
	// let's figure out the restricted bind value
	accessCodeObj, err := sss.hfClientSet.HobbyfarmV1().AccessCodes().Get(accessCode, metav1.GetOptions{})
	if err != nil {
		util.ReturnHTTPMessage(w, r, 500, "error", "could not retrieve access code")
		return
	}
	if accessCodeObj.Spec.RestrictedBind {
		restrictedBind = accessCodeObj.Spec.RestrictedBind
		restrictedBindVal = accessCodeObj.Spec.RestrictedBindValue
	}

	if id == "" {
		util.ReturnHTTPMessage(w, r, 500, "error", "no course/scenario id passed in")
		return
	}
	random := util.RandStringRunes(10)
	var course hfv1.Course
	var scenario hfv1.Scenario

	// get the course and/or scenario objects
	if courseid != "" {
		course, err = sss.courseClient.GetCourseById(courseid)
		if err != nil {
			glog.Errorf("course not found %v", err)
			util.ReturnHTTPMessage(w, r, 500, "error", "no course found")
			return
		}
	}
	if scenarioid != "" {
		scenario, err = sss.scenarioClient.GetScenarioById(scenarioid)
		if err != nil {
			glog.Errorf("scenario not found %v", err)
			util.ReturnHTTPMessage(w, r, 500, "error", "no scenario found")
			return
		}
	}

	// now we should check for existing sessions

	sessions, err := sss.hfClientSet.HobbyfarmV1().Sessions().List(metav1.ListOptions{})

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
		if v.Spec.UserId == user.Spec.Id &&
			(v.Spec.CourseId == course.Spec.Id || v.Spec.ScenarioId == scenario.Spec.Id) &&
			!v.Status.Finished &&
			v.Status.Active && expires.After(now) {
			// we should just return this session...

			// if this is a course, return the same scenario id that was given to us
			// i.e., reuse the course id and give them the scenario they asked for
			if v.Spec.CourseId != "" {
				v.Spec.ScenarioId = scenarioid

				retryErr := retry.RetryOnConflict(retry.DefaultRetry, func() error {
					result, getErr := sss.hfClientSet.HobbyfarmV1().Sessions().Get(v.Name, metav1.GetOptions{})
					if getErr != nil {
						return fmt.Errorf("error retrieving latest version of session %s: %v", v.Name, getErr)
					}

					result.Spec.ScenarioId = scenarioid

					_, updateErr := sss.hfClientSet.HobbyfarmV1().Sessions().Update(result)
					glog.V(4).Infof("updated session for new scenario")

					return updateErr
				})

				if retryErr != nil {
					glog.Errorf("error updating session %v", err)
					util.ReturnHTTPMessage(w, r, 500, "error", "something happened")
					return
				}

			}

			encodedSS, err := json.Marshal(v.Spec)
			if err != nil {
				glog.Error(err)
			}
			util.ReturnHTTPContent(w, r, 200, "exists", encodedSS)
			return
		}

	}

	sessionName := util.GenerateResourceName("ss", random, 10)
	session := hfv1.Session{}

	session.Name = sessionName
	session.Spec.Id = sessionName
	session.Spec.CourseId = course.Spec.Id
	session.Spec.ScenarioId = scenario.Spec.Id
	session.Spec.UserId = user.Spec.Id

	var vms []map[string]string
	if course.Spec.VirtualMachines != nil {
		vms = course.Spec.VirtualMachines
	} else {
		vms = scenario.Spec.VirtualMachines
	}

	session.Spec.VmClaimSet = make([]string, len(vms))
	for index, vmset := range vms {
		virtualMachineClaim := hfv1.VirtualMachineClaim{}
		vmcId := util.GenerateResourceName("vmc", util.RandStringRunes(10), 10)
		vmcLabels := make(map[string]string)
		vmcLabels[vmcSessionLabel] = session.Name // map vmc to session
		virtualMachineClaim.Labels = vmcLabels
		virtualMachineClaim.Spec.Id = vmcId
		virtualMachineClaim.Name = vmcId
		virtualMachineClaim.Spec.VirtualMachines = make(map[string]hfv1.VirtualMachineClaimVM)
		for vmName, vmTemplateName := range vmset {
			virtualMachineClaim.Spec.VirtualMachines[vmName] = hfv1.VirtualMachineClaimVM{vmTemplateName, ""}
		}
		virtualMachineClaim.Spec.UserId = user.Spec.Id
		virtualMachineClaim.Status.Bound = false
		virtualMachineClaim.Status.Ready = false
		virtualMachineClaim.Spec.DynamicCapable = true

		if restrictedBind {
			virtualMachineClaim.Spec.RestrictedBind = restrictedBind
			virtualMachineClaim.Spec.RestrictedBindValue = restrictedBindVal
		} else {
			virtualMachineClaim.Spec.RestrictedBind = false
		}

		createdVmClaim, err := sss.hfClientSet.HobbyfarmV1().VirtualMachineClaims().Create(&virtualMachineClaim)
		if err != nil {
			glog.Errorf("error creating vm claim %v", err)
			util.ReturnHTTPMessage(w, r, 500, "error", "something happened")
			return
		}
		session.Spec.VmClaimSet[index] = createdVmClaim.Spec.Id
	}

	var ssTimeout string

	if course.Spec.KeepAliveDuration != "" {
		ssTimeout = course.Spec.KeepAliveDuration
	} else if scenario.Spec.KeepAliveDuration != "" {
		ssTimeout = scenario.Spec.KeepAliveDuration
	} else {
		ssTimeout = newSSTimeout
	}

	session.Status.StartTime = now.Format(time.UnixDate)
	duration, _ := time.ParseDuration(ssTimeout)

	session.Status.ExpirationTime = now.Add(duration).Format(time.UnixDate)
	session.Status.Active = true
	session.Status.Finished = false

	createdSession, err := sss.hfClientSet.HobbyfarmV1().Sessions().Create(&session)

	if err != nil {
		glog.Errorf("error creating session %v", err)
		util.ReturnHTTPMessage(w, r, 500, "error", "something happened")
		return
	}

	glog.V(2).Infof("created session ID %s", createdSession.Spec.Id)
	encodedSS, err := json.Marshal(createdSession.Spec)
	if err != nil {
		glog.Error(err)
	}
	util.ReturnHTTPContent(w, r, 201, "created", encodedSS)
	return
}

func (sss SessionServer) FinishedSessionFunc(w http.ResponseWriter, r *http.Request) {
	user, err := sss.auth.AuthN(w, r)
	if err != nil {
		util.ReturnHTTPMessage(w, r, 403, "forbidden", "no access to create sessions")
		return
	}
	vars := mux.Vars(r)

	sessionId := vars["session_id"]
	if len(sessionId) == 0 {
		util.ReturnHTTPMessage(w, r, 500, "error", "no session id passed in")
		return
	}

	ss, err := sss.GetSessionById(sessionId)
	if ss.Spec.UserId != user.Spec.Id {
		util.ReturnHTTPMessage(w, r, 403, "forbidden", "no session found that matches this user")
		return
	}

	now := time.Now().Format(time.UnixDate)

	retryErr := retry.RetryOnConflict(retry.DefaultRetry, func() error {
		result, getErr := sss.hfClientSet.HobbyfarmV1().Sessions().Get(sessionId, metav1.GetOptions{})
		if getErr != nil {
			return fmt.Errorf("error retrieving latest version of session %s: %v", sessionId, getErr)
		}

		result.Status.ExpirationTime = now
		result.Status.Active = false
		result.Status.Finished = false

		_, updateErr := sss.hfClientSet.HobbyfarmV1().Sessions().Update(result)
		glog.V(4).Infof("updated result for environment")

		return updateErr
	})

	if retryErr != nil {
		glog.Errorf("error creating session %v", err)
		util.ReturnHTTPMessage(w, r, 500, "error", "something happened")
		return
	}

	util.ReturnHTTPMessage(w, r, 200, "updated", "updated session")
	return
}

func (sss SessionServer) KeepAliveSessionFunc(w http.ResponseWriter, r *http.Request) {
	user, err := sss.auth.AuthN(w, r)
	if err != nil {
		util.ReturnHTTPMessage(w, r, 403, "forbidden", "no access to create sessions")
		return
	}
	vars := mux.Vars(r)

	sessionId := vars["session_id"]
	if len(sessionId) == 0 {
		util.ReturnHTTPMessage(w, r, 500, "error", "no session id passed in")
		return
	}

	ss, err := sss.GetSessionById(sessionId)
	if ss.Spec.UserId != user.Spec.Id {
		util.ReturnHTTPMessage(w, r, 403, "forbidden", "no session found that matches this user")
		return
	}

	if ss.Status.Finished {
		util.ReturnHTTPMessage(w, r, 404, "notfound", "session was finished")
		return
	}

	if ss.Status.Paused {
		glog.V(4).Infof("session %s was paused, returning paused", ss.Spec.Id)

		now := time.Now()
		pauseExpiration, err := time.Parse(time.UnixDate, ss.Status.PausedTime)

		if err != nil {
			glog.Error(err)
			util.ReturnHTTPMessage(w, r, 304, "paused", "session is paused")
			return
		}

		timeUntilExpiration := pauseExpiration.Sub(now)

		util.ReturnHTTPMessage(w, r, 202, "paused", timeUntilExpiration.String())
		return
	}

	var scenario hfv1.Scenario
	var course hfv1.Course

	if ss.Spec.ScenarioId != "" {
		scenario, err = sss.scenarioClient.GetScenarioById(ss.Spec.ScenarioId)
		if err != nil {
			glog.Errorf("error retrieving scenario %v", err)
			util.ReturnHTTPMessage(w, r, 500, "error", "error getting scenario")
			return
		}
	}
	if ss.Spec.CourseId != "" {
		course, err = sss.courseClient.GetCourseById(ss.Spec.CourseId)
		if err != nil {
			glog.Errorf("error retrieving course %v", err)
			util.ReturnHTTPMessage(w, r, 500, "error", "error getting course")
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
		result, getErr := sss.hfClientSet.HobbyfarmV1().Sessions().Get(sessionId, metav1.GetOptions{})
		if getErr != nil {
			return fmt.Errorf("error retrieving latest version of session %s: %v", sessionId, getErr)
		}

		result.Status.ExpirationTime = expiration

		_, updateErr := sss.hfClientSet.HobbyfarmV1().Sessions().Update(result)
		glog.V(4).Infof("updated expiration time for session")

		return updateErr
	})

	if retryErr != nil {
		glog.Errorf("error creating session %v", err)
		util.ReturnHTTPMessage(w, r, 500, "error", "something happened")
		return
	}

	util.ReturnHTTPMessage(w, r, 202, "keepalived", "keepalive successful")
	return
}

func (sss SessionServer) PauseSessionFunc(w http.ResponseWriter, r *http.Request) {
	user, err := sss.auth.AuthN(w, r)
	if err != nil {
		util.ReturnHTTPMessage(w, r, 403, "forbidden", "no access to pause sessions")
		return
	}
	vars := mux.Vars(r)

	sessionId := vars["session_id"]
	if len(sessionId) == 0 {
		util.ReturnHTTPMessage(w, r, 500, "error", "no session id passed in")
		return
	}

	ss, err := sss.GetSessionById(sessionId)
	if ss.Spec.UserId != user.Spec.Id {
		util.ReturnHTTPMessage(w, r, 403, "forbidden", "no session found that matches this user")
		return
	}

	var course hfv1.Course
	var scenario hfv1.Scenario

	if ss.Spec.CourseId != "" {
		course, err = sss.courseClient.GetCourseById(ss.Spec.CourseId)
		if err != nil {
			glog.Errorf("error retrieving course %v", err)
			util.ReturnHTTPMessage(w, r, 500, "error", "error getting course")
			return
		}
	}
	if ss.Spec.ScenarioId != "" {
		scenario, err = sss.scenarioClient.GetScenarioById(ss.Spec.ScenarioId)
		if err != nil {
			glog.Errorf("error retrieving scenario %v", err)
			util.ReturnHTTPMessage(w, r, 500, "error", "error getting scenario")
			return
		}
	}

	if !course.Spec.Pauseable && !scenario.Spec.Pauseable {
		glog.Errorf("session is not pauseable %s", course.Spec.Id)
		util.ReturnHTTPMessage(w, r, 500, "error", "not pauseable")
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
		result, getErr := sss.hfClientSet.HobbyfarmV1().Sessions().Get(sessionId, metav1.GetOptions{})
		if getErr != nil {
			return fmt.Errorf("error retrieving latest version of session %s: %v", sessionId, getErr)
		}

		result.Status.PausedTime = pauseExpiration
		result.Status.Paused = true

		_, updateErr := sss.hfClientSet.HobbyfarmV1().Sessions().Update(result)
		glog.V(4).Infof("updated result for course session")

		return updateErr
	})

	if retryErr != nil {
		glog.Errorf("error creating session %v", err)
		util.ReturnHTTPMessage(w, r, 500, "error", "something happened")
		return
	}

	util.ReturnHTTPMessage(w, r, 204, "updated", "updated session")
	return
}

func (sss SessionServer) ResumeSessionFunc(w http.ResponseWriter, r *http.Request) {
	user, err := sss.auth.AuthN(w, r)
	if err != nil {
		util.ReturnHTTPMessage(w, r, 403, "forbidden", "no access to pause sessions")
		return
	}
	vars := mux.Vars(r)

	sessionId := vars["session_id"]
	if len(sessionId) == 0 {
		util.ReturnHTTPMessage(w, r, 500, "error", "no session id passed in")
		return
	}

	ss, err := sss.GetSessionById(sessionId)
	if ss.Spec.UserId != user.Spec.Id {
		util.ReturnHTTPMessage(w, r, 403, "forbidden", "no session found that matches this user")
		return
	}

	var course hfv1.Course
	var scenario hfv1.Scenario

	if ss.Spec.CourseId != "" {
		course, err = sss.courseClient.GetCourseById(ss.Spec.CourseId)
		if err != nil {
			glog.Errorf("error retrieving course %v", err)
			util.ReturnHTTPMessage(w, r, 500, "error", "error getting course")
			return
		}
	}
	if ss.Spec.ScenarioId != "" {
		scenario, err = sss.scenarioClient.GetScenarioById(ss.Spec.ScenarioId)
		if err != nil {
			glog.Errorf("error retrieving scenario %v", err)
			util.ReturnHTTPMessage(w, r, 500, "error", "error getting scenario")
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
		result, getErr := sss.hfClientSet.HobbyfarmV1().Sessions().Get(sessionId, metav1.GetOptions{})
		if getErr != nil {
			return fmt.Errorf("error retrieving latest version of session %s: %v", sessionId, getErr)
		}

		result.Status.PausedTime = ""
		result.Status.ExpirationTime = newExpiration
		result.Status.Paused = false

		_, updateErr := sss.hfClientSet.HobbyfarmV1().Sessions().Update(result)
		glog.V(4).Infof("updated result for session")

		return updateErr
	})

	if retryErr != nil {
		glog.Errorf("error creating session %v", err)
		util.ReturnHTTPMessage(w, r, 500, "error", "something happened")
		return
	}

	util.ReturnHTTPMessage(w, r, 204, "updated", "resumed session")
	return
}

func (sss SessionServer) GetSessionFunc(w http.ResponseWriter, r *http.Request) {
	user, err := sss.auth.AuthN(w, r)
	if err != nil {
		util.ReturnHTTPMessage(w, r, 403, "forbidden", "no access to get sessions")
		return
	}

	vars := mux.Vars(r)

	sessionId := vars["session_id"]
	if len(sessionId) == 0 {
		util.ReturnHTTPMessage(w, r, 500, "error", "no session id passed in")
		return
	}

	ss, err := sss.GetSessionById(sessionId)
	if ss.Spec.UserId != user.Spec.Id {
		util.ReturnHTTPMessage(w, r, 403, "forbidden", "no session found that matches this user")
		return
	}

	encodedSS, err := json.Marshal(ss.Spec)
	if err != nil {
		glog.Error(err)
	}
	util.ReturnHTTPContent(w, r, 200, "success", encodedSS)

	glog.V(2).Infof("retrieved session %s", ss.Spec.Id)
}

func ssIdIndexer(obj interface{}) ([]string, error) {
	ss, ok := obj.(*hfv1.Session)
	if !ok {
		return []string{}, nil
	}
	return []string{ss.Spec.Id}, nil
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
