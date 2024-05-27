package main

import (
	"sync"
	"time"

	"github.com/hobbyfarm/gargantua/v3/pkg/crd"
	"github.com/hobbyfarm/gargantua/v3/pkg/microservices"
	"github.com/hobbyfarm/gargantua/v3/pkg/signals"
	"github.com/hobbyfarm/gargantua/v3/pkg/util"

	environmentservice "github.com/hobbyfarm/gargantua/services/environmentsvc/v3/internal"
	hfInformers "github.com/hobbyfarm/gargantua/v3/pkg/client/informers/externalversions"
	authnpb "github.com/hobbyfarm/gargantua/v3/protos/authn"
	authrpb "github.com/hobbyfarm/gargantua/v3/protos/authr"
	environmentpb "github.com/hobbyfarm/gargantua/v3/protos/environment"
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

	crd.InstallCrds(environmentservice.EnvironmentCRDInstaller{}, cfg, "environment")

	services := []microservices.MicroService{
		microservices.AuthN,
		microservices.AuthR,
		microservices.ScheduledEvent,
	}
	connections := microservices.EstablishConnections(services, serviceConfig.ClientCert)
	for _, conn := range connections {
		defer conn.Close()
	}

	authnClient := authnpb.NewAuthNClient(connections[microservices.AuthN])
	authrClient := authrpb.NewAuthRClient(connections[microservices.AuthR])
	scheduledEventClient := scheduledeventpb.NewScheduledEventSvcClient(connections[microservices.ScheduledEvent])

	gs := microservices.CreateGRPCServer(serviceConfig.ServerCert.Clone())

	es := environmentservice.NewGrpcEnvironmentServer(hfClient, hfInformerFactory)
	environmentpb.RegisterEnvironmentSvcServer(gs, es)

	var wg sync.WaitGroup
	wg.Add(1)

	go func() {
		defer wg.Done()
		microservices.StartGRPCServer(gs, serviceConfig.EnableReflection)
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()

		environmentServer := environmentservice.NewEnvironmentServer(
			authnClient,
			authrClient,
			scheduledEventClient,
			es,
		)
		microservices.StartAPIServer(environmentServer)
	}()

	hfInformerFactory.Start(stopCh)

	wg.Wait()
}
