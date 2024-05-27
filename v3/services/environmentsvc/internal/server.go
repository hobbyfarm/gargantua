package environmentservice

import (
	"github.com/golang/glog"
	"github.com/gorilla/mux"
	authnpb "github.com/hobbyfarm/gargantua/v3/protos/authn"
	authrpb "github.com/hobbyfarm/gargantua/v3/protos/authr"
	scheduledeventpb "github.com/hobbyfarm/gargantua/v3/protos/scheduledevent"
)

type EnvironmentServer struct {
	authnClient               authnpb.AuthNClient
	authrClient               authrpb.AuthRClient
	scheduledEventClient      scheduledeventpb.ScheduledEventSvcClient
	internalEnvironmentServer *GrpcEnvironmentServer
}

func NewEnvironmentServer(
	authnClient authnpb.AuthNClient,
	authrClient authrpb.AuthRClient,
	scheduledEventClient scheduledeventpb.ScheduledEventSvcClient,
	internalEnvironmentServer *GrpcEnvironmentServer,
) EnvironmentServer {
	return EnvironmentServer{
		authnClient:               authnClient,
		authrClient:               authrClient,
		scheduledEventClient:      scheduledEventClient,
		internalEnvironmentServer: internalEnvironmentServer,
	}
}

func (e EnvironmentServer) SetupRoutes(r *mux.Router) {
	r.HandleFunc("/a/environment/list", e.ListFunc).Methods("GET")
	r.HandleFunc("/a/environment/{id}", e.GetFunc).Methods("GET")
	r.HandleFunc("/a/environment/create", e.CreateFunc).Methods("POST")
	r.HandleFunc("/a/environment/{id}/update", e.UpdateFunc).Methods("PUT")
	r.HandleFunc("/a/environment/{environment_id}/available", e.PostEnvironmentAvailableFunc).Methods("POST")
	glog.V(2).Infof("set up routes for environment server")
}
