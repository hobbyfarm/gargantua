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

	terraformservice "github.com/hobbyfarm/gargantua/services/terraformsvc/v3/internal"
	hfInformers "github.com/hobbyfarm/gargantua/v3/pkg/client/informers/externalversions"
	environmentpb "github.com/hobbyfarm/gargantua/v3/protos/environment"
	terraformpb "github.com/hobbyfarm/gargantua/v3/protos/terraform"
	vmpb "github.com/hobbyfarm/gargantua/v3/protos/vm"
	vmclaimpb "github.com/hobbyfarm/gargantua/v3/protos/vmclaim"
	vmsetpb "github.com/hobbyfarm/gargantua/v3/protos/vmset"
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

	crd.InstallCrds(terraformservice.TerraformCRDInstaller{}, cfg, "terraform")

	services := []microservices.MicroService{
		microservices.Environment,
		microservices.VMClaim,
		microservices.VM,
		microservices.VMSet,
		microservices.VMTemplate,
	}
	connections := microservices.EstablishConnections(services, serviceConfig.ClientCert)
	for _, conn := range connections {
		defer conn.Close()
	}
	environmentClient := environmentpb.NewEnvironmentSvcClient(connections[microservices.Environment])
	vmClaimClient := vmclaimpb.NewVMClaimSvcClient(connections[microservices.VMClaim])
	vmClient := vmpb.NewVMSvcClient(connections[microservices.VM])
	vmSetClient := vmsetpb.NewVMSetSvcClient(connections[microservices.VMSet])

	gs := microservices.CreateGRPCServer(serviceConfig.ServerCert.Clone())
	ts := terraformservice.NewGrpcTerraformServer(hfClient, hfInformerFactory)
	terraformpb.RegisterTerraformSvcServer(gs, ts)

	vmController, err := terraformservice.NewVMController(
		hfClient,
		kubeClient,
		vmClient,
		hfInformerFactory,
		environmentClient,
		ts,
		vmClaimClient,
		vmSetClient,
		ctx,
	)

	if err != nil {
		glog.Fatalf("failed creating vm controller: %s", err.Error())
	}

	var wg sync.WaitGroup
	wg.Add(1)

	go func() {
		defer wg.Done()
		microservices.StartGRPCServer(gs, serviceConfig.EnableReflection)
	}()

	go func() {
		defer wg.Done()
		vmController.RunSharded(stopCh, microservices.Terraform)
	}()

	hfInformerFactory.Start(stopCh)

	wg.Wait()
}
