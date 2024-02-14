package main

import (
	"sync"

	"github.com/ebauman/crder"
	"github.com/hobbyfarm/gargantua/v3/pkg/microservices"

	"github.com/golang/glog"
	dbconfigservice "github.com/hobbyfarm/gargantua/services/dbconfigsvc/v3/internal"
	dbconfigProto "github.com/hobbyfarm/gargantua/v3/protos/dbconfig"
)

var (
	serviceConfig *microservices.ServiceConfig
)

func init() {
	serviceConfig = microservices.BuildServiceConfig()
}

func main() {
	cfg, hfClient := microservices.BuildClusterConfig(serviceConfig)

	crds := dbconfigservice.GenerateDynamicBindConfigurationCRD()
	glog.Info("installing/updating dynamicbindconfiguration CRDs")
	err := crder.InstallUpdateCRDs(cfg, crds...)
	if err != nil {
		glog.Fatalf("failed installing/updating dynamicbindconfiguration CRDs: %s", err.Error())
	}
	glog.Info("finished installing/updating dynamicbindconfiguration CRDs")

	gs := microservices.CreateGRPCServer(serviceConfig.ServerCert.Clone())

	ds := dbconfigservice.NewGrpcDynamicBindConfigurationServer(hfClient)
	dbconfigProto.RegisterDynamicBindConfigSvcServer(gs, ds)

	var wg sync.WaitGroup
	wg.Add(1)

	go func() {
		defer wg.Done()
		microservices.StartGRPCServer(gs, serviceConfig.EnableReflection)
	}()

	wg.Wait()
}
