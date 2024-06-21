package scenarioservice

import (
	accesscodepb "github.com/hobbyfarm/gargantua/v3/protos/accesscode"
	authnpb "github.com/hobbyfarm/gargantua/v3/protos/authn"
	authrpb "github.com/hobbyfarm/gargantua/v3/protos/authr"
	coursepb "github.com/hobbyfarm/gargantua/v3/protos/course"
	scheduledeventpb "github.com/hobbyfarm/gargantua/v3/protos/scheduledevent"
	sessionpb "github.com/hobbyfarm/gargantua/v3/protos/session"

	"github.com/golang/glog"
	"github.com/gorilla/mux"
)

type ScenarioServer struct {
	authnClient            authnpb.AuthNClient
	authrClient            authrpb.AuthRClient
	acClient               accesscodepb.AccessCodeSvcClient
	courseClient           coursepb.CourseSvcClient
	scheduledEventClient   scheduledeventpb.ScheduledEventSvcClient
	sessionClient          sessionpb.SessionSvcClient
	internalScenarioServer *GrpcScenarioServer
}

func NewScenarioServer(
	authnClient authnpb.AuthNClient,
	authrClient authrpb.AuthRClient,
	acClient accesscodepb.AccessCodeSvcClient,
	courseClient coursepb.CourseSvcClient,
	scheduledEventClient scheduledeventpb.ScheduledEventSvcClient,
	sessionClient sessionpb.SessionSvcClient,
	internalScenarioServer *GrpcScenarioServer,
) ScenarioServer {
	return ScenarioServer{
		authnClient:            authnClient,
		authrClient:            authrClient,
		acClient:               acClient,
		courseClient:           courseClient,
		scheduledEventClient:   scheduledEventClient,
		sessionClient:          sessionClient,
		internalScenarioServer: internalScenarioServer,
	}
}

func (s ScenarioServer) SetupRoutes(r *mux.Router) {
	r.HandleFunc("/scenario/list/{access_code}", s.ListScenariosForAccessCode).Methods("GET")
	r.HandleFunc("/a/scenario/categories", s.ListCategories).Methods("GET")
	r.HandleFunc("/a/scenario/list/{category}", s.ListByCategoryFunc).Methods("GET")
	r.HandleFunc("/a/scenario/list", s.ListAllFunc).Methods("GET")
	r.HandleFunc("/a/scenario/{id}", s.AdminGetFunc).Methods("GET")
	r.HandleFunc("/a/scenario/{id}", s.AdminDeleteFunc).Methods("DELETE")
	r.HandleFunc("/scenario/{scenario_id}", s.GetScenarioFunc).Methods("GET")
	r.HandleFunc("/scenario/{id}/printable", s.PrintFunc).Methods("GET")
	r.HandleFunc("/a/scenario/{id}/printable", s.AdminPrintFunc).Methods("GET")
	r.HandleFunc("/a/scenario/new", s.CreateFunc).Methods("POST")
	r.HandleFunc("/a/scenario/copy/{id}", s.CopyFunc).Methods("POST")
	r.HandleFunc("/a/scenario/{id}", s.UpdateFunc).Methods("PUT")
	r.HandleFunc("/scenario/{scenario_id}/step/{step_id:[0-9]+}", s.GetScenarioStepFunc).Methods("GET")
	glog.V(2).Infof("set up route")
}
