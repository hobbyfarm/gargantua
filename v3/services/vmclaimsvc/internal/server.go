package vmclaimservice

import (
	"github.com/hobbyfarm/gargantua/v3/pkg/rbac"

	"github.com/golang/glog"
	"github.com/gorilla/mux"
	authnpb "github.com/hobbyfarm/gargantua/v3/protos/authn"
	authrpb "github.com/hobbyfarm/gargantua/v3/protos/authr"
)

const (
	idIndex        = "vmcs.hobbyfarm.io/id-index"
	resourcePlural = rbac.ResourcePluralVMClaim
)

type VMClaimServer struct {
	authnClient           authnpb.AuthNClient
	authrClient           authrpb.AuthRClient
	internalVMClaimServer *GrpcVMClaimServer
}

func NewVMClaimServer(
	authnClient authnpb.AuthNClient,
	authrClient authrpb.AuthRClient,
	internalVMClaimServer *GrpcVMClaimServer,
) VMClaimServer {
	return VMClaimServer{
		authnClient:           authnClient,
		authrClient:           authrClient,
		internalVMClaimServer: internalVMClaimServer,
	}
}

func (vmcs VMClaimServer) SetupRoutes(r *mux.Router) {
	r.HandleFunc("/vmclaim/{vm_claim_id}", vmcs.GetVMClaimFunc).Methods("GET")
	glog.V(2).Infof("set up routes")
}
