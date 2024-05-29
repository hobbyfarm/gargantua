package main

import (
	"sync"
	"time"

	"github.com/hobbyfarm/gargantua/v3/pkg/crd"
	"github.com/hobbyfarm/gargantua/v3/pkg/microservices"
	"github.com/hobbyfarm/gargantua/v3/pkg/signals"
	"github.com/hobbyfarm/gargantua/v3/pkg/util"

	scenarioservice "github.com/hobbyfarm/gargantua/services/scenariosvc/v3/internal"
	hfInformers "github.com/hobbyfarm/gargantua/v3/pkg/client/informers/externalversions"
	accesscodepb "github.com/hobbyfarm/gargantua/v3/protos/accesscode"
	authnpb "github.com/hobbyfarm/gargantua/v3/protos/authn"
	authrpb "github.com/hobbyfarm/gargantua/v3/protos/authr"
	coursepb "github.com/hobbyfarm/gargantua/v3/protos/course"
	scenariopb "github.com/hobbyfarm/gargantua/v3/protos/scenario"
	scheduledeventpb "github.com/hobbyfarm/gargantua/v3/protos/scheduledevent"
	sessionpb "github.com/hobbyfarm/gargantua/v3/protos/session"
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

	crd.InstallCrds(scenarioservice.ScenarioCRDInstaller{}, cfg, "scenario")

	services := []microservices.MicroService{
		microservices.AuthN,
		microservices.AuthR,
		microservices.AccessCode,
		microservices.Course,
		microservices.ScheduledEvent,
		microservices.Session,
	}
	connections := microservices.EstablishConnections(services, serviceConfig.ClientCert)
	for _, conn := range connections {
		defer conn.Close()
	}

	authnClient := authnpb.NewAuthNClient(connections[microservices.AuthN])
	authrClient := authrpb.NewAuthRClient(connections[microservices.AuthR])
	acClient := accesscodepb.NewAccessCodeSvcClient(connections[microservices.AccessCode])
	courseClient := coursepb.NewCourseSvcClient(connections[microservices.Course])
	scheduledEventClient := scheduledeventpb.NewScheduledEventSvcClient(connections[microservices.ScheduledEvent])
	sessionClient := sessionpb.NewSessionSvcClient(connections[microservices.Session])

	gs := microservices.CreateGRPCServer(serviceConfig.ServerCert.Clone())

	ss := scenarioservice.NewGrpcScenarioServer(hfClient, hfInformerFactory)
	scenariopb.RegisterScenarioSvcServer(gs, ss)

	var wg sync.WaitGroup
	wg.Add(1)

	go func() {
		defer wg.Done()
		microservices.StartGRPCServer(gs, serviceConfig.EnableReflection)
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()

		scenarioServer := scenarioservice.NewScenarioServer(
			authnClient,
			authrClient,
			acClient,
			courseClient,
			scheduledEventClient,
			sessionClient,
			ss,
		)
		microservices.StartAPIServer(scenarioServer)
	}()

	hfInformerFactory.Start(stopCh)

	wg.Wait()
}
