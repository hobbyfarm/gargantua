package main

import (
	"context"
	"sync"
	"time"

	"github.com/golang/glog"
	"github.com/hobbyfarm/gargantua/v3/pkg/crd"
	"github.com/hobbyfarm/gargantua/v3/pkg/microservices"
	"github.com/hobbyfarm/gargantua/v3/pkg/signals"
	"github.com/hobbyfarm/gargantua/v3/pkg/util"

	sessionservice "github.com/hobbyfarm/gargantua/services/sessionsvc/v3/internal"
	hfInformers "github.com/hobbyfarm/gargantua/v3/pkg/client/informers/externalversions"
	accesscodepb "github.com/hobbyfarm/gargantua/v3/protos/accesscode"
	authnpb "github.com/hobbyfarm/gargantua/v3/protos/authn"
	authrpb "github.com/hobbyfarm/gargantua/v3/protos/authr"
	coursepb "github.com/hobbyfarm/gargantua/v3/protos/course"
	progresspb "github.com/hobbyfarm/gargantua/v3/protos/progress"
	scenariopb "github.com/hobbyfarm/gargantua/v3/protos/scenario"
	scheduledeventpb "github.com/hobbyfarm/gargantua/v3/protos/scheduledevent"
	sessionpb "github.com/hobbyfarm/gargantua/v3/protos/session"
	vmpb "github.com/hobbyfarm/gargantua/v3/protos/vm"
	vmclaimpb "github.com/hobbyfarm/gargantua/v3/protos/vmclaim"
)

var (
	serviceConfig *microservices.ServiceConfig
)

func init() {
	serviceConfig = microservices.BuildServiceConfig()
}

func main() {
	stopCh := signals.SetupSignalHandler()
	ctx := context.Background()

	cfg, hfClient, kubeClient := microservices.BuildClusterConfig(serviceConfig)

	namespace := util.GetReleaseNamespace()
	hfInformerFactory := hfInformers.NewSharedInformerFactoryWithOptions(hfClient, time.Second*30, hfInformers.WithNamespace(namespace))

	crd.InstallCrds(sessionservice.SessionCRDInstaller{}, cfg, "session")

	services := []microservices.MicroService{
		microservices.AuthN,
		microservices.AuthR,
		microservices.AccessCode,
		microservices.Course,
		microservices.Progress,
		microservices.Scenario,
		microservices.ScheduledEvent,
		microservices.VM,
		microservices.VMClaim,
	}
	connections := microservices.EstablishConnections(services, serviceConfig.ClientCert)
	for _, conn := range connections {
		defer conn.Close()
	}

	authnClient := authnpb.NewAuthNClient(connections[microservices.AuthN])
	authrClient := authrpb.NewAuthRClient(connections[microservices.AuthR])
	acClient := accesscodepb.NewAccessCodeSvcClient(connections[microservices.AccessCode])
	courseClient := coursepb.NewCourseSvcClient(connections[microservices.Course])
	progressClient := progresspb.NewProgressSvcClient(connections[microservices.Progress])
	scenarioClient := scenariopb.NewScenarioSvcClient(connections[microservices.Scenario])
	scheduledEventClient := scheduledeventpb.NewScheduledEventSvcClient(connections[microservices.ScheduledEvent])
	vmClient := vmpb.NewVMSvcClient(connections[microservices.VM])
	vmClaimClient := vmclaimpb.NewVMClaimSvcClient(connections[microservices.VMClaim])

	gs := microservices.CreateGRPCServer(serviceConfig.ServerCert.Clone())

	ss := sessionservice.NewGrpcSessionServer(hfClient, hfInformerFactory)
	sessionpb.RegisterSessionSvcServer(gs, ss)
	sessionController, err := sessionservice.NewSessionController(
		kubeClient,
		ss,
		hfInformerFactory,
		progressClient,
		vmClient,
		vmClaimClient,
		ctx,
	)
	if err != nil {
		glog.Fatalf("failed creating scheduled event controller: %s", err.Error())
	}

	var wg sync.WaitGroup
	// only add 1 to our wait group since our service should stop (and restart) as soon as one of the go routines terminates
	wg.Add(1)

	go func() {
		defer wg.Done()
		microservices.StartGRPCServer(gs, serviceConfig.EnableReflection)
	}()

	go func() {
		defer wg.Done()

		sessionServer := sessionservice.NewSessionServer(
			authnClient,
			authrClient,
			acClient,
			courseClient,
			progressClient,
			scenarioClient,
			scheduledEventClient,
			vmClaimClient,
			ss,
		)
		microservices.StartAPIServer(sessionServer)
	}()

	go func() {
		defer wg.Done()
		sessionController.RunSharded(stopCh, microservices.ScheduledEvent)
	}()

	hfInformerFactory.Start(stopCh)

	wg.Wait()
}
