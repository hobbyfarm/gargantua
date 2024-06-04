package main

import (
	"context"
	"sync"
	"time"

	"github.com/hobbyfarm/gargantua/v3/pkg/crd"
	"github.com/hobbyfarm/gargantua/v3/pkg/microservices"
	"github.com/hobbyfarm/gargantua/v3/pkg/signals"
	"github.com/hobbyfarm/gargantua/v3/pkg/util"
	"k8s.io/client-go/util/workqueue"

	"github.com/golang/glog"
	vmclaimservice "github.com/hobbyfarm/gargantua/services/vmclaimsvc/v3/internal"
	hfInformers "github.com/hobbyfarm/gargantua/v3/pkg/client/informers/externalversions"
	accesscodepb "github.com/hobbyfarm/gargantua/v3/protos/accesscode"
	authnpb "github.com/hobbyfarm/gargantua/v3/protos/authn"
	authrpb "github.com/hobbyfarm/gargantua/v3/protos/authr"
	dbconfigpb "github.com/hobbyfarm/gargantua/v3/protos/dbconfig"
	environmentpb "github.com/hobbyfarm/gargantua/v3/protos/environment"
	progresspb "github.com/hobbyfarm/gargantua/v3/protos/progress"
	scheduledeventpb "github.com/hobbyfarm/gargantua/v3/protos/scheduledevent"
	sessionpb "github.com/hobbyfarm/gargantua/v3/protos/session"
	vmpb "github.com/hobbyfarm/gargantua/v3/protos/vm"
	vmclaimpb "github.com/hobbyfarm/gargantua/v3/protos/vmclaim"
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

	crd.InstallCrds(vmclaimservice.VMClaimCRDInstaller{}, cfg, "virtual machine claim")

	services := []microservices.MicroService{
		microservices.AuthN,
		microservices.AuthR,
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

	authnClient := authnpb.NewAuthNClient(connections[microservices.AuthN])
	authrClient := authrpb.NewAuthRClient(connections[microservices.AuthR])
	acClient := accesscodepb.NewAccessCodeSvcClient(connections[microservices.AccessCode])
	dbcClient := dbconfigpb.NewDynamicBindConfigSvcClient(connections[microservices.DBConfig])
	envClient := environmentpb.NewEnvironmentSvcClient(connections[microservices.Environment])
	eventClient := scheduledeventpb.NewScheduledEventSvcClient(connections[microservices.ScheduledEvent])
	progressClient := progresspb.NewProgressSvcClient(connections[microservices.Progress])
	sessionClient := sessionpb.NewSessionSvcClient(connections[microservices.Session])
	vmClient := vmpb.NewVMSvcClient(connections[microservices.VM])
	vmTemplateClient := vmtemplatepb.NewVMTemplateSvcClient(connections[microservices.VMTemplate])

	vmClaimWorkqueue := workqueue.NewDelayingQueueWithConfig(workqueue.DelayingQueueConfig{Name: "vmclaim-controller"})

	gs := microservices.CreateGRPCServer(serviceConfig.ServerCert.Clone())

	vs := vmclaimservice.NewGrpcVMClaimServer(hfClient, hfInformerFactory, vmClaimWorkqueue)
	vmclaimpb.RegisterVMClaimSvcServer(gs, vs)
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

	wg.Add(1)
	go func() {
		defer wg.Done()

		vmClaimServer := vmclaimservice.NewVMClaimServer(
			authnClient,
			authrClient,
			vs,
		)
		microservices.StartAPIServer(vmClaimServer)
	}()

	go func() {
		defer wg.Done()
		vmClaimController.RunSharded(stopCh, microservices.VMClaim)
	}()

	hfInformerFactory.Start(stopCh)

	wg.Wait()
}
