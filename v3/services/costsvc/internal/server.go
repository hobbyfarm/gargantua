package costservice

import (
	"github.com/golang/glog"
	"github.com/gorilla/mux"
	authnpb "github.com/hobbyfarm/gargantua/v3/protos/authn"
	authrpb "github.com/hobbyfarm/gargantua/v3/protos/authr"
)

type CostServer struct {
	authnClient        authnpb.AuthNClient
	authrClient        authrpb.AuthRClient
	internalCostServer *GrpcCostServer
}

func NewCostServer(
	authnClient authnpb.AuthNClient,
	authrClient authrpb.AuthRClient,
	internalCostServer *GrpcCostServer,
) CostServer {
	return CostServer{
		authnClient:        authnClient,
		authrClient:        authrClient,
		internalCostServer: internalCostServer,
	}
}

func (cs CostServer) SetupRoutes(r *mux.Router) {
	r.HandleFunc("/a/cost/all/{cost_group}", cs.GetCostFunc).Methods("GET")
	r.HandleFunc("/a/cost/history/{cost_group}", cs.GetCostHistoryFunc).Methods("GET")
	r.HandleFunc("/a/cost/present/{cost_group}", cs.GetCostPresentFunc).Methods("GET")
	r.HandleFunc("/a/cost/detail/{cost_group}", cs.GetCostDetailFunc).Methods("GET")
	r.HandleFunc("/a/cost/list", cs.GetAllCostListFunc).Methods("GET")
	glog.V(2).Infof("set up routes")
}
