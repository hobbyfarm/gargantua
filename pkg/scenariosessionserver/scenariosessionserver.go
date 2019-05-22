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
	"github.com/hobbyfarm/gargantua/pkg/util"
	"k8s.io/client-go/tools/cache"
	"net/http"
)

const (
	ssIndex = "sss.hobbyfarm.io/scenariosession-id-index"
)

type ScenarioSessionServer struct {
	hfClientSet *hfClientset.Clientset
	auth *authclient.AuthClient
	ssIndexer cache.Indexer
}

func NewScenarioSessionServer(authClient *authclient.AuthClient, hfClientSet *hfClientset.Clientset, hfInformerFactory hfInformers.SharedInformerFactory) (*ScenarioSessionServer, error) {
	a := ScenarioSessionServer{}
	a.hfClientSet = hfClientSet
	a.auth = authClient
	inf := hfInformerFactory.Hobbyfarm().V1().ScenarioSessions().Informer()
	indexers := map[string]cache.IndexFunc{ssIndex: ssIdIndexer}
	inf.AddIndexers(indexers)
	a.ssIndexer = inf.GetIndexer()
	return &a, nil
}

func (sss ScenarioSessionServer) SetupRoutes(r *mux.Router) {
	r.HandleFunc("/session/new", sss.NewScenarioSessionFunc)
	r.HandleFunc("/session/{scenario_session_id}", sss.GetScenarioSessionFunc)
	glog.V(2).Infof("set up routes")
}

func (sss ScenarioSessionServer) NewScenarioSessionFunc(w http.ResponseWriter, r *http.Request) {

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