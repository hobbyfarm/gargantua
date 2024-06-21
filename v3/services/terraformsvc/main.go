package main

import (
	"sync"
	"time"

	"github.com/hobbyfarm/gargantua/v3/pkg/crd"
	"github.com/hobbyfarm/gargantua/v3/pkg/microservices"
	"github.com/hobbyfarm/gargantua/v3/pkg/signals"
	"github.com/hobbyfarm/gargantua/v3/pkg/util"

	terraformservice "github.com/hobbyfarm/gargantua/services/terraformsvc/v3/internal"
	hfInformers "github.com/hobbyfarm/gargantua/v3/pkg/client/informers/externalversions"
	terraformpb "github.com/hobbyfarm/gargantua/v3/protos/terraform"
)

var (
	serviceConfig *microservices.ServiceConfig
)

func init() {
	serviceConfig = microservices.BuildServiceConfig()
}

func main() {
	stopCh := signals.SetupSignalHandler()

	cfg, hfClient, _ := microservices.BuildClusterConfig(serviceConfig)

	namespace := util.GetReleaseNamespace()
	hfInformerFactory := hfInformers.NewSharedInformerFactoryWithOptions(hfClient, time.Second*30, hfInformers.WithNamespace(namespace))

	crd.InstallCrds(terraformservice.TerraformCRDInstaller{}, cfg, "terraform")

	gs := microservices.CreateGRPCServer(serviceConfig.ServerCert.Clone())

	ts := terraformservice.NewGrpcTerraformServer(hfClient, hfInformerFactory)
	terraformpb.RegisterTerraformSvcServer(gs, ts)

	var wg sync.WaitGroup
	wg.Add(1)

	go func() {
		defer wg.Done()
		microservices.StartGRPCServer(gs, serviceConfig.EnableReflection)
	}()

	hfInformerFactory.Start(stopCh)

	wg.Wait()
}
