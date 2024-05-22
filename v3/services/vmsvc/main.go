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
	terraformpb "github.com/hobbyfarm/gargantua/v3/protos/terraform"
	vmProto "github.com/hobbyfarm/gargantua/v3/protos/vm"
	vmclaimProto "github.com/hobbyfarm/gargantua/v3/protos/vmclaim"
	vmsetProto "github.com/hobbyfarm/gargantua/v3/protos/vmset"
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
		microservices.Terraform,
		microservices.VMClaim,
		microservices.VMSet,
	}
	connections := microservices.EstablishConnections(services, serviceConfig.ClientCert)
	for _, conn := range connections {
		defer conn.Close()
	}
	terraformClient := terraformpb.NewTerraformSvcClient(connections[microservices.Terraform])
	vmClaimClient := vmclaimProto.NewVMClaimSvcClient(connections[microservices.VMClaim])
	vmSetClient := vmsetProto.NewVMSetSvcClient(connections[microservices.VMSet])

	gs := microservices.CreateGRPCServer(serviceConfig.ServerCert.Clone())

	vs := vmservice.NewGrpcVMServer(hfClient, hfInformerFactory)
	vmProto.RegisterVMSvcServer(gs, vs)
	vmController, err := vmservice.NewVMController(
		kubeClient,
		vs,
		hfInformerFactory,
		terraformClient,
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
		vmController.RunSharded(stopCh, microservices.VM)
	}()

	hfInformerFactory.Start(stopCh)

	wg.Wait()
}
