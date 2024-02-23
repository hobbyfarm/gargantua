package main

import (
	"sync"
	"time"

	"github.com/ebauman/crder"
	"github.com/hobbyfarm/gargantua/v3/pkg/microservices"
	"github.com/hobbyfarm/gargantua/v3/pkg/signals"
	"github.com/hobbyfarm/gargantua/v3/pkg/util"

	"github.com/golang/glog"
	accesscodeservice "github.com/hobbyfarm/gargantua/services/accesscodesvc/v3/internal"
	hfInformers "github.com/hobbyfarm/gargantua/v3/pkg/client/informers/externalversions"
	accessCodeProto "github.com/hobbyfarm/gargantua/v3/protos/accesscode"
	"github.com/hobbyfarm/gargantua/v3/protos/user"
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

	crds := accesscodeservice.GenerateAccessCodeCRD()
	glog.Info("installing/updating access code CRDs")
	err := crder.InstallUpdateCRDs(cfg, crds...)
	if err != nil {
		glog.Fatalf("failed installing/updating access code CRDs: %s", err.Error())
	}
	glog.Info("finished installing/updating access code CRDs")

	services := []microservices.MicroService{
		microservices.User,
	}
	connections := microservices.EstablishConnections(services, serviceConfig.ClientCert)
	for _, conn := range connections {
		defer conn.Close()
	}

	userClient := user.NewUserSvcClient(connections[microservices.User])

	gs := microservices.CreateGRPCServer(serviceConfig.ServerCert.Clone())
	as := accesscodeservice.NewGrpcAccessCodeServer(hfClient, hfInformerFactory, userClient)
	accessCodeProto.RegisterAccessCodeSvcServer(gs, as)

	var wg sync.WaitGroup
	wg.Add(1)

	go func() {
		defer wg.Done()
		microservices.StartGRPCServer(gs, serviceConfig.EnableReflection)
	}()

	hfInformerFactory.Start(stopCh)

	wg.Wait()
}
