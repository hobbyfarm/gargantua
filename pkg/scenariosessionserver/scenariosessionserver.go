package scenariosessionserver

import (
	"encoding/json"
	"fmt"
	"github.com/golang/glog"
	"github.com/gorilla/mux"
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
	ssIndex = "sss.hobbyfarm.io/scenariosession-id-index"
)

type ScenarioSessionServer struct {
	hfClientSet    *hfClientset.Clientset
	scenarioClient *scenarioclient.ScenarioClient
	auth           *authclient.AuthClient
	ssIndexer      cache.Indexer
}

func NewScenarioSessionServer(authClient *authclient.AuthClient, scenarioClient *scenarioclient.ScenarioClient, hfClientSet *hfClientset.Clientset, hfInformerFactory hfInformers.SharedInformerFactory) (*ScenarioSessionServer, error) {
	a := ScenarioSessionServer{}
	a.hfClientSet = hfClientSet
	a.scenarioClient = scenarioClient
	a.auth = authClient
	inf := hfInformerFactory.Hobbyfarm().V1().ScenarioSessions().Informer()
	indexers := map[string]cache.IndexFunc{ssIndex: ssIdIndexer}
	inf.AddIndexers(indexers)
	a.ssIndexer = inf.GetIndexer()
	return &a, nil
}

func (sss ScenarioSessionServer) SetupRoutes(r *mux.Router) {
	r.HandleFunc("/session/new", sss.NewScenarioSessionFunc).Methods("POST")
	r.HandleFunc("/session/{scenario_session_id}", sss.GetScenarioSessionFunc).Methods("GET")
	r.HandleFunc("/session/{scenario_session_id}/finished", sss.FinishedScenarioSessionFunc).Methods("PUT")
	r.HandleFunc("/session/{scenario_session_id}/keepalive", sss.KeepAliveScenarioSessionFunc).Methods("PUT")
	glog.V(2).Infof("set up routes")
}

func (sss ScenarioSessionServer) NewScenarioSessionFunc(w http.ResponseWriter, r *http.Request) {
	user, err := sss.auth.AuthN(w, r)
	if err != nil {
		util.ReturnHTTPMessage(w, r, 403, "forbidden", "no access to create scenario sessions")
		return
	}

	scenarioId := r.PostFormValue("scenario")

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

	scenarioSessionName := util.GenerateResourceName("ss", random, 10)
	scenarioSession := hfv1.ScenarioSession{}

	scenarioSession.Name = scenarioSessionName
	scenarioSession.Spec.Id = scenarioSessionName
	scenarioSession.Spec.ScenarioId = scenario.Spec.Id
	scenarioSession.Spec.UserId = user.Spec.Id

	scenarioSession.Spec.VmClaimSet = make([]string, len(scenario.Spec.VirtualMachines))
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
		virtualMachineClaim.Spec.VirtualMachineClassId = "default"
		virtualMachineClaim.Status.Bound = false
		virtualMachineClaim.Status.Ready = false

		createdVmClaim, err := sss.hfClientSet.HobbyfarmV1().VirtualMachineClaims().Create(&virtualMachineClaim)
		if err != nil {
			glog.Errorf("error creating vm claim %v", err)
			util.ReturnHTTPMessage(w, r, 500, "error", "something happened")
			return
		}
		scenarioSession.Spec.VmClaimSet[index] = createdVmClaim.Spec.Id
	}

	now := time.Now()
	scenarioSession.Status.StartTime = now.Format(time.UnixDate)
	duration, _ := time.ParseDuration("5m")

	scenarioSession.Status.ExpirationTime = now.Add(duration).Format(time.UnixDate)
	scenarioSession.Status.Active = true

	createdScenarioSession, err := sss.hfClientSet.HobbyfarmV1().ScenarioSessions().Create(&scenarioSession)

	if err != nil {
		glog.Errorf("error creating scenario session %v", err)
		util.ReturnHTTPMessage(w, r, 500, "error", "something happened")
		return
	}

	glog.V(2).Infof("created scenario session ID %s", createdScenarioSession.Spec.Id)
	encodedSS, err := json.Marshal(createdScenarioSession.Spec)
	if err != nil {
		glog.Error(err)
	}
	util.ReturnHTTPContent(w, r, 201, "created", encodedSS)
	return
}

func (sss ScenarioSessionServer) FinishedScenarioSessionFunc(w http.ResponseWriter, r *http.Request) {
	user, err := sss.auth.AuthN(w, r)
	if err != nil {
		util.ReturnHTTPMessage(w, r, 403, "forbidden", "no access to create scenario sessions")
		return
	}
	vars := mux.Vars(r)

	scenarioSessionId := vars["scenario_session_id"]
	if len(scenarioSessionId) == 0 {
		util.ReturnHTTPMessage(w, r, 500, "error", "no scenario session id passed in")
		return
	}

	ss, err := sss.GetScenarioSessionById(scenarioSessionId)
	if ss.Spec.UserId != user.Spec.Id {
		util.ReturnHTTPMessage(w, r, 403, "forbidden", "no scenario session found that matches this user")
		return
	}

	now := time.Now().Format(time.UnixDate)

	retryErr := retry.RetryOnConflict(retry.DefaultRetry, func() error {
		result, getErr := sss.hfClientSet.HobbyfarmV1().ScenarioSessions().Get(scenarioSessionId, metav1.GetOptions{})
		if getErr != nil {
			return fmt.Errorf("error retrieving latest version of Scenario Session %s: %v", scenarioSessionId, getErr)
		}

		result.Status.ExpirationTime = now
		result.Status.Active = false
		result.Status.Finished = true

		_, updateErr := sss.hfClientSet.HobbyfarmV1().ScenarioSessions().Update(result)
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

func (sss ScenarioSessionServer) KeepAliveScenarioSessionFunc(w http.ResponseWriter, r *http.Request) {
	user, err := sss.auth.AuthN(w, r)
	if err != nil {
		util.ReturnHTTPMessage(w, r, 403, "forbidden", "no access to create scenario sessions")
		return
	}
	vars := mux.Vars(r)

	scenarioSessionId := vars["scenario_session_id"]
	if len(scenarioSessionId) == 0 {
		util.ReturnHTTPMessage(w, r, 500, "error", "no scenario session id passed in")
		return
	}

	ss, err := sss.GetScenarioSessionById(scenarioSessionId)
	if ss.Spec.UserId != user.Spec.Id {
		util.ReturnHTTPMessage(w, r, 403, "forbidden", "no scenario session found that matches this user")
		return
	}

	now := time.Now()
	duration, _ := time.ParseDuration("5m")

	expiration := now.Add(duration).Format(time.UnixDate)

	retryErr := retry.RetryOnConflict(retry.DefaultRetry, func() error {
		result, getErr := sss.hfClientSet.HobbyfarmV1().ScenarioSessions().Get(scenarioSessionId, metav1.GetOptions{})
		if getErr != nil {
			return fmt.Errorf("error retrieving latest version of Scenario Session %s: %v", scenarioSessionId, getErr)
		}

		result.Status.ExpirationTime = expiration

		_, updateErr := sss.hfClientSet.HobbyfarmV1().ScenarioSessions().Update(result)
		glog.V(4).Infof("updated result for environment")

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

func (sss ScenarioSessionServer) GetScenarioSessionFunc(w http.ResponseWriter, r *http.Request) {
	user, err := sss.auth.AuthN(w, r)
	if err != nil {
		util.ReturnHTTPMessage(w, r, 403, "forbidden", "no access to get scenario sessions")
		return
	}

	vars := mux.Vars(r)

	scenarioSessionId := vars["scenario_session_id"]
	if len(scenarioSessionId) == 0 {
		util.ReturnHTTPMessage(w, r, 500, "error", "no scenario session id passed in")
		return
	}

	ss, err := sss.GetScenarioSessionById(scenarioSessionId)
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
	ss, ok := obj.(*hfv1.ScenarioSession)
	if !ok {
		return []string{}, nil
	}
	return []string{ss.Spec.Id}, nil
}

func (sss ScenarioSessionServer) GetScenarioSessionById(id string) (hfv1.ScenarioSession, error) {
	if len(id) == 0 {
		return hfv1.ScenarioSession{}, fmt.Errorf("id passed in was empty")
	}

	obj, err := sss.ssIndexer.ByIndex(ssIndex, id)
	if err != nil {
		return hfv1.ScenarioSession{}, fmt.Errorf("error while retrieving scenario session by id: %s with error: %v", id, err)
	}

	if len(obj) < 1 {
		return hfv1.ScenarioSession{}, fmt.Errorf("sss not found by id: %s", id)
	}

	scenarioSession, ok := obj[0].(*hfv1.ScenarioSession)

	if !ok {
		return hfv1.ScenarioSession{}, fmt.Errorf("error while converting secnario session found by id to object: %s", id)
	}

	return *scenarioSession, nil

}
