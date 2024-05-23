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

	eventservice "github.com/hobbyfarm/gargantua/services/scheduledeventsvc/v3/internal"
	hfInformers "github.com/hobbyfarm/gargantua/v3/pkg/client/informers/externalversions"
	accesscodepb "github.com/hobbyfarm/gargantua/v3/protos/accesscode"
	dbconfigpb "github.com/hobbyfarm/gargantua/v3/protos/dbconfig"
	environmentpb "github.com/hobbyfarm/gargantua/v3/protos/environment"
	progresspb "github.com/hobbyfarm/gargantua/v3/protos/progress"
	scheduledeventpb "github.com/hobbyfarm/gargantua/v3/protos/scheduledevent"
	sessionpb "github.com/hobbyfarm/gargantua/v3/protos/session"
	settingpb "github.com/hobbyfarm/gargantua/v3/protos/setting"
	vmsetpb "github.com/hobbyfarm/gargantua/v3/protos/vmset"
	vmtemplatepb "github.com/hobbyfarm/gargantua/v3/protos/vmtemplate"
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

	crd.InstallCrds(eventservice.ScheduledEventCRDInstaller{}, cfg, "scheduled event")

	services := []microservices.MicroService{
		microservices.AccessCode,
		microservices.DBConfig,
		microservices.Environment,
		microservices.Progress,
		microservices.Session,
		microservices.VMSet,
		microservices.VMTemplate,
		microservices.Setting,
	}
	connections := microservices.EstablishConnections(services, serviceConfig.ClientCert)
	for _, conn := range connections {
		defer conn.Close()
	}

	acClient := accesscodepb.NewAccessCodeSvcClient(connections[microservices.AccessCode])
	dbcClient := dbconfigpb.NewDynamicBindConfigSvcClient(connections[microservices.DBConfig])
	envClient := environmentpb.NewEnvironmentSvcClient(connections[microservices.Environment])
	progressClient := progresspb.NewProgressSvcClient(connections[microservices.Progress])
	sessionClient := sessionpb.NewSessionSvcClient(connections[microservices.Session])
	vmSetClient := vmsetpb.NewVMSetSvcClient(connections[microservices.VMSet])
	vmTemplateClient := vmtemplatepb.NewVMTemplateSvcClient(connections[microservices.VMTemplate])
	settingClient := settingpb.NewSettingSvcClient(connections[microservices.Setting])

	gs := microservices.CreateGRPCServer(serviceConfig.ServerCert.Clone())

	ss := eventservice.NewGrpcScheduledEventServer(hfClient, hfInformerFactory)
	scheduledeventpb.RegisterScheduledEventSvcServer(gs, ss)
	seController, err := eventservice.NewScheduledEventController(
		kubeClient,
		ss,
		hfInformerFactory,
		acClient,
		dbcClient,
		envClient,
		progressClient,
		sessionClient,
		vmSetClient,
		vmTemplateClient,
		settingClient,
		ctx,
	)
	if err != nil {
		glog.Fatalf("failed creating scheduled event controller: %s", err.Error())
	}

	var wg sync.WaitGroup
	wg.Add(1)

	go func() {
		defer wg.Done()
		microservices.StartGRPCServer(gs, serviceConfig.EnableReflection)
	}()

	go func() {
		defer wg.Done()
		seController.RunSharded(stopCh, microservices.ScheduledEvent)
	}()

	hfInformerFactory.Start(stopCh)

	wg.Wait()
}
