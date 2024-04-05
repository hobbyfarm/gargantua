package main

import (
	"context"
	"sync"
	"time"

	"github.com/hobbyfarm/gargantua/v3/pkg/crd"
	"github.com/hobbyfarm/gargantua/v3/pkg/microservices"
	"github.com/hobbyfarm/gargantua/v3/pkg/signals"
	"github.com/hobbyfarm/gargantua/v3/pkg/util"

	"github.com/golang/glog"
	vmclaimservice "github.com/hobbyfarm/gargantua/services/vmclaimsvc/v3/internal"
	hfInformers "github.com/hobbyfarm/gargantua/v3/pkg/client/informers/externalversions"
	accesscodeProto "github.com/hobbyfarm/gargantua/v3/protos/accesscode"
	"github.com/hobbyfarm/gargantua/v3/protos/dbconfig"
	"github.com/hobbyfarm/gargantua/v3/protos/environment"
	"github.com/hobbyfarm/gargantua/v3/protos/progress"
	"github.com/hobbyfarm/gargantua/v3/protos/scheduledevent"
	"github.com/hobbyfarm/gargantua/v3/protos/session"
	"github.com/hobbyfarm/gargantua/v3/protos/vm"
	vmclaimProto "github.com/hobbyfarm/gargantua/v3/protos/vmclaim"
	"github.com/hobbyfarm/gargantua/v3/protos/vmtemplate"
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

	crd.InstallCrds(vmclaimservice.VMClaimCRDInstaller{}, cfg, "virtual machine claim")

	services := []microservices.MicroService{
		microservices.AccessCode,
		microservices.DBConfig,
		microservices.Environment,
		microservices.Progress,
		microservices.ScheduledEvent,
		microservices.Session,
		microservices.VM,
		microservices.VMTemplate,
	}
	connections := microservices.EstablishConnections(services, serviceConfig.ClientCert)
	for _, conn := range connections {
		defer conn.Close()
	}

	acClient := accesscodeProto.NewAccessCodeSvcClient(connections[microservices.AccessCode])
	dbcClient := dbconfig.NewDynamicBindConfigSvcClient(connections[microservices.DBConfig])
	envClient := environment.NewEnvironmentSvcClient(connections[microservices.Environment])
	eventClient := scheduledevent.NewScheduledEventSvcClient(connections[microservices.ScheduledEvent])
	progressClient := progress.NewProgressSvcClient(connections[microservices.Progress])
	sessionClient := session.NewSessionSvcClient(connections[microservices.Session])
	vmClient := vm.NewVMSvcClient(connections[microservices.VM])
	vmTemplateClient := vmtemplate.NewVMTemplateSvcClient(connections[microservices.VMTemplate])

	gs := microservices.CreateGRPCServer(serviceConfig.ServerCert.Clone())

	vs := vmclaimservice.NewGrpcVMClaimServer(hfClient, hfInformerFactory)
	vmclaimProto.RegisterVMClaimSvcServer(gs, vs)
	// @TODO: Push fully implemented controller.go for this service
	vmClaimController, err := vmclaimservice.NewVMClaimController(
		kubeClient,
		vs,
		hfInformerFactory,
		acClient,
		dbcClient,
		envClient,
		eventClient,
		progressClient,
		sessionClient,
		vmClient,
		vmTemplateClient,
		ctx,
	)
	if err != nil {
		glog.Fatalf("failed creating vm claim controller: %s", err.Error())
	}

	var wg sync.WaitGroup
	wg.Add(1)

	go func() {
		defer wg.Done()
		microservices.StartGRPCServer(gs, serviceConfig.EnableReflection)
	}()

	go func() {
		defer wg.Done()
		vmClaimController.RunSharded(stopCh, microservices.VMClaim)
	}()

	hfInformerFactory.Start(stopCh)

	wg.Wait()
}
