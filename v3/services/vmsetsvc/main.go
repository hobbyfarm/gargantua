package main

import (
	"sync"
	"time"

	"github.com/hobbyfarm/gargantua/v3/pkg/crd"
	"github.com/hobbyfarm/gargantua/v3/pkg/microservices"
	"github.com/hobbyfarm/gargantua/v3/pkg/signals"
	"github.com/hobbyfarm/gargantua/v3/pkg/util"

	vmsetservice "github.com/hobbyfarm/gargantua/services/vmsetsvc/v3/internal"
	hfInformers "github.com/hobbyfarm/gargantua/v3/pkg/client/informers/externalversions"
	vmsetProto "github.com/hobbyfarm/gargantua/v3/protos/vmset"
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

	crd.InstallCrds(vmsetservice.VMSetCRDInstaller{}, cfg, "virtual machine set")

	gs := microservices.CreateGRPCServer(serviceConfig.ServerCert.Clone())

	ds := vmsetservice.NewGrpcVMSetServer(hfClient, hfInformerFactory)
	vmsetProto.RegisterVMSetSvcServer(gs, ds)

	var wg sync.WaitGroup
	wg.Add(1)

	go func() {
		defer wg.Done()
		microservices.StartGRPCServer(gs, serviceConfig.EnableReflection)
	}()

	hfInformerFactory.Start(stopCh)

	wg.Wait()
}
