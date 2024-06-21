package eventservice

import (
	"github.com/golang/glog"
	"github.com/gorilla/mux"
	accesscodepb "github.com/hobbyfarm/gargantua/v3/protos/accesscode"
	authnpb "github.com/hobbyfarm/gargantua/v3/protos/authn"
	authrpb "github.com/hobbyfarm/gargantua/v3/protos/authr"
	dbconfigpb "github.com/hobbyfarm/gargantua/v3/protos/dbconfig"
	progresspb "github.com/hobbyfarm/gargantua/v3/protos/progress"
	sessionpb "github.com/hobbyfarm/gargantua/v3/protos/session"
	vmsetpb "github.com/hobbyfarm/gargantua/v3/protos/vmset"
)

// session
// vmset
// progress
type ScheduledEventServer struct {
	authnClient                  authnpb.AuthNClient
	authrClient                  authrpb.AuthRClient
	acClient                     accesscodepb.AccessCodeSvcClient
	dbconfigClient               dbconfigpb.DynamicBindConfigSvcClient
	progressClient               progresspb.ProgressSvcClient
	sessionClient                sessionpb.SessionSvcClient
	vmsetClient                  vmsetpb.VMSetSvcClient
	internalScheduledEventServer *GrpcScheduledEventServer
}

func NewScheduledEventServer(
	authnClient authnpb.AuthNClient,
	authrClient authrpb.AuthRClient,
	acClient accesscodepb.AccessCodeSvcClient,
	dbconfigClient dbconfigpb.DynamicBindConfigSvcClient,
	progressClient progresspb.ProgressSvcClient,
	sessionClient sessionpb.SessionSvcClient,
	vmsetClient vmsetpb.VMSetSvcClient,
	internalScheduledEventServer *GrpcScheduledEventServer,
) ScheduledEventServer {
	return ScheduledEventServer{
		authnClient:                  authnClient,
		authrClient:                  authrClient,
		acClient:                     acClient,
		dbconfigClient:               dbconfigClient,
		progressClient:               progressClient,
		sessionClient:                sessionClient,
		vmsetClient:                  vmsetClient,
		internalScheduledEventServer: internalScheduledEventServer,
	}
}

func (s ScheduledEventServer) SetupRoutes(r *mux.Router) {
	r.HandleFunc("/a/scheduledevent/list", s.ListFunc).Methods("GET")
	r.HandleFunc("/a/scheduledevent/new", s.CreateFunc).Methods("POST")
	r.HandleFunc("/a/scheduledevent/{id}", s.GetFunc).Methods("GET")
	r.HandleFunc("/a/scheduledevent/{id}", s.UpdateFunc).Methods("PUT")
	r.HandleFunc("/a/scheduledevent/{id}/otacs/add/{count}", s.GenerateOTACsFunc).Methods("POST")
	r.HandleFunc("/a/scheduledevent/{id}/otacs/delete/{otac}", s.DeleteOTACFunc).Methods("GET")
	r.HandleFunc("/a/scheduledevent/{id}/otacs/list", s.GetOTACsFunc).Methods("GET")
	r.HandleFunc("/a/scheduledevent/delete/{id}", s.DeleteFunc).Methods("DELETE")
	glog.V(2).Infof("set up routes for admin scheduledevent server")
}
