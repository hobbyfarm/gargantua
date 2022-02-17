package vmsetserver

import (
	"context"
	"encoding/json"
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
)

type VMSetServer struct {
	auth        *authclient.AuthClient
	hfClientSet hfClientset.Interface
	ctx 		context.Context
	vmIndexer cache.Indexer
}

type PreparedVirtualMachineSet struct {
	hfv1.VirtualMachineSetSpec
	hfv1.VirtualMachineSetStatus
}

func NewVMSetServer(authClient *authclient.AuthClient, hfClientset hfClientset.Interface, hfInformerFactory hfInformers.SharedInformerFactory, ctx context.Context) (*VMSetServer, error) {
	vms := VMSetServer{}

	vms.hfClientSet = hfClientset
	vms.auth = authClient
	vms.ctx = ctx

	inf := hfInformerFactory.Hobbyfarm().V1().VirtualMachines().Informer()
	indexers := map[string]cache.IndexFunc{idIndex: vmIdIndexer}
	inf.AddIndexers(indexers)
	vms.vmIndexer = inf.GetIndexer()

	return &vms, nil
}

func (vms VMSetServer) SetupRoutes(r *mux.Router) {
	r.HandleFunc("/a/vmset", vms.GetVMSetListFunc).Methods("GET")
	glog.V(2).Infof("set up routes")
}

func (vms VMSetServer) GetVMSetListFunc(w http.ResponseWriter, r *http.Request) {
	_, err := vms.auth.AuthNAdmin(w, r)
	if err != nil {
		util.ReturnHTTPMessage(w, r, 403, "forbidden", "no access to list vmsets")
		return
	}

	vmSetList, err := vms.hfClientSet.HobbyfarmV1().VirtualMachineSets(util.GetReleaseNamespace()).List(vms.ctx, metav1.ListOptions{})

	if err != nil {
		glog.Errorf("error while retrieving vmsets %v", err)
		util.ReturnHTTPMessage(w, r, 500, "error", "error retreiving vmsets")
		return
	}

	preparedVMSets := []PreparedVirtualMachineSet{}
	for _, vmSet := range vmSetList.Items {
		pVMSet := PreparedVirtualMachineSet{vmSet.Spec, vmSet.Status}
		preparedVMSets = append(preparedVMSets, pVMSet)
	}

	encodedVMSets, err := json.Marshal(preparedVMSets)
	if err != nil {
		glog.Error(err)
	}
	util.ReturnHTTPContent(w, r, 200, "success", encodedVMSets)
}

func vmIdIndexer(obj interface{}) ([]string, error) {
	vm, ok := obj.(*hfv1.VirtualMachine)
	if !ok {
		return []string{}, nil
	}
	return []string{vm.Spec.Id}, nil
}
