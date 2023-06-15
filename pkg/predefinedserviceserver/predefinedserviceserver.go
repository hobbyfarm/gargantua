package predefinedservicesserver

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/golang/glog"
	"github.com/gorilla/mux"
	hfv1 "github.com/hobbyfarm/gargantua/pkg/apis/hobbyfarm.io/v1"
	hfClientset "github.com/hobbyfarm/gargantua/pkg/client/clientset/versioned"
	"github.com/hobbyfarm/gargantua/pkg/rbac"
	"github.com/hobbyfarm/gargantua/pkg/util"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type AdminPreparedPredefinedService struct {
	ID string `json:"id"`
	hfv1.ServiceSpec
}

type PredefinedServiceServer struct {
	tlsCA       string
	hfClientSet hfClientset.Interface
	ctx         context.Context
}

func NewPredefinedServiceServer(tlsCA string, hfClientset hfClientset.Interface, ctx context.Context) (*PredefinedServiceServer, error) {
	pss := PredefinedServiceServer{}

	pss.hfClientSet = hfClientset
	pss.tlsCA = tlsCA
	pss.ctx = ctx
	return &pss, nil
}

func (s PredefinedServiceServer) SetupRoutes(r *mux.Router) {
	r.HandleFunc("/a/predefinedservices/list", s.ListFunc).Methods("GET")
	glog.V(2).Infof("set up routes for PredefinedServiceServer")
}

func (s PredefinedServiceServer) ListFunc(w http.ResponseWriter, r *http.Request) {
	user, err := rbac.AuthenticateRequest(r, s.tlsCA)
	if err != nil {
		util.ReturnHTTPMessage(w, r, 401, "unauthorized", "authentication failed")
		return
	}

	impersonatedUserId := user.GetId()
	authrResponse, err := rbac.AuthorizeSimple(r, s.tlsCA, impersonatedUserId, rbac.HobbyfarmPermission(rbac.ResourcePluralVMTemplate, rbac.VerbList))
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
