package main

import (
	"sync"

	"github.com/ebauman/crder"
	"github.com/hobbyfarm/gargantua/v3/pkg/microservices"

	"github.com/golang/glog"
	vmsetservice "github.com/hobbyfarm/gargantua/services/vmsetsvc/v3/internal"
	vmsetProto "github.com/hobbyfarm/gargantua/v3/protos/vmset"
)

var (
	serviceConfig *microservices.ServiceConfig
)

func init() {
	serviceConfig = microservices.BuildServiceConfig()
}

func main() {
	cfg, hfClient := microservices.BuildClusterConfig(serviceConfig)

	crds := vmsetservice.GenerateVMSetCRD()
	glog.Info("installing/updating vm set CRDs")
	err := crder.InstallUpdateCRDs(cfg, crds...)
	if err != nil {
		glog.Fatalf("failed installing/updating vm set CRDs: %s", err.Error())
	}
	glog.Info("finished installing/updating vm set CRDs")

	gs := microservices.CreateGRPCServer(serviceConfig.ServerCert.Clone())

	ds := vmsetservice.NewGrpcVMSetServer(hfClient)
	vmsetProto.RegisterVMSetSvcServer(gs, ds)

	var wg sync.WaitGroup
	wg.Add(1)

	go func() {
		defer wg.Done()
		microservices.StartGRPCServer(gs, serviceConfig.EnableReflection)
	}()

	wg.Wait()
}
