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
	"github.com/hobbyfarm/gargantua/v3/protos/environment"
	"github.com/hobbyfarm/gargantua/v3/protos/vm"
	vmsetProto "github.com/hobbyfarm/gargantua/v3/protos/vmset"
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

	crd.InstallCrds(vmsetservice.VMSetCRDInstaller{}, cfg, "virtual machine set")

	services := []microservices.MicroService{
		microservices.Environment,
		microservices.VM,
		microservices.VMTemplate,
	}
	connections := microservices.EstablishConnections(services, serviceConfig.ClientCert)
	for _, conn := range connections {
		defer conn.Close()
	}

	envClient := environment.NewEnvironmentSvcClient(connections[microservices.Environment])
	vmClient := vm.NewVMSvcClient(connections[microservices.VM])
	vmTemplateClient := vmtemplate.NewVMTemplateSvcClient(connections[microservices.VMTemplate])

	vmSetWorkqueue := workqueue.NewDelayingQueueWithConfig(workqueue.DelayingQueueConfig{Name: "vmclaim-controller"})

	gs := microservices.CreateGRPCServer(serviceConfig.ServerCert.Clone())

	vs := vmsetservice.NewGrpcVMSetServer(hfClient, hfInformerFactory, vmSetWorkqueue)
	vmsetProto.RegisterVMSetSvcServer(gs, vs)
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
	wg.Add(1)

	go func() {
		defer wg.Done()
		microservices.StartGRPCServer(gs, serviceConfig.EnableReflection)
	}()

	go func() {
		defer wg.Done()
		vmSetController.RunSharded(stopCh, microservices.VMSet)
	}()

	hfInformerFactory.Start(stopCh)

	wg.Wait()
}
