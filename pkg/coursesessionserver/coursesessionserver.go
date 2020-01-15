package coursesessionserver

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
	"github.com/hobbyfarm/gargantua/pkg/scenarioclient"
	"github.com/hobbyfarm/gargantua/pkg/util"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/retry"
	"net/http"
	"time"
)

const (
	ssIndex            = "sss.hobbyfarm.io/coursesession-id-index"
	newSSTimeout       = "5m"
	keepaliveSSTimeout = "5m"
	pauseSSTimeout     = "2h"
)

type CourseSessionServer struct {
	hfClientSet      *hfClientset.Clientset
	scenarioClient   *scenarioclient.ScenarioClient
	accessCodeClient *accesscode.AccessCodeClient
	auth             *authclient.AuthClient
	ssIndexer        cache.Indexer
}

func NewCourseSessionServer(authClient *authclient.AuthClient, accessCodeClient *accesscode.AccessCodeClient, scenarioClient *scenarioclient.ScenarioClient, hfClientSet *hfClientset.Clientset, hfInformerFactory hfInformers.SharedInformerFactory) (*CourseSessionServer, error) {
	a := CourseSessionServer{}
	a.hfClientSet = hfClientSet
	a.scenarioClient = scenarioClient
	a.auth = authClient
	a.accessCodeClient = accessCodeClient
	inf := hfInformerFactory.Hobbyfarm().V1().CourseSessions().Informer()
	indexers := map[string]cache.IndexFunc{ssIndex: ssIdIndexer}
	inf.AddIndexers(indexers)
	a.ssIndexer = inf.GetIndexer()
	return &a, nil
}

func (sss CourseSessionServer) SetupRoutes(r *mux.Router) {
	r.HandleFunc("/session/new", sss.NewCourseSessionFunc).Methods("POST")
	r.HandleFunc("/session/{scenario_session_id}", sss.GetCourseSessionFunc).Methods("GET")
	r.HandleFunc("/session/{scenario_session_id}/finished", sss.FinishedCourseSessionFunc).Methods("PUT")
	r.HandleFunc("/session/{scenario_session_id}/keepalive", sss.KeepAliveCourseSessionFunc).Methods("PUT")
	r.HandleFunc("/session/{scenario_session_id}/pause", sss.PauseCourseSessionFunc).Methods("PUT")
	r.HandleFunc("/session/{scenario_session_id}/resume", sss.ResumeCourseSessionFunc).Methods("PUT")
	glog.V(2).Infof("set up routes for scenario session server")
}

func (sss CourseSessionServer) NewCourseSessionFunc(w http.ResponseWriter, r *http.Request) {
	user, err := sss.auth.AuthN(w, r)
	if err != nil {
		util.ReturnHTTPMessage(w, r, 403, "forbidden", "no access to create scenario sessions")
		return
	}

	scenarioId := r.PostFormValue("scenario")

	accessCode := r.PostFormValue("access_code")
	glog.V(4).Infof("access code passed in was %s", accessCode)

	restrictedBind := false
	restrictedBindVal := ""

	if accessCode == "" {
		accessCode, err = sss.accessCodeClient.GetClosestAccessCodeForScenario(user.Spec.Id, scenarioId)
		if err != nil {
			glog.Error(err)
			util.ReturnHTTPMessage(w, r, 500, "error", "error retrieving access code applicable to scenario")
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

	if scenarioId == "" {
		util.ReturnHTTPMessage(w, r, 500, "error", "no scenario id passed in")
		return
	}
	random := util.RandStringRunes(10)

	scenario, err := sss.scenarioClient.GetScenarioById(scenarioId)
	if err != nil {
		glog.Errorf("scenario not found %v", err)
		util.ReturnHTTPMessage(w, r, 500, "error", "no scenario found")
		return
	}

	// now we should check for existing scenario sessions

	courseSessions, err := sss.hfClientSet.HobbyfarmV1().CourseSessions().List(metav1.ListOptions{})

	if err != nil {
		glog.Error(err)
	}
	now := time.Now()

	// should we check the scenario sessions list for the restricted bind value and match if one is passed in? probably...
	for _, v := range courseSessions.Items {
		expires, err := time.Parse(time.UnixDate, v.Status.ExpirationTime)
		if err != nil {
			continue
		}
		if v.Spec.UserId == user.Spec.Id &&
			v.Spec.ScenarioId == scenario.Spec.Id &&
			!v.Status.Finished &&
			v.Status.Active && expires.After(now) {
			// we should just return this scenario session...
			encodedSS, err := json.Marshal(v.Spec)
			if err != nil {
				glog.Error(err)
			}
			util.ReturnHTTPContent(w, r, 200, "exists", encodedSS)
			return
		}

	}

	courseSessionName := util.GenerateResourceName("ss", random, 10)
	courseSession := hfv1.CourseSession{}

	courseSession.Name = courseSessionName
	courseSession.Spec.Id = courseSessionName
	courseSession.Spec.ScenarioId = scenario.Spec.Id
	courseSession.Spec.UserId = user.Spec.Id

	courseSession.Spec.VmClaimSet = make([]string, len(scenario.Spec.VirtualMachines))
	for index, vmset := range scenario.Spec.VirtualMachines {
		virtualMachineClaim := hfv1.VirtualMachineClaim{}
		vmcId := util.GenerateResourceName("vmc", util.RandStringRunes(10), 10)
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
		courseSession.Spec.VmClaimSet[index] = createdVmClaim.Spec.Id
	}

	var ssTimeout string

	if scenario.Spec.KeepAliveDuration != "" {
		ssTimeout = scenario.Spec.KeepAliveDuration
	} else {
		ssTimeout = newSSTimeout
	}

	courseSession.Status.StartTime = now.Format(time.UnixDate)
	duration, _ := time.ParseDuration(ssTimeout)

	courseSession.Status.ExpirationTime = now.Add(duration).Format(time.UnixDate)
	courseSession.Status.Active = true
	courseSession.Status.Finished = false

	createdCourseSession, err := sss.hfClientSet.HobbyfarmV1().CourseSessions().Create(&courseSession)

	if err != nil {
		glog.Errorf("error creating scenario session %v", err)
		util.ReturnHTTPMessage(w, r, 500, "error", "something happened")
		return
	}

	glog.V(2).Infof("created scenario session ID %s", createdCourseSession.Spec.Id)
	encodedSS, err := json.Marshal(createdCourseSession.Spec)
	if err != nil {
		glog.Error(err)
	}
	util.ReturnHTTPContent(w, r, 201, "created", encodedSS)
	return
}

func (sss CourseSessionServer) FinishedCourseSessionFunc(w http.ResponseWriter, r *http.Request) {
	user, err := sss.auth.AuthN(w, r)
	if err != nil {
		util.ReturnHTTPMessage(w, r, 403, "forbidden", "no access to create scenario sessions")
		return
	}
	vars := mux.Vars(r)

	courseSessionId := vars["scenario_session_id"]
	if len(courseSessionId) == 0 {
		util.ReturnHTTPMessage(w, r, 500, "error", "no scenario session id passed in")
		return
	}

	ss, err := sss.GetCourseSessionById(courseSessionId)
	if ss.Spec.UserId != user.Spec.Id {
		util.ReturnHTTPMessage(w, r, 403, "forbidden", "no scenario session found that matches this user")
		return
	}

	now := time.Now().Format(time.UnixDate)

	retryErr := retry.RetryOnConflict(retry.DefaultRetry, func() error {
		result, getErr := sss.hfClientSet.HobbyfarmV1().CourseSessions().Get(courseSessionId, metav1.GetOptions{})
		if getErr != nil {
			return fmt.Errorf("error retrieving latest version of Scenario Session %s: %v", courseSessionId, getErr)
		}

		result.Status.ExpirationTime = now
		result.Status.Active = false
		result.Status.Finished = false

		_, updateErr := sss.hfClientSet.HobbyfarmV1().CourseSessions().Update(result)
		glog.V(4).Infof("updated result for environment")

		return updateErr
	})

	if retryErr != nil {
		glog.Errorf("error creating scenario session %v", err)
		util.ReturnHTTPMessage(w, r, 500, "error", "something happened")
		return
	}

	util.ReturnHTTPMessage(w, r, 200, "updated", "updated scenario session")
	return
}

func (sss CourseSessionServer) KeepAliveCourseSessionFunc(w http.ResponseWriter, r *http.Request) {
	user, err := sss.auth.AuthN(w, r)
	if err != nil {
		util.ReturnHTTPMessage(w, r, 403, "forbidden", "no access to create scenario sessions")
		return
	}
	vars := mux.Vars(r)

	courseSessionId := vars["scenario_session_id"]
	if len(courseSessionId) == 0 {
		util.ReturnHTTPMessage(w, r, 500, "error", "no scenario session id passed in")
		return
	}

	ss, err := sss.GetCourseSessionById(courseSessionId)
	if ss.Spec.UserId != user.Spec.Id {
		util.ReturnHTTPMessage(w, r, 403, "forbidden", "no scenario session found that matches this user")
		return
	}

	if ss.Status.Finished {
		util.ReturnHTTPMessage(w, r, 404, "notfound", "scenario session was finished")
		return
	}

	if ss.Status.Paused {
		glog.V(4).Infof("Scenario session %s was paused, returning paused", ss.Spec.Id)

		now := time.Now()
		pauseExpiration, err := time.Parse(time.UnixDate, ss.Status.PausedTime)

		if err != nil {
			glog.Error(err)
			util.ReturnHTTPMessage(w, r, 304, "paused", "scenario session is paused")
			return
		}

		timeUntilExpiration := pauseExpiration.Sub(now)

		util.ReturnHTTPMessage(w, r, 202, "paused", timeUntilExpiration.String())
		return
	}

	scenario, err := sss.scenarioClient.GetScenarioById(ss.Spec.ScenarioId)

	if err != nil {
		glog.Errorf("error retrieving scenario %v", err)
		util.ReturnHTTPMessage(w, r, 500, "error", "error getting scenario")
		return
	}

	var ssTimeout string

	if scenario.Spec.KeepAliveDuration != "" {
		ssTimeout = scenario.Spec.KeepAliveDuration
	} else {
		ssTimeout = newSSTimeout
	}

	now := time.Now()
	duration, _ := time.ParseDuration(ssTimeout)

	expiration := now.Add(duration).Format(time.UnixDate)

	retryErr := retry.RetryOnConflict(retry.DefaultRetry, func() error {
		result, getErr := sss.hfClientSet.HobbyfarmV1().CourseSessions().Get(courseSessionId, metav1.GetOptions{})
		if getErr != nil {
			return fmt.Errorf("error retrieving latest version of Scenario Session %s: %v", courseSessionId, getErr)
		}

		result.Status.ExpirationTime = expiration

		_, updateErr := sss.hfClientSet.HobbyfarmV1().CourseSessions().Update(result)
		glog.V(4).Infof("updated expiration time for scenario session")

		return updateErr
	})

	if retryErr != nil {
		glog.Errorf("error creating scenario session %v", err)
		util.ReturnHTTPMessage(w, r, 500, "error", "something happened")
		return
	}

	util.ReturnHTTPMessage(w, r, 202, "keepalived", "keepalive successful")
	return
}

func (sss CourseSessionServer) PauseCourseSessionFunc(w http.ResponseWriter, r *http.Request) {
	user, err := sss.auth.AuthN(w, r)
	if err != nil {
		util.ReturnHTTPMessage(w, r, 403, "forbidden", "no access to pause scenario sessions")
		return
	}
	vars := mux.Vars(r)

	courseSessionId := vars["scenario_session_id"]
	if len(courseSessionId) == 0 {
		util.ReturnHTTPMessage(w, r, 500, "error", "no scenario session id passed in")
		return
	}

	ss, err := sss.GetCourseSessionById(courseSessionId)
	if ss.Spec.UserId != user.Spec.Id {
		util.ReturnHTTPMessage(w, r, 403, "forbidden", "no scenario session found that matches this user")
		return
	}

	scenario, err := sss.scenarioClient.GetScenarioById(ss.Spec.ScenarioId)

	if err != nil {
		glog.Errorf("error retrieving scenario %v", err)
		util.ReturnHTTPMessage(w, r, 500, "error", "error getting scenario")
		return
	}

	if !scenario.Spec.Pauseable {
		glog.Errorf("scenario is not pauseable %s", scenario.Spec.Id)
		util.ReturnHTTPMessage(w, r, 500, "error", "not pauseable")
		return
	}

	var ssTimeout string

	if scenario.Spec.PauseDuration != "" {
		ssTimeout = scenario.Spec.PauseDuration
	} else {
		ssTimeout = pauseSSTimeout
	}

	now := time.Now()
	duration, _ := time.ParseDuration(ssTimeout)

	pauseExpiration := now.Add(duration).Format(time.UnixDate)

	retryErr := retry.RetryOnConflict(retry.DefaultRetry, func() error {
		result, getErr := sss.hfClientSet.HobbyfarmV1().CourseSessions().Get(courseSessionId, metav1.GetOptions{})
		if getErr != nil {
			return fmt.Errorf("error retrieving latest version of Scenario Session %s: %v", courseSessionId, getErr)
		}

		result.Status.PausedTime = pauseExpiration
		result.Status.Paused = true

		_, updateErr := sss.hfClientSet.HobbyfarmV1().CourseSessions().Update(result)
		glog.V(4).Infof("updated result for scenario session")

		return updateErr
	})

	if retryErr != nil {
		glog.Errorf("error creating scenario session %v", err)
		util.ReturnHTTPMessage(w, r, 500, "error", "something happened")
		return
	}

	util.ReturnHTTPMessage(w, r, 204, "updated", "updated scenario session")
	return
}

func (sss CourseSessionServer) ResumeCourseSessionFunc(w http.ResponseWriter, r *http.Request) {
	user, err := sss.auth.AuthN(w, r)
	if err != nil {
		util.ReturnHTTPMessage(w, r, 403, "forbidden", "no access to pause scenario sessions")
		return
	}
	vars := mux.Vars(r)

	courseSessionId := vars["scenario_session_id"]
	if len(courseSessionId) == 0 {
		util.ReturnHTTPMessage(w, r, 500, "error", "no scenario session id passed in")
		return
	}

	ss, err := sss.GetCourseSessionById(courseSessionId)
	if ss.Spec.UserId != user.Spec.Id {
		util.ReturnHTTPMessage(w, r, 403, "forbidden", "no scenario session found that matches this user")
		return
	}

	scenario, err := sss.scenarioClient.GetScenarioById(ss.Spec.ScenarioId)

	if err != nil {
		glog.Errorf("error retrieving scenario %v", err)
		util.ReturnHTTPMessage(w, r, 500, "error", "error getting scenario")
		return
	}

	var ssTimeout string

	if scenario.Spec.KeepAliveDuration != "" {
		ssTimeout = scenario.Spec.KeepAliveDuration
	} else {
		ssTimeout = keepaliveSSTimeout
	}

	now := time.Now()
	duration, _ := time.ParseDuration(ssTimeout)

	newExpiration := now.Add(duration).Format(time.UnixDate)

	retryErr := retry.RetryOnConflict(retry.DefaultRetry, func() error {
		result, getErr := sss.hfClientSet.HobbyfarmV1().CourseSessions().Get(courseSessionId, metav1.GetOptions{})
		if getErr != nil {
			return fmt.Errorf("error retrieving latest version of Scenario Session %s: %v", courseSessionId, getErr)
		}

		result.Status.PausedTime = ""
		result.Status.ExpirationTime = newExpiration
		result.Status.Paused = false

		_, updateErr := sss.hfClientSet.HobbyfarmV1().CourseSessions().Update(result)
		glog.V(4).Infof("updated result for scenario session")

		return updateErr
	})

	if retryErr != nil {
		glog.Errorf("error creating scenario session %v", err)
		util.ReturnHTTPMessage(w, r, 500, "error", "something happened")
		return
	}

	util.ReturnHTTPMessage(w, r, 204, "updated", "resumed scenario session")
	return
}

func (sss CourseSessionServer) GetCourseSessionFunc(w http.ResponseWriter, r *http.Request) {
	user, err := sss.auth.AuthN(w, r)
	if err != nil {
		util.ReturnHTTPMessage(w, r, 403, "forbidden", "no access to get scenario sessions")
		return
	}

	vars := mux.Vars(r)

	courseSessionId := vars["scenario_session_id"]
	if len(courseSessionId) == 0 {
		util.ReturnHTTPMessage(w, r, 500, "error", "no scenario session id passed in")
		return
	}

	ss, err := sss.GetCourseSessionById(courseSessionId)
	if ss.Spec.UserId != user.Spec.Id {
		util.ReturnHTTPMessage(w, r, 403, "forbidden", "no scenario session found that matches this user")
		return
	}

	encodedSS, err := json.Marshal(ss.Spec)
	if err != nil {
		glog.Error(err)
	}
	util.ReturnHTTPContent(w, r, 200, "success", encodedSS)

	glog.V(2).Infof("retrieved SS %s", ss.Spec.Id)
}

func ssIdIndexer(obj interface{}) ([]string, error) {
	ss, ok := obj.(*hfv1.CourseSession)
	if !ok {
		return []string{}, nil
	}
	return []string{ss.Spec.Id}, nil
}

func (sss CourseSessionServer) GetCourseSessionById(id string) (hfv1.CourseSession, error) {
	if len(id) == 0 {
		return hfv1.CourseSession{}, fmt.Errorf("id passed in was empty")
	}

	obj, err := sss.ssIndexer.ByIndex(ssIndex, id)
	if err != nil {
		return hfv1.CourseSession{}, fmt.Errorf("error while retrieving scenario session by id: %s with error: %v", id, err)
	}

	if len(obj) < 1 {
		return hfv1.CourseSession{}, fmt.Errorf("sss not found by id: %s", id)
	}

	courseSession, ok := obj[0].(*hfv1.CourseSession)

	if !ok {
		return hfv1.CourseSession{}, fmt.Errorf("error while converting secnario session found by id to object: %s", id)
	}

	return *courseSession, nil

}
