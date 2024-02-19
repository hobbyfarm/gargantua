package main

import (
	"sync"

	"github.com/ebauman/crder"
	"github.com/hobbyfarm/gargantua/v3/pkg/microservices"

	"github.com/golang/glog"
	progressService "github.com/hobbyfarm/gargantua/services/progresssvc/v3/internal"
	progressProto "github.com/hobbyfarm/gargantua/v3/protos/progress"
)

var (
	serviceConfig *microservices.ServiceConfig
)

func init() {
	serviceConfig = microservices.BuildServiceConfig()
}

func main() {
	cfg, hfClient := microservices.BuildClusterConfig(serviceConfig)

	crds := progressService.GenerateProgressCRD()
	glog.Info("installing/updating progress CRDs")
	err := crder.InstallUpdateCRDs(cfg, crds...)
	if err != nil {
		glog.Fatalf("failed installing/updating progress CRDs: %s", err.Error())
	}
	glog.Info("finished installing/updating progress CRDs")

	gs := microservices.CreateGRPCServer(serviceConfig.ServerCert.Clone())

	ds := progressService.NewGrpcProgressServer(hfClient)
	progressProto.RegisterProgressSvcServer(gs, ds)

	var wg sync.WaitGroup
	wg.Add(1)

	go func() {
		defer wg.Done()
		microservices.StartGRPCServer(gs, serviceConfig.EnableReflection)
	}()

	wg.Wait()
}
