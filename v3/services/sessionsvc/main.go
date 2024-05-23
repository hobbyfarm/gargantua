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
	progresspb "github.com/hobbyfarm/gargantua/v3/protos/progress"
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
		microservices.Progress,
		microservices.VM,
		microservices.VMClaim,
	}
	connections := microservices.EstablishConnections(services, serviceConfig.ClientCert)
	for _, conn := range connections {
		defer conn.Close()
	}

	progressClient := progresspb.NewProgressSvcClient(connections[microservices.Progress])
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
	wg.Add(1)

	go func() {
		defer wg.Done()
		microservices.StartGRPCServer(gs, serviceConfig.EnableReflection)
	}()

	go func() {
		defer wg.Done()
		sessionController.RunSharded(stopCh, microservices.ScheduledEvent)
	}()

	hfInformerFactory.Start(stopCh)

	wg.Wait()
}
