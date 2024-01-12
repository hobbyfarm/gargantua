package vmclaimserver

import (
	"encoding/json"
	"fmt"
	hfv1 "github.com/hobbyfarm/gargantua/v3/pkg/apis/hobbyfarm.io/v1"
	hfClientset "github.com/hobbyfarm/gargantua/v3/pkg/client/clientset/versioned"
	hfInformers "github.com/hobbyfarm/gargantua/v3/pkg/client/informers/externalversions"
	rbac2 "github.com/hobbyfarm/gargantua/v3/pkg/rbac"
	"github.com/hobbyfarm/gargantua/v3/pkg/util"
	"net/http"

	"github.com/golang/glog"
	"github.com/gorilla/mux"
	"github.com/hobbyfarm/gargantua/v3/protos/authn"
	"github.com/hobbyfarm/gargantua/v3/protos/authr"
	"k8s.io/client-go/tools/cache"
)

const (
	idIndex        = "vmcs.hobbyfarm.io/id-index"
	resourcePlural = rbac2.ResourcePluralVMClaim
)

type VMClaimServer struct {
	authnClient authn.AuthNClient
	authrClient authr.AuthRClient
	hfClientSet hfClientset.Interface

	vmClaimIndexer cache.Indexer
}

func NewVMClaimServer(authnClient authn.AuthNClient, authrClient authr.AuthRClient, hfClientset hfClientset.Interface, hfInformerFactory hfInformers.SharedInformerFactory) (*VMClaimServer, error) {
	vmcs := VMClaimServer{}

	vmcs.authnClient = authnClient
	vmcs.authrClient = authrClient
	vmcs.hfClientSet = hfClientset

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
	ID string `json:"id"`
	hfv1.VirtualMachineClaimSpec
	hfv1.VirtualMachineClaimStatus
}

func (vmcs VMClaimServer) GetVMClaimFunc(w http.ResponseWriter, r *http.Request) {
	user, err := rbac2.AuthenticateRequest(r, vmcs.authnClient)
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
		glog.Errorf("did not find the right virtual machine claim ID")
		util.ReturnHTTPMessage(w, r, http.StatusNotFound, "error", "no vm found")
		return
	}

	if vmc.Spec.UserId != user.GetId() {
		impersonatedUserId := user.GetId()
		authrResponse, err := rbac2.AuthorizeSimple(r, vmcs.authrClient, impersonatedUserId, rbac2.HobbyfarmPermission(resourcePlural, rbac2.VerbGet))
		if err != nil || !authrResponse.Success {
			util.ReturnHTTPMessage(w, r, 403, "forbidden", "access denied to get vmclaim")
			return
		}
	}

	preparedVMC := PreparedVirtualMachineClaim{vmc.Name, vmc.Spec, vmc.Status}

	encodedVMC, err := json.Marshal(preparedVMC)
	if err != nil {
		glog.Error(err)
	}
	util.ReturnHTTPContent(w, r, 200, "success", encodedVMC)

	glog.V(2).Infof("retrieved vmc %s", vmc.Name)
}

func vmcIdIndexer(obj interface{}) ([]string, error) {
	vmc, ok := obj.(*hfv1.VirtualMachineClaim)
	if !ok {
		return []string{}, nil
	}
	return []string{vmc.Name}, nil
}
