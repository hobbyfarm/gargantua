package main

import (
	"sync"
	"time"

	"github.com/hobbyfarm/gargantua/v3/pkg/crd"
	"github.com/hobbyfarm/gargantua/v3/pkg/microservices"
	"github.com/hobbyfarm/gargantua/v3/pkg/signals"
	"github.com/hobbyfarm/gargantua/v3/pkg/util"

	vmservice "github.com/hobbyfarm/gargantua/services/vmsvc/v3/internal"
	hfInformers "github.com/hobbyfarm/gargantua/v3/pkg/client/informers/externalversions"
	authnpb "github.com/hobbyfarm/gargantua/v3/protos/authn"
	authrpb "github.com/hobbyfarm/gargantua/v3/protos/authr"
	environmentpb "github.com/hobbyfarm/gargantua/v3/protos/environment"
	vmpb "github.com/hobbyfarm/gargantua/v3/protos/vm"
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

	cfg, hfClient, kubeClient := microservices.BuildClusterConfig(serviceConfig)

	namespace := util.GetReleaseNamespace()
	hfInformerFactory := hfInformers.NewSharedInformerFactoryWithOptions(hfClient, time.Second*30, hfInformers.WithNamespace(namespace))

	crd.InstallCrds(vmservice.VmCRDInstaller{}, cfg, "virtual machine")

	services := []microservices.MicroService{
		microservices.AuthN,
		microservices.AuthR,
		microservices.VMTemplate,
		microservices.Environment,
	}
	connections := microservices.EstablishConnections(services, serviceConfig.ClientCert)
	for _, conn := range connections {
		defer conn.Close()
	}
	authnClient := authnpb.NewAuthNClient(connections[microservices.AuthN])
	authrClient := authrpb.NewAuthRClient(connections[microservices.AuthR])
	vmTemplateClient := vmtemplatepb.NewVMTemplateSvcClient(connections[microservices.VMTemplate])
	envClient := environmentpb.NewEnvironmentSvcClient(connections[microservices.Environment])

	gs := microservices.CreateGRPCServer(serviceConfig.ServerCert.Clone())

	vs := vmservice.NewGrpcVMServer(hfClient, hfInformerFactory, kubeClient, envClient, vmTemplateClient)
	vmpb.RegisterVMSvcServer(gs, vs)

	var wg sync.WaitGroup
	// only add 1 to our wait group since our service should stop (and restart) as soon as one of the go routines terminates
	wg.Add(1)

	go func() {
		defer wg.Done()
		microservices.StartGRPCServer(gs, serviceConfig.EnableReflection)
	}()

	go func() {
		defer wg.Done()

		vmServer := vmservice.NewVMServer(
			authnClient,
			authrClient,
			vmTemplateClient,
			vs,
		)
		microservices.StartAPIServer(vmServer)
	}()

	hfInformerFactory.Start(stopCh)

	wg.Wait()
}
