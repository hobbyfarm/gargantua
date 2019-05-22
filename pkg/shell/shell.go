package shell

import (
	"github.com/golang/glog"
	"github.com/gorilla/mux"
	"github.com/hobbyfarm/gargantua/pkg/authclient"
	"github.com/hobbyfarm/gargantua/pkg/util"
	"github.com/hobbyfarm/gargantua/pkg/vmclient"
	"net/http"
)

type ShellProxy struct {

	auth *authclient.AuthClient
	vmClient *vmclient.VirtualMachineClient

}

func NewShellProxy(authClient *authclient.AuthClient, vmClient *vmclient.VirtualMachineClient) (*ShellProxy, error) {
	shellProxy := ShellProxy{}

	shellProxy.auth = authClient
	shellProxy.vmClient = vmClient

	return &shellProxy, nil
}


func (sp ShellProxy) SetupRoutes(r *mux.Router) {
	r.HandleFunc("/shell/{vm_id}/connect", sp.ConnectFunc)
	glog.V(2).Infof("set up routes")
}

func (sp ShellProxy) ConnectFunc(w http.ResponseWriter, r *http.Request) {
	user, err := sp.auth.AuthN(w, r)
	if err != nil {
		util.ReturnHTTPMessage(w, r, 403, "forbidden", "no access to get vm")
		return
	}

	vars := mux.Vars(r)

	vmId := vars["vm_id"]

	if len(vmId) == 0 {
		util.ReturnHTTPMessage(w, r, 500, "error", "no vm id passed in")
		return
	}

	vm, err := sp.vmClient.GetVirtualMachineById(vmId)

	if err != nil {
		glog.Errorf("did not find the right virtual machine ID")
		util.ReturnHTTPMessage(w, r, 500, "error", "no vm found")
		return
	}

	if vm.Spec.UserId != user.Spec.Id {
		util.ReturnHTTPMessage(w, r, 403, "forbidden", "you do not have access to shell")
		return
	}

	glog.Infof("Going to upgrade connection now... %s", vm.Spec.Id)
}