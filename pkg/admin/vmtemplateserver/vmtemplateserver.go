package vmtemplateserver

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
	"net/http"
)

type AdminVirtualMachineTemplateServer struct {
	auth        *authclient.AuthClient
	hfClientSet *hfClientset.Clientset
}

func NewAdminVirtualMachineTemplateServer(authClient *authclient.AuthClient, hfClientset *hfClientset.Clientset) (*AdminVirtualMachineTemplateServer, error) {
	as := AdminVirtualMachineTemplateServer{}

	as.hfClientSet = hfClientset
	as.auth = authClient

	return &as, nil
}

func (a AdminVirtualMachineTemplateServer) getVirtualMachineTemplate(id string) (hfv1.VirtualMachineTemplate, error) {

	empty := hfv1.VirtualMachineTemplate{}

	if len(id) == 0 {
		return empty, fmt.Errorf("vm template id passed in was empty")
	}

	obj, err := a.hfClientSet.HobbyfarmV1().VirtualMachineTemplates().Get(id, metav1.GetOptions{})
	if err != nil {
		return empty, fmt.Errorf("error while retrieving Virtual Machine Template by id: %s with error: %v", id, err)
	}

	return *obj, nil

}

func (a AdminVirtualMachineTemplateServer) SetupRoutes(r *mux.Router) {
	r.HandleFunc("/a/vmtemplate/list", a.ListFunc).Methods("GET")
	r.HandleFunc("/a/vmtemplate/{id}", a.GetFunc).Methods("GET")
	glog.V(2).Infof("set up routes for admin vmtemplate server")
}

type PreparedVMTemplate struct {
	hfv1.VirtualMachineTemplateSpec
}

func (a AdminVirtualMachineTemplateServer) GetFunc(w http.ResponseWriter, r *http.Request) {
	_, err := a.auth.AuthNAdmin(w, r)
	if err != nil {
		util.ReturnHTTPMessage(w, r, 403, "forbidden", "no access to get vm template")
		return
	}

	vars := mux.Vars(r)

	vmtId := vars["id"]

	if len(vmtId) == 0 {
		util.ReturnHTTPMessage(w, r, 500, "error", "no vmt id passed in")
		return
	}

	vmt, err := a.getVirtualMachineTemplate(vmtId)

	if err != nil {
		glog.Errorf("error while retrieving virtual machine template %v", err)
		util.ReturnHTTPMessage(w, r, 500, "error", "no virtual machine template found")
		return
	}

	preparedEnvironment := PreparedVMTemplate{vmt.Spec}

	encodedEnvironment, err := json.Marshal(preparedEnvironment)
	if err != nil {
		glog.Error(err)
	}
	util.ReturnHTTPContent(w, r, 200, "success", encodedEnvironment)

	glog.V(2).Infof("retrieved vmt %s", vmt.Name)
}

func (a AdminVirtualMachineTemplateServer) ListFunc(w http.ResponseWriter, r *http.Request) {
	_, err := a.auth.AuthNAdmin(w, r)
	if err != nil {
		util.ReturnHTTPMessage(w, r, 403, "forbidden", "no access to list vmts")
		return
	}

	vmts, err := a.hfClientSet.HobbyfarmV1().VirtualMachineTemplates().List(metav1.ListOptions{})

	if err != nil {
		glog.Errorf("error while listing all vmts %v", err)
		util.ReturnHTTPMessage(w, r, 500, "error", "error listing all vmts")
		return
	}

	preparedVirtualMachineTemplates := []PreparedVMTemplate{}

	for _, vmt := range vmts.Items {
		preparedVirtualMachineTemplates = append(preparedVirtualMachineTemplates, PreparedVMTemplate{vmt.Spec})
	}

	encodedVirtualMachineTemplates, err := json.Marshal(preparedVirtualMachineTemplates)
	if err != nil {
		glog.Error(err)
	}
	util.ReturnHTTPContent(w, r, 200, "success", encodedVirtualMachineTemplates)

	glog.V(2).Infof("retrieved list of all environments")
}
