package shell

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
	"github.com/hobbyfarm/gargantua/pkg/scenariosessionclient"
	"github.com/hobbyfarm/gargantua/pkg/util"
	"k8s.io/client-go/tools/cache"
	"net/http"
	"strconv"
)

const (
	idIndex = "shell.hobbyfarm.io/id-index"
	nameIndex = "shell.hobbyfarm.io/name-index"
)

type ShellProxy struct {

	auth *authclient.AuthClient
	hfClientSet *hfClientset.Clientset
	ssClient *scenariosessionclient.ScenarioSessionClient

	virtualMachineIndexer cache.Indexer

}

func NewShellProxy(authClient *authclient.AuthClient, scenarioSessionClient *scenariosessionclient.ScenarioSessionClient, hfClientset *hfClientset.Clientset, hfInformerFactory hfInformers.SharedInformerFactory) (*ShellProxy, error) {
	shellProxy := ShellProxy{}

	shellProxy.ssClient = scenarioSessionClient
	shellProxy.hfClientSet = hfClientset
	shellProxy.auth = authClient
	inf := hfInformerFactory.Hobbyfarm().V1().VirtualMachines().Informer()
	indexers := map[string]cache.IndexFunc{idIndex: vmIdIndexer, nameIndex: nameIndexer}
	inf.AddIndexers(indexers)
	shellProxy.virtualMachineIndexer = inf.GetIndexer()

	return &shellProxy, nil
}


func (sp ShellProxy) SetupRoutes(r *mux.Router) {
	r.HandleFunc("/shell/{scenario_session_id}/{vm_id}", sp.GetVMFunc)
	r.HandleFunc("/shell/{scenario_session_id}/{vm_id}/connect", sp.ConnectFunc)
	glog.V(2).Infof("set up routes")
}

func (sp ShellProxy) GetVMFunc(w http.ResponseWriter, r *http.Request) {
	_, err := sp.auth.AuthN(w, r)
	if err != nil {
		util.ReturnHTTPMessage(w, r, 403, "forbidden", "no access to get vm")
		return
	}

	vars := mux.Vars(r)

	scenarioSessionId := vars["scenario_session_id"]
	vmId := vars["vm_id"]
	if len(scenarioSessionId) == 0 {
		util.ReturnHTTPMessage(w, r, 500, "error", "no scenario session id passed in")
		return
	}

	if len(vmId) == 0 {
		util.ReturnHTTPMessage(w, r, 500, "error", "no vm id passed in")
		return
	}

	scenarioSession, err := sp.ssClient.GetScenarioSessionById(scenarioSessionId)

	if err != nil {
		glog.Errorf("did not find the scenario session corresponding to %s", scenarioSessionId)
		util.ReturnHTTPMessage(w, r, 500, "error", "no session found")
		return
	}

	scenarioSession.Spec.Vm
}

func (sp ShellProxy) ConnectFunc(w http.ResponseWriter, r *http.Request) {
	_, err := sp.auth.AuthN(w, r)
	if err != nil {
		util.ReturnHTTPMessage(w, r, 403, "forbidden", "no access to get vm")
		return
	}

	vars := mux.Vars(r)

}

func vmIdIndexer(obj interface{}) ([]string, error) {
	vm, ok := obj.(*hfv1.VirtualMachine)
	if !ok {
		return []string{}, nil
	}
	return []string{vm.Spec.Id}, nil
}

func nameIndexer(obj interface{}) ([]string, error) {
	vm, ok := obj.(*hfv1.VirtualMachine)
	if !ok {
		return []string{}, nil
	}
	return []string{vm.Status.MachineName}, nil
}