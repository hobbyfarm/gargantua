package main

import (
	"context"
	hfInformers "github.com/hobbyfarm/gargantua/v3/pkg/client/informers/externalversions"
	"github.com/hobbyfarm/gargantua/v3/pkg/crd"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/dynamic/dynamicinformer"
	"sync"
	"time"

	"github.com/hobbyfarm/gargantua/v3/pkg/microservices"
	"github.com/hobbyfarm/gargantua/v3/pkg/signals"
	"github.com/hobbyfarm/gargantua/v3/pkg/util"

	costservice "github.com/hobbyfarm/gargantua/services/costsvc/v3/internal"
	authnpb "github.com/hobbyfarm/gargantua/v3/protos/authn"
	authrpb "github.com/hobbyfarm/gargantua/v3/protos/authr"
	costpb "github.com/hobbyfarm/gargantua/v3/protos/cost"
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

	cfg, hfClient, _ := microservices.BuildClusterConfig(serviceConfig)
	dynamicClient := microservices.BuildDynamicClient(cfg)

	namespace := util.GetReleaseNamespace()
	hfInformerFactory := hfInformers.NewSharedInformerFactoryWithOptions(hfClient, time.Second*30, hfInformers.WithNamespace(namespace))
	dynamicInformerFactory := dynamicinformer.NewFilteredDynamicSharedInformerFactory(dynamicClient, time.Second*30, namespace, func(options *metav1.ListOptions) {
		// only resources with label cost-group
		options.LabelSelector = costservice.LabelCostGroup
	})

	crd.InstallCrds(costservice.CostCRDInstaller{}, cfg, "cost")

	services := []microservices.MicroService{
		microservices.AuthN,
		microservices.AuthR,
	}
	connections := microservices.EstablishConnections(services, serviceConfig.ClientCert)
	for _, conn := range connections {
		defer conn.Close()
	}
	authnClient := authnpb.NewAuthNClient(connections[microservices.AuthN])
	authrClient := authrpb.NewAuthRClient(connections[microservices.AuthR])

	gs := microservices.CreateGRPCServer(serviceConfig.ServerCert.Clone())

	cs := costservice.NewGrpcCostServer(hfClient, hfInformerFactory)
	costpb.RegisterCostSvcServer(gs, cs)

	costservice.NewCostController(cs, dynamicInformerFactory, ctx, costservice.ParseConfig()...)

	var wg sync.WaitGroup
	// only add 1 to our wait group since our service should stop (and restart) as soon as one of the go routines terminates
	wg.Add(1)

	go func() {
		defer wg.Done()
		microservices.StartGRPCServer(gs, serviceConfig.EnableReflection)
	}()

	go func() {
		defer wg.Done()

		costServer := costservice.NewCostServer(
			authnClient,
			authrClient,
			cs,
		)
		microservices.StartAPIServer(costServer)
	}()

	hfInformerFactory.Start(stopCh)
	dynamicInformerFactory.Start(stopCh)

	wg.Wait()
}
