package vmsetservice

import (
	"github.com/golang/glog"
	"github.com/gorilla/mux"
	authnpb "github.com/hobbyfarm/gargantua/v3/protos/authn"
	authrpb "github.com/hobbyfarm/gargantua/v3/protos/authr"
)

type VMSetServer struct {
	authnClient         authnpb.AuthNClient
	authrClient         authrpb.AuthRClient
	internalVMSetServer *GrpcVMSetServer
}

func NewVMSetServer(
	authnClient authnpb.AuthNClient,
	authrClient authrpb.AuthRClient,
	internalVMSetServer *GrpcVMSetServer,
) VMSetServer {
	return VMSetServer{
		authnClient:         authnClient,
		authrClient:         authrClient,
		internalVMSetServer: internalVMSetServer,
	}
}

func (vms VMSetServer) SetupRoutes(r *mux.Router) {
	r.HandleFunc("/a/vmset/{se_id}", vms.GetVMSetListByScheduledEventFunc).Methods("GET")
	r.HandleFunc("/a/vmset", vms.GetAllVMSetListFunc).Methods("GET")
	glog.V(2).Infof("set up routes")
}
