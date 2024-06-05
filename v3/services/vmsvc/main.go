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

	vmservice "github.com/hobbyfarm/gargantua/services/vmsvc/v3/internal"
	hfInformers "github.com/hobbyfarm/gargantua/v3/pkg/client/informers/externalversions"
	authnpb "github.com/hobbyfarm/gargantua/v3/protos/authn"
	authrpb "github.com/hobbyfarm/gargantua/v3/protos/authr"
	environmentpb "github.com/hobbyfarm/gargantua/v3/protos/environment"
	terraformpb "github.com/hobbyfarm/gargantua/v3/protos/terraform"
	vmpb "github.com/hobbyfarm/gargantua/v3/protos/vm"
	vmclaimpb "github.com/hobbyfarm/gargantua/v3/protos/vmclaim"
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

	crd.InstallCrds(vmservice.VmCRDInstaller{}, cfg, "virtual machine")

	services := []microservices.MicroService{
		microservices.AuthN,
		microservices.AuthR,
		microservices.Environment,
		microservices.Terraform,
		microservices.VMClaim,
		microservices.VMSet,
		microservices.VMTemplate,
	}
	connections := microservices.EstablishConnections(services, serviceConfig.ClientCert)
	for _, conn := range connections {
		defer conn.Close()
	}
	authnClient := authnpb.NewAuthNClient(connections[microservices.AuthN])
	authrClient := authrpb.NewAuthRClient(connections[microservices.AuthR])
	environmentClient := environmentpb.NewEnvironmentSvcClient(connections[microservices.Environment])
	terraformClient := terraformpb.NewTerraformSvcClient(connections[microservices.Terraform])
	vmClaimClient := vmclaimpb.NewVMClaimSvcClient(connections[microservices.VMClaim])
	vmSetClient := vmsetpb.NewVMSetSvcClient(connections[microservices.VMSet])
	vmTemplateClient := vmtemplatepb.NewVMTemplateSvcClient(connections[microservices.VMTemplate])

	gs := microservices.CreateGRPCServer(serviceConfig.ServerCert.Clone())

	vs := vmservice.NewGrpcVMServer(hfClient, hfInformerFactory)
	vmpb.RegisterVMSvcServer(gs, vs)
	vmController, err := vmservice.NewVMController(
		kubeClient,
		vs,
		hfInformerFactory,
		environmentClient,
		terraformClient,
		vmClaimClient,
		vmSetClient,
		vmTemplateClient,
		ctx,
	)
	if err != nil {
		glog.Fatalf("failed creating vm controller: %s", err.Error())
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

		vmServer := vmservice.NewVMServer(
			authnClient,
			authrClient,
			vmTemplateClient,
			vs,
		)
		microservices.StartAPIServer(vmServer)
	}()

	go func() {
		defer wg.Done()
		vmController.RunSharded(stopCh, microservices.VM)
	}()

	hfInformerFactory.Start(stopCh)

	wg.Wait()
}
