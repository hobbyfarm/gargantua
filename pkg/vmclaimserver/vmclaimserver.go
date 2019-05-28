package vmclaimserver

import (
	"encoding/json"
	"fmt"
	"github.com/golang/glog"
	"github.com/gorilla/mux"
	hfv1 "github.com/hobbyfarm/gargantua/pkg/apis/hobbyfarm.io/v1"
	"github.com/hobbyfarm/gargantua/pkg/authclient"
	hfClientset "github.com/hobbyfarm/gargantua/pkg/client/clientset/versioned"
	hfInformers "github.com/hobbyfarm/gargantua/pkg/client/informers/externalversions"
	"github.com/hobbyfarm/gargantua/pkg/util"
	"k8s.io/client-go/tools/cache"
	"net/http"
)

const (
	idIndex = "vmcs.hobbyfarm.io/id-index"
)

type VMClaimServer struct {
	auth        *authclient.AuthClient
	hfClientSet *hfClientset.Clientset

	vmClaimIndexer cache.Indexer
}

func NewVMClaimServer(authClient *authclient.AuthClient, hfClientset *hfClientset.Clientset, hfInformerFactory hfInformers.SharedInformerFactory) (*VMClaimServer, error) {
	vmcs := VMClaimServer{}

	vmcs.hfClientSet = hfClientset
	vmcs.auth = authClient

	inf := hfInformerFactory.Hobbyfarm().V1().VirtualMachineClaims().Informer()
	indexers := map[string]cache.IndexFunc{idIndex: vmcIdIndexer}
	inf.AddIndexers(indexers)
	vmcs.vmClaimIndexer = inf.GetIndexer()

	return &vmcs, nil
}

func (vmcs VMClaimServer) GetVirtualMachineClaimById(id string) (hfv1.VirtualMachineClaim, error) {

	empty := hfv1.VirtualMachineClaim{}

	if len(id) == 0 {
		return empty, fmt.Errorf("vm claim id passed in was empty")
	}

	obj, err := vmcs.vmClaimIndexer.ByIndex(idIndex, id)
	if err != nil {
		return empty, fmt.Errorf("error while retrieving virtualmachineclaim by id: %s with error: %v", id, err)
	}

	if len(obj) < 1 {
		return empty, fmt.Errorf("virtualmachineclaim not found by id: %s", id)
	}

	result, ok := obj[0].(*hfv1.VirtualMachineClaim)

	if !ok {
		return empty, fmt.Errorf("error while converting virtualmachineclaim found by id to object: %s", id)
	}

	return *result, nil

}

func (vmcs VMClaimServer) SetupRoutes(r *mux.Router) {
	r.HandleFunc("/vmclaim/{vm_claim_id}", vmcs.GetVMClaimFunc).Methods("GET")
	glog.V(2).Infof("set up routes")
}

type PreparedVirtualMachineClaim struct {
	hfv1.VirtualMachineClaimSpec
	hfv1.VirtualMachineClaimStatus
}

func (vmcs VMClaimServer) GetVMClaimFunc(w http.ResponseWriter, r *http.Request) {
	user, err := vmcs.auth.AuthN(w, r)
	if err != nil {
		util.ReturnHTTPMessage(w, r, 403, "forbidden", "no access to get vmc")
		return
	}

	vars := mux.Vars(r)

	vmId := vars["vm_claim_id"]

	if len(vmId) == 0 {
		util.ReturnHTTPMessage(w, r, 500, "error", "no vmc id passed in")
		return
	}

	vmc, err := vmcs.GetVirtualMachineClaimById(vmId)

	if err != nil {
		glog.Errorf("did not find the right virtual machine ID")
		util.ReturnHTTPMessage(w, r, 500, "error", "no vm found")
		return
	}

	if vmc.Spec.UserId != user.Spec.Id {
		glog.Errorf("user forbidden from accessing vmc id %s", vmc.Spec.Id)
		util.ReturnHTTPMessage(w, r, 403, "forbidden", "forbidden")
	}

	preparedVMC := PreparedVirtualMachineClaim{vmc.Spec, vmc.Status}

	encodedVMC, err := json.Marshal(preparedVMC)
	if err != nil {
		glog.Error(err)
	}
	util.ReturnHTTPContent(w, r, 200, "success", encodedVMC)

	glog.V(2).Infof("retrieved vmc %s", vmc.Spec.Id)
}

func vmcIdIndexer(obj interface{}) ([]string, error) {
	vmc, ok := obj.(*hfv1.VirtualMachineClaim)
	if !ok {
		return []string{}, nil
	}
	return []string{vmc.Spec.Id}, nil
}
