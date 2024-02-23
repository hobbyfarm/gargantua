package main

import (
	"context"
	"sync"
	"time"

	"github.com/ebauman/crder"
	"github.com/hobbyfarm/gargantua/v3/pkg/microservices"
	"github.com/hobbyfarm/gargantua/v3/pkg/signals"
	"github.com/hobbyfarm/gargantua/v3/pkg/util"

	"github.com/golang/glog"
	vmclaimservice "github.com/hobbyfarm/gargantua/services/vmclaimsvc/v3/internal"
	hfInformers "github.com/hobbyfarm/gargantua/v3/pkg/client/informers/externalversions"
	accesscodeProto "github.com/hobbyfarm/gargantua/v3/protos/accesscode"
	vmclaimProto "github.com/hobbyfarm/gargantua/v3/protos/vmclaim"
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

	crds := vmclaimservice.GenerateVMClaimCRD()
	glog.Info("installing/updating vm claim CRDs")
	err := crder.InstallUpdateCRDs(cfg, crds...)
	if err != nil {
		glog.Fatalf("failed installing/updating vm claim CRDs: %s", err.Error())
	}
	glog.Info("finished installing/updating vm claim CRDs")

	services := []microservices.MicroService{
		microservices.AccessCode,
	}
	connections := microservices.EstablishConnections(services, serviceConfig.ClientCert)
	for _, conn := range connections {
		defer conn.Close()
	}

	acClient := accesscodeProto.NewAccessCodeSvcClient(connections[microservices.AccessCode])

	gs := microservices.CreateGRPCServer(serviceConfig.ServerCert.Clone())

	ds := vmclaimservice.NewGrpcVMClaimServer(hfClient, hfInformerFactory)
	vmclaimProto.RegisterVMClaimSvcServer(gs, ds)
	// @TODO: Push fully implemented controller.go for this service
	vmClaimController, err := vmclaimservice.NewVMClaimController(kubeClient, hfClient, hfInformerFactory, &acClient, ctx)
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
