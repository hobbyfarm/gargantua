package vmserver

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/golang/glog"
	"github.com/gorilla/mux"
	hfv1 "github.com/hobbyfarm/gargantua/pkg/apis/hobbyfarm.io/v1"
	"github.com/hobbyfarm/gargantua/pkg/authclient"
	hfClientset "github.com/hobbyfarm/gargantua/pkg/client/clientset/versioned"
	hfInformers "github.com/hobbyfarm/gargantua/pkg/client/informers/externalversions"
	"github.com/hobbyfarm/gargantua/pkg/rbacclient"
	"github.com/hobbyfarm/gargantua/pkg/util"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/cache"
)

const (
	idIndex             = "vms.hobbyfarm.io/id-index"
	resourcePlural		= "virtualmachines"
)

type VMServer struct {
	auth        *authclient.AuthClient
	hfClientSet hfClientset.Interface
	ctx         context.Context
	vmIndexer   cache.Indexer
}

type PreparedVirtualMachine struct {
	ID string `json:"id"`
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
	r.HandleFunc("/a/vm/list", vms.GetAllVMListFunc).Methods("GET")
	r.HandleFunc("/a/vm/scheduledevent/{se_id}", vms.GetVMListByScheduledEventFunc).Methods("GET")
	r.HandleFunc("/a/vm/count", vms.CountByScheduledEvent).Methods("GET")
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
		util.ReturnHTTPMessage(w, r, http.StatusNotFound, "error", "no vm found")
		return
	}

	if vm.Spec.UserId != user.Name {
		_, err := vms.auth.AuthGrant(rbacclient.RbacRequest().HobbyfarmPermission(resourcePlural, rbacclient.VerbGet), w, r)
		if err != nil {
			glog.Errorf("user forbidden from accessing vm id %s", vm.Name)
			util.ReturnHTTPMessage(w, r, 403, "forbidden", "no access to get vm")
			return
		}
	}

	preparedVM := PreparedVirtualMachine{vm.Name, vm.Spec, vm.Status}

	encodedVM, err := json.Marshal(preparedVM)
	if err != nil {
		glog.Error(err)
	}
	util.ReturnHTTPContent(w, r, 200, "success", encodedVM)

	glog.V(2).Infof("retrieved vm %s", vm.Name)
}

func (vms VMServer) GetVMListFunc(w http.ResponseWriter, r *http.Request, listOptions metav1.ListOptions) {
	_, err := vms.auth.AuthGrant(rbacclient.RbacRequest().HobbyfarmPermission(resourcePlural, rbacclient.VerbList), w, r)
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
		pVM := PreparedVirtualMachine{vm.Name, vm.Spec, vm.Status}
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

	id := vars["se_id"]

	if len(id) == 0 {
		util.ReturnHTTPMessage(w, r, 500, "error", "no scheduledEvent id passed in")
		return
	}

	lo := metav1.ListOptions{LabelSelector: fmt.Sprintf("%s=%s", util.ScheduledEventLabel, id)}

	vms.GetVMListFunc(w, r, lo)
}

func (vms VMServer) CountByScheduledEvent(w http.ResponseWriter, r *http.Request) {
	_, err := vms.auth.AuthGrant(rbacclient.RbacRequest().HobbyfarmPermission(resourcePlural, rbacclient.VerbList), w, r)
	if err != nil {
		util.ReturnHTTPMessage(w, r, 403, "forbidden", "no access to list virtualmachines")
		return
	}

	virtualmachines, err := vms.hfClientSet.HobbyfarmV1().VirtualMachines(util.GetReleaseNamespace()).List(vms.ctx, metav1.ListOptions{})
	if err != nil {
		glog.Errorf("error while retrieving virtualmachine %v", err)
		util.ReturnHTTPMessage(w, r, 500, "error", "no virtualmachine found")
		return
	}

	countMap := map[string]int{}
	for _, vm := range virtualmachines.Items {
		se := vm.Labels[util.ScheduledEventLabel]
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
	util.ReturnHTTPContent(w, r, 200, "success", encodedMap)
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
