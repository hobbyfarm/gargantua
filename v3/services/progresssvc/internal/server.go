package progressservice

import (
	"github.com/golang/glog"
	"github.com/gorilla/mux"
	authnpb "github.com/hobbyfarm/gargantua/v3/protos/authn"
	authrpb "github.com/hobbyfarm/gargantua/v3/protos/authr"
)

type ProgressServer struct {
	authnClient            authnpb.AuthNClient
	authrClient            authrpb.AuthRClient
	internalProgressServer *GrpcProgressServer
}

func NewProgressServer(
	authnClient authnpb.AuthNClient,
	authrClient authrpb.AuthRClient,
	internalProgressServer *GrpcProgressServer,
) ProgressServer {
	return ProgressServer{
		authnClient:            authnClient,
		authrClient:            authrClient,
		internalProgressServer: internalProgressServer,
	}
}

func (s ProgressServer) SetupRoutes(r *mux.Router) {
	r.HandleFunc("/a/progress/scheduledevent/{id}", s.ListByScheduledEventFunc).Methods("GET")
	r.HandleFunc("/a/progress/user/{id}", s.ListByUserFunc).Methods("GET")
	r.HandleFunc("/a/progress/count", s.CountByScheduledEvent).Methods("GET")
	r.HandleFunc("/a/progress/range", s.ListByRangeFunc).Methods("GET")
	r.HandleFunc("/progress/update/{id}", s.Update).Methods("POST")
	r.HandleFunc("/progress/list", s.ListForUserFunc).Methods("GET")
	glog.V(2).Infof("set up routes for ProgressServer")
}
