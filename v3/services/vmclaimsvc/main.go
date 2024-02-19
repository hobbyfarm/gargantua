package main

import (
	"sync"

	"github.com/ebauman/crder"
	"github.com/hobbyfarm/gargantua/v3/pkg/microservices"

	"github.com/golang/glog"
	vmclaimservice "github.com/hobbyfarm/gargantua/services/vmclaimsvc/v3/internal"
	vmclaimProto "github.com/hobbyfarm/gargantua/v3/protos/vmclaim"
)

var (
	serviceConfig *microservices.ServiceConfig
)

func init() {
	serviceConfig = microservices.BuildServiceConfig()
}

func main() {
	cfg, hfClient, _ := microservices.BuildClusterConfig(serviceConfig)

	crds := vmclaimservice.GenerateVMClaimCRD()
	glog.Info("installing/updating vm claim CRDs")
	err := crder.InstallUpdateCRDs(cfg, crds...)
	if err != nil {
		glog.Fatalf("failed installing/updating vm claim CRDs: %s", err.Error())
	}
	glog.Info("finished installing/updating vm claim CRDs")

	gs := microservices.CreateGRPCServer(serviceConfig.ServerCert.Clone())

	ds := vmclaimservice.NewGrpcVMClaimServer(hfClient)
	vmclaimProto.RegisterVMClaimSvcServer(gs, ds)

	var wg sync.WaitGroup
	wg.Add(1)

	go func() {
		defer wg.Done()
		microservices.StartGRPCServer(gs, serviceConfig.EnableReflection)
	}()

	wg.Wait()
}
