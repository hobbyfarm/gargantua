package vmserver

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	hfv1 "github.com/hobbyfarm/gargantua/v3/pkg/apis/hobbyfarm.io/v1"
	hfClientset "github.com/hobbyfarm/gargantua/v3/pkg/client/clientset/versioned"
	hfInformers "github.com/hobbyfarm/gargantua/v3/pkg/client/informers/externalversions"
	hflabels "github.com/hobbyfarm/gargantua/v3/pkg/labels"
	rbac2 "github.com/hobbyfarm/gargantua/v3/pkg/rbac"
	util2 "github.com/hobbyfarm/gargantua/v3/pkg/util"

	"github.com/golang/glog"
	"github.com/gorilla/mux"
	authnpb "github.com/hobbyfarm/gargantua/v3/protos/authn"
	authrpb "github.com/hobbyfarm/gargantua/v3/protos/authr"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/cache"
)

const (
	idIndex        = "vms.hobbyfarm.io/id-index"
	resourcePlural = rbac2.ResourcePluralVM
)

type VMServer struct {
	authnClient authnpb.AuthNClient
	authrClient authrpb.AuthRClient
	hfClientSet hfClientset.Interface
	ctx         context.Context
	vmIndexer   cache.Indexer
}

type PreparedVirtualMachine struct {
	ID string `json:"id"`
	hfv1.VirtualMachineSpec
	hfv1.VirtualMachineStatus
}

func NewVMServer(authnClient authnpb.AuthNClient, authrClient authrpb.AuthRClient, hfClientset hfClientset.Interface, hfInformerFactory hfInformers.SharedInformerFactory, ctx context.Context) (*VMServer, error) {
	vms := VMServer{}

	vms.authnClient = authnClient
	vms.authrClient = authrClient
	vms.hfClientSet = hfClientset
	vms.ctx = ctx

	inf := hfInformerFactory.Hobbyfarm().V1().VirtualMachines().Informer()
	indexers := map[string]cache.IndexFunc{idIndex: vmIdIndexer}
	inf.AddIndexers(indexers)
	vms.vmIndexer = inf.GetIndexer()

	return &vms, nil
}

func (vms VMServer) SetupRoutes(r *mux.Router) {
	r.HandleFunc("/vm/{vm_id}", vms.GetVMFunc).Methods("GET")
	r.HandleFunc("/vm/getwebinterfaces/{vm_id}", vms.getWebinterfaces).Methods("GET")
	r.HandleFunc("/a/vm/list", vms.GetAllVMListFunc).Methods("GET")
	r.HandleFunc("/a/vm/scheduledevent/{se_id}", vms.GetVMListByScheduledEventFunc).Methods("GET")
	r.HandleFunc("/a/vm/count", vms.CountByScheduledEvent).Methods("GET")
	glog.V(2).Infof("set up routes")
}

/*
* Checks if VMTemplate used to create VM has "webinterfaces" in ConfigMap.
* Returns those webinterface definitions or http Error Codes.
 */
func (vms VMServer) getWebinterfaces(w http.ResponseWriter, r *http.Request) {
	// Check if User has access to VMs
	user, err := rbac2.AuthenticateRequest(r, vms.authnClient)
	if err != nil {
		util2.ReturnHTTPMessage(w, r, 403, "forbidden", "no access to get vm")
		return
	}
	vars := mux.Vars(r)
	// Check if id for the VM was provided
	vmId := vars["vm_id"]
	if len(vmId) == 0 {
		util2.ReturnHTTPMessage(w, r, 500, "error", "no vm id passed in")
		return
	}
	// Get the VM, Error if none is found for the given id
	vm, err := vms.GetVirtualMachineById(vmId)
	if err != nil {
		glog.Errorf("did not find the right virtual machine ID")
		util2.ReturnHTTPMessage(w, r, 500, "error", "no vm found")
		return
	}

	// Check if the VM belongs to the User or User has RBAC-Rights to access VMs
	if vm.Spec.UserId != user.GetId() {
		impersonatedUserId := user.GetId()
		authrResponse, err := rbac2.AuthorizeSimple(r, vms.authrClient, impersonatedUserId, rbac2.HobbyfarmPermission(resourcePlural, rbac2.VerbGet))
		if err != nil || !authrResponse.Success {
			glog.Errorf("user forbidden from accessing vm id %s", vm.Name)
			util2.ReturnHTTPMessage(w, r, 403, "forbidden", "no access to get vm")
			return
		}
	}

	// Get the corresponding VMTemplate for the VM and Check for "ide"
	vmt, err := vms.hfClientSet.HobbyfarmV1().VirtualMachineTemplates(util2.GetReleaseNamespace()).Get(vms.ctx, vm.Spec.VirtualMachineTemplateId, v1.GetOptions{})
	if err != nil {
		util2.ReturnHTTPMessage(w, r, 404, "error", "no vm template found")
		return
	}

	services, found := vmt.Spec.ConfigMap["webinterfaces"]
	if !found {
		util2.ReturnHTTPMessage(w, r, 404, "error", "No Webinterfaces found for this VM")
		return
	}

	encodedWebinterfaceDefinitions, err := json.Marshal(services)
	if err != nil {
		glog.Error(err)
	}
	util2.ReturnHTTPContent(w, r, 200, "success", encodedWebinterfaceDefinitions)
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

func (vms VMServer) GetVMFunc(w http.ResponseWriter, r *http.Request) {
	user, err := rbac2.AuthenticateRequest(r, vms.authnClient)
	if err != nil {
		util2.ReturnHTTPMessage(w, r, 403, "forbidden", "no access to get vm")
		return
	}

	vars := mux.Vars(r)

	vmId := vars["vm_id"]

	if len(vmId) == 0 {
		util2.ReturnHTTPMessage(w, r, 500, "error", "no vm id passed in")
		return
	}

	vm, err := vms.GetVirtualMachineById(vmId)

	if err != nil {
		glog.Errorf("did not find the right virtual machine ID")
		util2.ReturnHTTPMessage(w, r, http.StatusNotFound, "error", "no vm found")
		return
	}

	if vm.Spec.UserId != user.GetId() {
		impersonatedUserId := user.GetId()
		authrResponse, err := rbac2.AuthorizeSimple(r, vms.authrClient, impersonatedUserId, rbac2.HobbyfarmPermission(resourcePlural, rbac2.VerbGet))
		if err != nil || !authrResponse.Success {
			glog.Errorf("user forbidden from accessing vm id %s", vm.Name)
			util2.ReturnHTTPMessage(w, r, 403, "forbidden", "no access to get vm")
			return
		}
	}

	preparedVM := PreparedVirtualMachine{vm.Name, vm.Spec, vm.Status}

	encodedVM, err := json.Marshal(preparedVM)
	if err != nil {
		glog.Error(err)
	}
	util2.ReturnHTTPContent(w, r, 200, "success", encodedVM)

	glog.V(2).Infof("retrieved vm %s", vm.Name)
}

func (vms VMServer) GetVMListFunc(w http.ResponseWriter, r *http.Request, listOptions metav1.ListOptions) {
	user, err := rbac2.AuthenticateRequest(r, vms.authnClient)
	if err != nil {
		util2.ReturnHTTPMessage(w, r, 401, "unauthorized", "authentication failed")
		return
	}

	impersonatedUserId := user.GetId()
	authrResponse, err := rbac2.AuthorizeSimple(r, vms.authrClient, impersonatedUserId, rbac2.HobbyfarmPermission(resourcePlural, rbac2.VerbList))
	if err != nil || !authrResponse.Success {
		util2.ReturnHTTPMessage(w, r, 403, "forbidden", "no access to list virtualmachines")
		return
	}

	vmList, err := vms.hfClientSet.HobbyfarmV1().VirtualMachines(util2.GetReleaseNamespace()).List(vms.ctx, listOptions)

	if err != nil {
		glog.Errorf("error while retrieving vms %v", err)
		util2.ReturnHTTPMessage(w, r, 500, "error", "error retreiving vms")
		return
	}

	preparedVMs := []PreparedVirtualMachine{}
	for _, vm := range vmList.Items {
		pVM := PreparedVirtualMachine{vm.Name, vm.Spec, vm.Status}
		preparedVMs = append(preparedVMs, pVM)
	}

	encodedVMs, err := json.Marshal(preparedVMs)
	if err != nil {
		glog.Error(err)
	}
	util2.ReturnHTTPContent(w, r, 200, "success", encodedVMs)
}

func (vms VMServer) GetVMListByScheduledEventFunc(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)

	id := vars["se_id"]

	if len(id) == 0 {
		util2.ReturnHTTPMessage(w, r, 500, "error", "no scheduledEvent id passed in")
		return
	}

	lo := metav1.ListOptions{LabelSelector: fmt.Sprintf("%s=%s", hflabels.ScheduledEventLabel, id)}

	vms.GetVMListFunc(w, r, lo)
}

func (vms VMServer) CountByScheduledEvent(w http.ResponseWriter, r *http.Request) {
	user, err := rbac2.AuthenticateRequest(r, vms.authnClient)
	if err != nil {
		util2.ReturnHTTPMessage(w, r, 401, "unauthorized", "authentication failed")
		return
	}

	impersonatedUserId := user.GetId()
	authrResponse, err := rbac2.AuthorizeSimple(r, vms.authrClient, impersonatedUserId, rbac2.HobbyfarmPermission(resourcePlural, rbac2.VerbList))
	if err != nil || !authrResponse.Success {
		util2.ReturnHTTPMessage(w, r, 403, "forbidden", "no access to list virtualmachines")
		return
	}

	virtualmachines, err := vms.hfClientSet.HobbyfarmV1().VirtualMachines(util2.GetReleaseNamespace()).List(vms.ctx, metav1.ListOptions{})
	if err != nil {
		glog.Errorf("error while retrieving virtualmachine %v", err)
		util2.ReturnHTTPMessage(w, r, 500, "error", "no virtualmachine found")
		return
	}

	countMap := map[string]int{}
	for _, vm := range virtualmachines.Items {
		se := vm.Labels[hflabels.ScheduledEventLabel]
		if _, ok := countMap[se]; ok {
			countMap[se] = countMap[se] + 1
		} else {
			countMap[se] = 1
		}
	}

	encodedMap, err := json.Marshal(countMap)
	if err != nil {
		glog.Error(err)
	}
	util2.ReturnHTTPContent(w, r, 200, "success", encodedMap)
}

func (vms VMServer) GetAllVMListFunc(w http.ResponseWriter, r *http.Request) {
	vms.GetVMListFunc(w, r, metav1.ListOptions{})
}

func vmIdIndexer(obj interface{}) ([]string, error) {
	vm, ok := obj.(*hfv1.VirtualMachine)
	if !ok {
		return []string{}, nil
	}
	return []string{vm.Name}, nil
}
