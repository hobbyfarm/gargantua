package main

import (
	"sync"
	"time"

	"github.com/ebauman/crder"
	"github.com/hobbyfarm/gargantua/v3/pkg/microservices"
	"github.com/hobbyfarm/gargantua/v3/pkg/signals"
	"github.com/hobbyfarm/gargantua/v3/pkg/util"

	"github.com/golang/glog"
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

	crds := vmtemplateservice.GenerateVMSetCRD()
	glog.Info("installing/updating vm template CRDs")
	err := crder.InstallUpdateCRDs(cfg, crds...)
	if err != nil {
		glog.Fatalf("failed installing/updating vm template CRDs: %s", err.Error())
	}
	glog.Info("finished installing/updating vm template CRDs")

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
