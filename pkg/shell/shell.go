package shell

import (
	"github.com/golang/glog"
	"github.com/gorilla/mux"
	"github.com/hobbyfarm/gargantua/pkg/authclient"
	"github.com/hobbyfarm/gargantua/pkg/scenariosessionclient"
	"github.com/hobbyfarm/gargantua/pkg/util"
	"github.com/hobbyfarm/gargantua/pkg/vmclient"
	"net/http"
)

type ShellProxy struct {

	auth *authclient.AuthClient
	ssClient *scenariosessionclient.ScenarioSessionClient
	vmClient *vmclient.VirtualMachineClient

}

func NewShellProxy(authClient *authclient.AuthClient, vmClient *vmclient.VirtualMachineClient, scenarioSessionClient *scenariosessionclient.ScenarioSessionClient) (*ShellProxy, error) {
	shellProxy := ShellProxy{}

	shellProxy.ssClient = scenarioSessionClient
	shellProxy.auth = authClient
	shellProxy.vmClient = vmClient

	return &shellProxy, nil
}


func (sp ShellProxy) SetupRoutes(r *mux.Router) {
	r.HandleFunc("/shell/{scenario_session_id}/{vm_id}/connect", sp.ConnectFunc)
	glog.V(2).Infof("set up routes")
}

func (sp ShellProxy) ConnectFunc(w http.ResponseWriter, r *http.Request) {
	user, err := sp.auth.AuthN(w, r)
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

	if scenarioSession.Spec.UserId != user.Spec.Id {
		util.ReturnHTTPMessage(w, r, 403, "forbidden", "you do not have access to scenario session")
		return
	}

	vm, err := sp.vmClient.GetVirtualMachineById(vmId)

	if err != nil {
		glog.Errorf("did not find the right virtual machine ID")
		util.ReturnHTTPMessage(w, r, 500, "error", "no vm found")
		return
	}

	glog.Infof("Going to upgrade connection now... %s", vm.Spec.Id)
}