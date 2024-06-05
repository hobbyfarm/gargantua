package vmservice

import (
	"github.com/golang/glog"
	"github.com/gorilla/mux"
	authnpb "github.com/hobbyfarm/gargantua/v3/protos/authn"
	authrpb "github.com/hobbyfarm/gargantua/v3/protos/authr"
	vmtemplatepb "github.com/hobbyfarm/gargantua/v3/protos/vmtemplate"
)

type VMServer struct {
	authnClient      authnpb.AuthNClient
	authrClient      authrpb.AuthRClient
	vmTemplateClient vmtemplatepb.VMTemplateSvcClient
	internalVMServer *GrpcVMServer
}

func NewVMServer(
	authnClient authnpb.AuthNClient,
	authrClient authrpb.AuthRClient,
	vmTemplateClient vmtemplatepb.VMTemplateSvcClient,
	internalVMServer *GrpcVMServer,
) VMServer {
	return VMServer{
		authnClient:      authnClient,
		authrClient:      authrClient,
		vmTemplateClient: vmTemplateClient,
		internalVMServer: internalVMServer,
	}
}

func (vms VMServer) SetupRoutes(r *mux.Router) {
	r.HandleFunc("/vm/{vm_id}", vms.GetVMFunc).Methods("GET")
	r.HandleFunc("/vm/getwebinterfaces/{vm_id}", vms.getWebinterfaces).Methods("GET")
	r.HandleFunc("/a/vm/list", vms.GetAllVMListFunc).Methods("GET")
	r.HandleFunc("/a/vm/scheduledevent/{se_id}", vms.GetVMListByScheduledEventFunc).Methods("GET")
	r.HandleFunc("/a/vm/count", vms.CountByScheduledEvent).Methods("GET")
	glog.V(2).Infof("set up routes")
}
