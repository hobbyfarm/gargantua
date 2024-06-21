package main

import (
	"sync"
	"time"

	"github.com/hobbyfarm/gargantua/v3/pkg/crd"
	"github.com/hobbyfarm/gargantua/v3/pkg/microservices"
	"github.com/hobbyfarm/gargantua/v3/pkg/signals"
	"github.com/hobbyfarm/gargantua/v3/pkg/util"

	vmtemplateservice "github.com/hobbyfarm/gargantua/services/vmtemplatesvc/v3/internal"
	hfInformers "github.com/hobbyfarm/gargantua/v3/pkg/client/informers/externalversions"
	authnpb "github.com/hobbyfarm/gargantua/v3/protos/authn"
	authrpb "github.com/hobbyfarm/gargantua/v3/protos/authr"
	scheduledeventpb "github.com/hobbyfarm/gargantua/v3/protos/scheduledevent"
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

	cfg, hfClient, _ := microservices.BuildClusterConfig(serviceConfig)

	namespace := util.GetReleaseNamespace()
	hfInformerFactory := hfInformers.NewSharedInformerFactoryWithOptions(hfClient, time.Second*30, hfInformers.WithNamespace(namespace))

	crd.InstallCrds(vmtemplateservice.VMTemplateCRDInstaller{}, cfg, "virtual machine template")

	services := []microservices.MicroService{
		microservices.AuthN,
		microservices.AuthR,
		microservices.ScheduledEvent,
		microservices.VM,
		microservices.VMClaim,
		microservices.VMSet,
	}
	connections := microservices.EstablishConnections(services, serviceConfig.ClientCert)
	for _, conn := range connections {
		defer conn.Close()
	}

	authnClient := authnpb.NewAuthNClient(connections[microservices.AuthN])
	authrClient := authrpb.NewAuthRClient(connections[microservices.AuthR])
	scheduledEventClient := scheduledeventpb.NewScheduledEventSvcClient(connections[microservices.ScheduledEvent])
	vmClient := vmpb.NewVMSvcClient(connections[microservices.VM])
	vmClaimClient := vmclaimpb.NewVMClaimSvcClient(connections[microservices.VMClaim])
	vmSetClient := vmsetpb.NewVMSetSvcClient(connections[microservices.VMSet])

	gs := microservices.CreateGRPCServer(serviceConfig.ServerCert.Clone())

	vs := vmtemplateservice.NewGrpcVMTemplateServer(hfClient, hfInformerFactory)

	vmtemplatepb.RegisterVMTemplateSvcServer(gs, vs)

	var wg sync.WaitGroup
	// only add 1 to our wait group since our service should stop (and restart) as soon as one of the go routines terminates
	wg.Add(1)

	go func() {
		defer wg.Done()
		microservices.StartGRPCServer(gs, serviceConfig.EnableReflection)
	}()

	go func() {
		defer wg.Done()

		vmTemplateServer := vmtemplateservice.NewVirtualMachineTemplateServer(
			authnClient,
			authrClient,
			scheduledEventClient,
			vmClient,
			vmClaimClient,
			vmSetClient,
			vs,
		)
		microservices.StartAPIServer(vmTemplateServer)
	}()

	hfInformerFactory.Start(stopCh)

	wg.Wait()
}
