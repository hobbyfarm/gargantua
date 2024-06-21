package main

import (
	"sync"
	"time"

	"github.com/hobbyfarm/gargantua/v3/pkg/crd"
	"github.com/hobbyfarm/gargantua/v3/pkg/microservices"
	"github.com/hobbyfarm/gargantua/v3/pkg/signals"
	"github.com/hobbyfarm/gargantua/v3/pkg/util"

	accesscodeservice "github.com/hobbyfarm/gargantua/services/accesscodesvc/v3/internal"
	hfInformers "github.com/hobbyfarm/gargantua/v3/pkg/client/informers/externalversions"
	accesscodepb "github.com/hobbyfarm/gargantua/v3/protos/accesscode"
	scheduledeventpb "github.com/hobbyfarm/gargantua/v3/protos/scheduledevent"
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

	crd.InstallCrds(accesscodeservice.AccessCodeCRDInstaller{}, cfg, "access code")

	services := []microservices.MicroService{
		microservices.ScheduledEvent,
	}
	connections := microservices.EstablishConnections(services, serviceConfig.ClientCert)
	for _, conn := range connections {
		defer conn.Close()
	}

	eventClient := scheduledeventpb.NewScheduledEventSvcClient(connections[microservices.ScheduledEvent])

	gs := microservices.CreateGRPCServer(serviceConfig.ServerCert.Clone())
	as := accesscodeservice.NewGrpcAccessCodeServer(hfClient, hfInformerFactory, eventClient)
	accesscodepb.RegisterAccessCodeSvcServer(gs, as)

	var wg sync.WaitGroup
	wg.Add(1)

	go func() {
		defer wg.Done()
		microservices.StartGRPCServer(gs, serviceConfig.EnableReflection)
	}()

	hfInformerFactory.Start(stopCh)

	wg.Wait()
}
