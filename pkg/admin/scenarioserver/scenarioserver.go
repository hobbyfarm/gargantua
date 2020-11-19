package scenarioserver

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
	"strings"
)

type AdminScenarioServer struct {
	auth        *authclient.AuthClient
	hfClientSet *hfClientset.Clientset
}

func NewAdminScenarioServer(authClient *authclient.AuthClient, hfClientset *hfClientset.Clientset) (*AdminScenarioServer, error) {
	s := AdminScenarioServer{}

	s.hfClientSet = hfClientset
	s.auth = authClient

	return &s, nil
}

func (a AdminScenarioServer) getScenario(id string) (hfv1.Scenario, error) {

	empty := hfv1.Scenario{}

	if len(id) == 0 {
		return empty, fmt.Errorf("scenario id passed in was empty")
	}

	obj, err := a.hfClientSet.HobbyfarmV1().Scenarios().Get(id, metav1.GetOptions{})
	if err != nil {
		return empty, fmt.Errorf("error while retrieving Scenario by id: %s with error: %v", id, err)
	}

	return *obj, nil

}

func (a AdminScenarioServer) SetupRoutes(r *mux.Router) {
	r.HandleFunc("/a/scenario/{id}", a.UpdateFunc).Methods("PUT")
	glog.V(2).Infof("set up routes for Scenario server")
}

type PreparedScenario struct {
	ID string `json:"id"`
	hfv1.ScenarioSpec
}








