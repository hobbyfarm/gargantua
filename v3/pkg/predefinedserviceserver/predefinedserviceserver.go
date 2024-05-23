package predefinedservicesserver

import (
	"context"
	"encoding/json"
	"net/http"

	hfv1 "github.com/hobbyfarm/gargantua/v3/pkg/apis/hobbyfarm.io/v1"
	hfClientset "github.com/hobbyfarm/gargantua/v3/pkg/client/clientset/versioned"
	rbac2 "github.com/hobbyfarm/gargantua/v3/pkg/rbac"
	"github.com/hobbyfarm/gargantua/v3/pkg/util"

	"github.com/golang/glog"
	"github.com/gorilla/mux"
	authnpb "github.com/hobbyfarm/gargantua/v3/protos/authn"
	authrpb "github.com/hobbyfarm/gargantua/v3/protos/authr"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type AdminPreparedPredefinedService struct {
	ID string `json:"id"`
	hfv1.ServiceSpec
}

type PredefinedServiceServer struct {
	authnClient authnpb.AuthNClient
	authrClient authrpb.AuthRClient
	hfClientSet hfClientset.Interface
	ctx         context.Context
}

func NewPredefinedServiceServer(authnClient authnpb.AuthNClient, authrClient authrpb.AuthRClient, hfClientset hfClientset.Interface, ctx context.Context) (*PredefinedServiceServer, error) {
	pss := PredefinedServiceServer{}

	pss.hfClientSet = hfClientset
	pss.authnClient = authnClient
	pss.authrClient = authrClient
	pss.ctx = ctx
	return &pss, nil
}

func (s PredefinedServiceServer) SetupRoutes(r *mux.Router) {
	r.HandleFunc("/a/predefinedservices/list", s.ListFunc).Methods("GET")
	glog.V(2).Infof("set up routes for PredefinedServiceServer")
}

func (s PredefinedServiceServer) ListFunc(w http.ResponseWriter, r *http.Request) {
	user, err := rbac2.AuthenticateRequest(r, s.authnClient)
	if err != nil {
		util.ReturnHTTPMessage(w, r, 401, "unauthorized", "authentication failed")
		return
	}

	impersonatedUserId := user.GetId()
	authrResponse, err := rbac2.AuthorizeSimple(r, s.authrClient, impersonatedUserId, rbac2.HobbyfarmPermission(rbac2.ResourcePluralVMTemplate, rbac2.VerbList))
	if err != nil || !authrResponse.Success {
		util.ReturnHTTPMessage(w, r, 403, "forbidden", "no access to list PredefinedServices")
		return
	}

	services, err := s.hfClientSet.HobbyfarmV1().PredefinedServices(util.GetReleaseNamespace()).List(s.ctx, metav1.ListOptions{})

	if err != nil {
		glog.Errorf("error while retrieving predefinedservices %v", err)
		util.ReturnHTTPMessage(w, r, 500, "error", "no PredefinedServices found")
		return
	}

	preparedServices := []AdminPreparedPredefinedService{}
	for _, s := range services.Items {
		pService := AdminPreparedPredefinedService{s.Name, s.Spec}
		preparedServices = append(preparedServices, pService)
	}

	encodedPredefinedServices, err := json.Marshal(preparedServices)
	if err != nil {
		glog.Error(err)
	}
	util.ReturnHTTPContent(w, r, 200, "success", encodedPredefinedServices)
}
