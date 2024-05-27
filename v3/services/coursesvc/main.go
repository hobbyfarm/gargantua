package main

import (
	"sync"
	"time"

	"github.com/hobbyfarm/gargantua/v3/pkg/crd"
	"github.com/hobbyfarm/gargantua/v3/pkg/microservices"
	"github.com/hobbyfarm/gargantua/v3/pkg/signals"
	"github.com/hobbyfarm/gargantua/v3/pkg/util"

	courseservice "github.com/hobbyfarm/gargantua/services/coursesvc/v3/internal"
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

	crd.InstallCrds(courseservice.CourseCRDInstaller{}, cfg, "course")

	services := []microservices.MicroService{
		microservices.AuthN,
		microservices.AuthR,
		microservices.AccessCode,
		microservices.Scenario,
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
	scenarioClient := scenariopb.NewScenarioSvcClient(connections[microservices.Scenario])
	scheduledEventClient := scheduledeventpb.NewScheduledEventSvcClient(connections[microservices.ScheduledEvent])
	sessionClient := sessionpb.NewSessionSvcClient(connections[microservices.Session])

	gs := microservices.CreateGRPCServer(serviceConfig.ServerCert.Clone())

	cs := courseservice.NewGrpcCourseServer(hfClient, hfInformerFactory)
	coursepb.RegisterCourseSvcServer(gs, cs)

	var wg sync.WaitGroup
	wg.Add(1)

	go func() {
		defer wg.Done()
		microservices.StartGRPCServer(gs, serviceConfig.EnableReflection)
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()

		courseServer := courseservice.NewCourseServer(
			authnClient,
			authrClient,
			acClient,
			scenarioClient,
			scheduledEventClient,
			sessionClient,
			cs,
		)
		microservices.StartAPIServer(courseServer)
	}()

	hfInformerFactory.Start(stopCh)

	wg.Wait()
}
