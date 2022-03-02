package vmserver

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/golang/glog"
	"github.com/gorilla/mux"
	hfv1 "github.com/hobbyfarm/gargantua/pkg/apis/hobbyfarm.io/v1"
	"github.com/hobbyfarm/gargantua/pkg/authclient"
	hfClientset "github.com/hobbyfarm/gargantua/pkg/client/clientset/versioned"
	hfInformers "github.com/hobbyfarm/gargantua/pkg/client/informers/externalversions"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"github.com/hobbyfarm/gargantua/pkg/util"
	"k8s.io/client-go/tools/cache"
	"net/http"
)

const (
	idIndex = "vms.hobbyfarm.io/id-index"
	ScheduledEventLabel  = "hobbyfarm.io/scheduledevent"
)

type VMServer struct {
	auth        *authclient.AuthClient
	hfClientSet hfClientset.Interface
	ctx 		context.Context
	vmIndexer cache.Indexer
}

type PreparedVirtualMachine struct {
	hfv1.VirtualMachineSpec
	hfv1.VirtualMachineStatus
}

func NewVMServer(authClient *authclient.AuthClient, hfClientset hfClientset.Interface, hfInformerFactory hfInformers.SharedInformerFactory, ctx context.Context) (*VMServer, error) {
	vms := VMServer{}

	vms.hfClientSet = hfClientset
	vms.auth = authClient
	vms.ctx = ctx

	inf := hfInformerFactory.Hobbyfarm().V1().VirtualMachines().Informer()
	indexers := map[string]cache.IndexFunc{idIndex: vmIdIndexer}
	inf.AddIndexers(indexers)
	vms.vmIndexer = inf.GetIndexer()

	return &vms, nil
}

func (vms VMServer) SetupRoutes(r *mux.Router) {
	r.HandleFunc("/vm/{vm_id}", vms.GetVMFunc).Methods("GET")
	r.HandleFunc("/a/vm", vms.GetAllVMListFunc).Methods("GET")
	r.HandleFunc("/a/vm/{se}", vms.GetVMListByScheduledEventFunc).Methods("GET")
	glog.V(2).Infof("set up routes")
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
	user, err := vms.auth.AuthN(w, r)
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

	vm, err := vms.GetVirtualMachineById(vmId)

	if err != nil {
		glog.Errorf("did not find the right virtual machine ID")
		util.ReturnHTTPMessage(w, r, 500, "error", "no vm found")
		return
	}

	if vm.Spec.UserId != user.Spec.Id {
		glog.Errorf("user forbidden from accessing vm id %s", vm.Spec.Id)
		util.ReturnHTTPMessage(w, r, 403, "forbidden", "forbidden")
	}

	preparedVM := PreparedVirtualMachine{vm.Spec, vm.Status}

	encodedVM, err := json.Marshal(preparedVM)
	if err != nil {
		glog.Error(err)
	}
	util.ReturnHTTPContent(w, r, 200, "success", encodedVM)

	glog.V(2).Infof("retrieved vm %s", vm.Spec.Id)
}

func (vms VMServer) GetVMListFunc(w http.ResponseWriter, r *http.Request, listOptions metav1.ListOptions) {
	_, err := vms.auth.AuthNAdmin(w, r)
	if err != nil {
		util.ReturnHTTPMessage(w, r, 403, "forbidden", "no access to list vms")
		return
	}

	vmList, err := vms.hfClientSet.HobbyfarmV1().VirtualMachines(util.GetReleaseNamespace()).List(vms.ctx, listOptions)

	if err != nil {
		glog.Errorf("error while retrieving vms %v", err)
		util.ReturnHTTPMessage(w, r, 500, "error", "error retreiving vms")
		return
	}

	preparedVMs := []PreparedVirtualMachine{}
	for _, vm := range vmList.Items {
		pVM := PreparedVirtualMachine{vm.Spec, vm.Status}
		preparedVMs = append(preparedVMs, pVM)
	}

	encodedVMs, err := json.Marshal(preparedVMs)
	if err != nil {
		glog.Error(err)
	}
	util.ReturnHTTPContent(w, r, 200, "success", encodedVMs)
}

func (vms VMServer) GetVMListByScheduledEventFunc(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)

	id := vars["id"]

	if len(id) == 0 {
		util.ReturnHTTPMessage(w, r, 500, "error", "no scheduledEvent id passed in")
		return
	}

	lo := metav1.ListOptions{LabelSelector: fmt.Sprintf("%s=%s", ScheduledEventLabel, id)}

	vms.GetVMListFunc(w,r, lo)
}

func (vms VMServer) GetAllVMListFunc(w http.ResponseWriter, r *http.Request) {
	vms.GetVMListFunc(w,r, metav1.ListOptions{})
}

func vmIdIndexer(obj interface{}) ([]string, error) {
	vm, ok := obj.(*hfv1.VirtualMachine)
	if !ok {
		return []string{}, nil
	}
	return []string{vm.Spec.Id}, nil
}
