package vmserver

import (
"encoding/json"
	"fmt"
	"github.com/golang/glog"
"github.com/gorilla/mux"
hfv1 "github.com/hobbyfarm/gargantua/pkg/apis/hobbyfarm.io/v1"
"github.com/hobbyfarm/gargantua/pkg/authclient"
hfClientset "github.com/hobbyfarm/gargantua/pkg/client/clientset/versioned"
hfInformers "github.com/hobbyfarm/gargantua/pkg/client/informers/externalversions"
"github.com/hobbyfarm/gargantua/pkg/scenariosessionclient"
"github.com/hobbyfarm/gargantua/pkg/util"
	"k8s.io/client-go/tools/cache"
	"net/http"
)

const (
	idIndex = "vms.hobbyfarm.io/id-index"
)

type VMServer struct {

	auth *authclient.AuthClient
	hfClientSet *hfClientset.Clientset
	ssClient *scenariosessionclient.ScenarioSessionClient

	vmIndexer cache.Indexer

}

func NewVMServer(authClient *authclient.AuthClient, scenarioSessionClient *scenariosessionclient.ScenarioSessionClient, hfClientset *hfClientset.Clientset, hfInformerFactory hfInformers.SharedInformerFactory) (*VMServer, error) {
	vms := VMServer{}

	vms.ssClient = scenarioSessionClient
	vms.hfClientSet = hfClientset
	vms.auth = authClient

	inf := hfInformerFactory.Hobbyfarm().V1().VirtualMachines().Informer()
	indexers := map[string]cache.IndexFunc{idIndex: vmIdIndexer}
	inf.AddIndexers(indexers)
	vms.vmIndexer = inf.GetIndexer()

	return &vms, nil
}

func (vms VMServer) GetVirtualMachineById(id string) (hfv1.VirtualMachine, error) {

	empty := hfv1.VirtualMachine{}

	if len(id) == 0 {
		return empty, fmt.Errorf("vm id passed in was empty")
	}

	obj, err := vms.vmIndexer.ByIndex(idIndex, id)
	if err != nil {
		return empty, fmt.Errorf("error while retrieving virtualmachine by id: %s with error: %v", id, err)
	}

	if len(obj) < 1 {
		return empty, fmt.Errorf("virtualmachine not found by id: %s", id)
	}

	result, ok := obj[0].(*hfv1.VirtualMachine)

	if !ok {
		return empty, fmt.Errorf("error while converting virtualmachine found by id to object: %s", id)
	}

	return *result, nil

}

func (vms VMServer) SetupRoutes(r *mux.Router) {
	r.HandleFunc("/vm/{scenario_session_id}/{vm_id}", vms.GetVMFunc)
	glog.V(2).Infof("set up routes")
}

type PreparedVirtualMachine struct {
	hfv1.VirtualMachineSpec
	hfv1.VirtualMachineStatus
}

func (vms VMServer) GetVMFunc(w http.ResponseWriter, r *http.Request) {
	user, err := vms.auth.AuthN(w, r)
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

	scenarioSession, err := vms.ssClient.GetScenarioSessionById(scenarioSessionId)

	if err != nil {
		glog.Errorf("did not find the scenario session corresponding to %s", scenarioSessionId)
		util.ReturnHTTPMessage(w, r, 500, "error", "no session found")
		return
	}

	if scenarioSession.Spec.UserId != user.Spec.Id {
		util.ReturnHTTPMessage(w, r, 403, "forbidden", "you do not have access to scenario session")
		return
	}

	vm, err := vms.GetVirtualMachineById(vmId)

	if err != nil {
		glog.Errorf("did not find the right virtual machine ID")
		util.ReturnHTTPMessage(w, r, 500, "error", "no vm found")
		return
	}

	preparedVM := PreparedVirtualMachine{vm.Spec, vm.Status}

	encodedVM, err := json.Marshal(preparedVM)
	if err != nil {
		glog.Error(err)
	}
	util.ReturnHTTPContent(w, r, 200, "success", encodedVM)

	glog.V(2).Infof("retrieved vm %s", vm.Spec.Id)
}

func vmIdIndexer(obj interface{}) ([]string, error) {
	vm, ok := obj.(*hfv1.VirtualMachine)
	if !ok {
		return []string{}, nil
	}
	return []string{vm.Spec.Id}, nil
}