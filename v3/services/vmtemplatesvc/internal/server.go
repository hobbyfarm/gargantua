package vmtemplateservice

import (
	authnpb "github.com/hobbyfarm/gargantua/v3/protos/authn"
	authrpb "github.com/hobbyfarm/gargantua/v3/protos/authr"
	scheduledeventpb "github.com/hobbyfarm/gargantua/v3/protos/scheduledevent"
	vmpb "github.com/hobbyfarm/gargantua/v3/protos/vm"
	vmclaimpb "github.com/hobbyfarm/gargantua/v3/protos/vmclaim"
	vmsetpb "github.com/hobbyfarm/gargantua/v3/protos/vmset"

	"github.com/golang/glog"
	"github.com/gorilla/mux"
)

type VirtualMachineTemplateServer struct {
	authnClient              authnpb.AuthNClient
	authrClient              authrpb.AuthRClient
	scheduledEventClient     scheduledeventpb.ScheduledEventSvcClient
	vmClient                 vmpb.VMSvcClient
	vmClaimClient            vmclaimpb.VMClaimSvcClient
	vmSetClient              vmsetpb.VMSetSvcClient
	internalVMTemplateServer *GrpcVMTemplateServer
}

func NewVirtualMachineTemplateServer(
	authnClient authnpb.AuthNClient,
	authrClient authrpb.AuthRClient,
	scheduledEventClient scheduledeventpb.ScheduledEventSvcClient,
	vmClient vmpb.VMSvcClient,
	vmClaimClient vmclaimpb.VMClaimSvcClient,
	vmSetClient vmsetpb.VMSetSvcClient,
	internalVMTemplateServer *GrpcVMTemplateServer,
) VirtualMachineTemplateServer {
	return VirtualMachineTemplateServer{
		authnClient:              authnClient,
		authrClient:              authrClient,
		scheduledEventClient:     scheduledEventClient,
		vmClient:                 vmClient,
		vmClaimClient:            vmClaimClient,
		vmSetClient:              vmSetClient,
		internalVMTemplateServer: internalVMTemplateServer,
	}
}

func (v VirtualMachineTemplateServer) SetupRoutes(r *mux.Router) {
	r.HandleFunc("/a/vmtemplate/list", v.ListFunc).Methods("GET")
	r.HandleFunc("/a/vmtemplate/{id}", v.GetFunc).Methods("GET")
	r.HandleFunc("/a/vmtemplate/create", v.CreateFunc).Methods("POST")
	r.HandleFunc("/a/vmtemplate/{id}/update", v.UpdateFunc).Methods("PUT")
	r.HandleFunc("/a/vmtemplate/{id}/delete", v.DeleteFunc).Methods("DELETE")
	glog.V(2).Infof("set up routes for admin vmtemplate server")
}
