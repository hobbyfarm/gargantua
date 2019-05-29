package shell

import (
	"github.com/golang/glog"
	"github.com/gorilla/mux"
	"github.com/hobbyfarm/gargantua/pkg/authclient"
	hfClientset "github.com/hobbyfarm/gargantua/pkg/client/clientset/versioned"
	"github.com/hobbyfarm/gargantua/pkg/util"
	"github.com/hobbyfarm/gargantua/pkg/vmclient"
	"k8s.io/client-go/kubernetes"
	"net/http"
)

type ShellProxy struct {

	auth *authclient.AuthClient
	vmClient *vmclient.VirtualMachineClient

	hfClient *hfClientset.Clientset
	kubeClient *kubernetes.Clientset

}

func NewShellProxy(authClient *authclient.AuthClient, vmClient *vmclient.VirtualMachineClient, hfClientSet *hfClientset.Clientset, kubeClient *kubernetes.Clientset) (*ShellProxy, error) {
	shellProxy := ShellProxy{}

	shellProxy.auth = authClient
	shellProxy.vmClient = vmClient
	shellProxy.hfClient = hfClientSet
	shellProxy.kubeClient = kubeClient

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