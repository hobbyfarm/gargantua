package main

import (
	"sync"
	"time"

	"github.com/hobbyfarm/gargantua/v3/pkg/crd"
	"github.com/hobbyfarm/gargantua/v3/pkg/microservices"
	"github.com/hobbyfarm/gargantua/v3/pkg/signals"
	"github.com/hobbyfarm/gargantua/v3/pkg/util"

	vmtemplateservice "github.com/hobbyfarm/gargantua/services/vmtemplatesvc/v3/internal"
	hfInformers "github.com/hobbyfarm/gargantua/v3/pkg/client/informers/externalversions"
	vmtemplateProto "github.com/hobbyfarm/gargantua/v3/protos/vmtemplate"
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

	crd.InstallCrds(vmtemplateservice.VMTemplateCRDInstaller{}, cfg, "virtual machine template")

	gs := microservices.CreateGRPCServer(serviceConfig.ServerCert.Clone())

	ds := vmtemplateservice.NewGrpcVMTemplateServer(hfClient, hfInformerFactory)

	vmtemplateProto.RegisterVMTemplateSvcServer(gs, ds)

	var wg sync.WaitGroup
	wg.Add(1)

	go func() {
		defer wg.Done()
		microservices.StartGRPCServer(gs, serviceConfig.EnableReflection)
	}()

	hfInformerFactory.Start(stopCh)

	wg.Wait()
}
