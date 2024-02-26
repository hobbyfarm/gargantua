package vmsetserver

import (
	"context"
	"encoding/json"
	"fmt"
	hfv1 "github.com/hobbyfarm/gargantua/v3/pkg/apis/hobbyfarm.io/v1"
	hfClientset "github.com/hobbyfarm/gargantua/v3/pkg/client/clientset/versioned"
	hfInformers "github.com/hobbyfarm/gargantua/v3/pkg/client/informers/externalversions"
	rbac2 "github.com/hobbyfarm/gargantua/v3/pkg/rbac"
	util2 "github.com/hobbyfarm/gargantua/v3/pkg/util"
	"net/http"

	"github.com/golang/glog"
	"github.com/gorilla/mux"
	"github.com/hobbyfarm/gargantua/v3/protos/authn"
	"github.com/hobbyfarm/gargantua/v3/protos/authr"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/cache"
)

const (
	idIndex        = "vms.hobbyfarm.io/id-index"
	resourcePlural = rbac2.ResourcePluralVMSet
)

type VMSetServer struct {
	authnClient authn.AuthNClient
	authrClient authr.AuthRClient
	hfClientSet hfClientset.Interface
	ctx         context.Context
	vmIndexer   cache.Indexer
}

type PreparedVirtualMachineSet struct {
	Id string `json:"id"`
	hfv1.VirtualMachineSetSpec
	hfv1.VirtualMachineSetStatus
}

func NewVMSetServer(authnClient authn.AuthNClient, authrClient authr.AuthRClient, hfClientset hfClientset.Interface, hfInformerFactory hfInformers.SharedInformerFactory, ctx context.Context) (*VMSetServer, error) {
	vms := VMSetServer{}

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

func (vms VMSetServer) SetupRoutes(r *mux.Router) {
	r.HandleFunc("/a/vmset/{se_id}", vms.GetVMSetListByScheduledEventFunc).Methods("GET")
	r.HandleFunc("/a/vmset", vms.GetAllVMSetListFunc).Methods("GET")
	glog.V(2).Infof("set up routes")
}

func (vms VMSetServer) GetVMSetListByScheduledEventFunc(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)

	id := vars["se_id"]

	if len(id) == 0 {
		util2.ReturnHTTPMessage(w, r, 500, "error", "no scheduledEvent id passed in")
		return
	}

	lo := metav1.ListOptions{LabelSelector: fmt.Sprintf("%s=%s", util2.ScheduledEventLabel, id)}

	vms.GetVMSetListFunc(w, r, lo)
}

func (vms VMSetServer) GetAllVMSetListFunc(w http.ResponseWriter, r *http.Request) {
	vms.GetVMSetListFunc(w, r, metav1.ListOptions{})
}

func (vms VMSetServer) GetVMSetListFunc(w http.ResponseWriter, r *http.Request, listOptions metav1.ListOptions) {
	user, err := rbac2.AuthenticateRequest(r, vms.authnClient)
	if err != nil {
		util2.ReturnHTTPMessage(w, r, 401, "unauthorized", "authentication failed")
		return
	}

	impersonatedUserId := user.GetId()
	authrResponse, err := rbac2.AuthorizeSimple(r, vms.authrClient, impersonatedUserId, rbac2.HobbyfarmPermission(resourcePlural, rbac2.VerbList))
	if err != nil || !authrResponse.Success {
		util2.ReturnHTTPMessage(w, r, 403, "forbidden", "no access to list vmsets")
		return
	}

	vmSetList, err := vms.hfClientSet.HobbyfarmV1().VirtualMachineSets(util2.GetReleaseNamespace()).List(vms.ctx, listOptions)

	if err != nil {
		glog.Errorf("error while retrieving vmsets %v", err)
		util2.ReturnHTTPMessage(w, r, 500, "error", "error retreiving vmsets")
		return
	}

	preparedVMSets := []PreparedVirtualMachineSet{}
	for _, vmSet := range vmSetList.Items {
		pVMSet := PreparedVirtualMachineSet{vmSet.Name, vmSet.Spec, vmSet.Status}
		preparedVMSets = append(preparedVMSets, pVMSet)
	}

	encodedVMSets, err := json.Marshal(preparedVMSets)
	if err != nil {
		glog.Error(err)
	}
	util2.ReturnHTTPContent(w, r, 200, "success", encodedVMSets)
}

func vmIdIndexer(obj interface{}) ([]string, error) {
	vm, ok := obj.(*hfv1.VirtualMachine)
	if !ok {
		return []string{}, nil
	}
	return []string{vm.Name}, nil
}
