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
	"k8s.io/client-go/util/workqueue"

	vmsetservice "github.com/hobbyfarm/gargantua/services/vmsetsvc/v3/internal"
	hfInformers "github.com/hobbyfarm/gargantua/v3/pkg/client/informers/externalversions"
	authnpb "github.com/hobbyfarm/gargantua/v3/protos/authn"
	authrpb "github.com/hobbyfarm/gargantua/v3/protos/authr"
	environmentpb "github.com/hobbyfarm/gargantua/v3/protos/environment"
	vmpb "github.com/hobbyfarm/gargantua/v3/protos/vm"
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

	crd.InstallCrds(vmsetservice.VMSetCRDInstaller{}, cfg, "virtual machine set")

	services := []microservices.MicroService{
		microservices.AuthN,
		microservices.AuthR,
		microservices.Environment,
		microservices.VM,
		microservices.VMTemplate,
	}
	connections := microservices.EstablishConnections(services, serviceConfig.ClientCert)
	for _, conn := range connections {
		defer conn.Close()
	}

	authnClient := authnpb.NewAuthNClient(connections[microservices.AuthN])
	authrClient := authrpb.NewAuthRClient(connections[microservices.AuthR])
	envClient := environmentpb.NewEnvironmentSvcClient(connections[microservices.Environment])
	vmClient := vmpb.NewVMSvcClient(connections[microservices.VM])
	vmTemplateClient := vmtemplatepb.NewVMTemplateSvcClient(connections[microservices.VMTemplate])

	vmSetWorkqueue := workqueue.NewDelayingQueueWithConfig(workqueue.DelayingQueueConfig{Name: "vmclaim-controller"})

	gs := microservices.CreateGRPCServer(serviceConfig.ServerCert.Clone())

	vs := vmsetservice.NewGrpcVMSetServer(hfClient, hfInformerFactory, vmSetWorkqueue)
	vmsetpb.RegisterVMSetSvcServer(gs, vs)
	vmSetController, err := vmsetservice.NewVMSetController(
		kubeClient,
		vs,
		hfInformerFactory,
		envClient,
		vmClient,
		vmTemplateClient,
		ctx,
	)
	if err != nil {
		glog.Fatalf("failed creating vm set controller: %s", err.Error())
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

		vmSetServer := vmsetservice.NewVMSetServer(
			authnClient,
			authrClient,
			vs,
		)
		microservices.StartAPIServer(vmSetServer)
	}()

	go func() {
		defer wg.Done()
		vmSetController.RunSharded(stopCh, microservices.VMSet)
	}()

	hfInformerFactory.Start(stopCh)

	wg.Wait()
}
