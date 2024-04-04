package main

import (
	"context"
	"os"
	"sync"
	"time"

	"github.com/ebauman/crder"
	"github.com/hobbyfarm/gargantua/v3/pkg/crd"
	"github.com/hobbyfarm/gargantua/v3/pkg/microservices"
	"github.com/hobbyfarm/gargantua/v3/pkg/signals"
	"github.com/hobbyfarm/gargantua/v3/pkg/util"

	"github.com/golang/glog"
	settingservice "github.com/hobbyfarm/gargantua/services/settingsvc/v3/internal"
	hfInformers "github.com/hobbyfarm/gargantua/v3/pkg/client/informers/externalversions"

	"github.com/hobbyfarm/gargantua/v3/protos/authn"
	"github.com/hobbyfarm/gargantua/v3/protos/authr"
	settingProto "github.com/hobbyfarm/gargantua/v3/protos/setting"
)

var (
	serviceConfig *microservices.ServiceConfig
)

func init() {
	serviceConfig = microservices.BuildServiceConfig()
}

func main() {
	cfg, hfClient, _ := microservices.BuildClusterConfig(serviceConfig)

	namespace := util.GetReleaseNamespace()

	hfInformerFactory := hfInformers.NewSharedInformerFactoryWithOptions(hfClient, time.Second*30, hfInformers.WithNamespace(namespace))

	ca, err := os.ReadFile(serviceConfig.WebhookTLSCA)
	if err != nil {
		glog.Fatalf("error reading ca certificate: %s", err.Error())
	}

	crds := settingservice.GenerateSettingCRD(string(ca), crd.ServiceReference{
		Namespace: util.GetReleaseNamespace(),
		Name:      "hobbyfarm-webhook",
	})

	glog.Info("installing/updating setting CRD")
	err = crder.InstallUpdateCRDs(cfg, crds...)
	if err != nil {
		glog.Fatalf("failed installing/updating setting crd: %s", err.Error())
	}
	glog.Info("finished installing/updating setting CRD")

	services := []microservices.MicroService{
		microservices.AuthN,
		microservices.AuthR,
	}
	connections := microservices.EstablishConnections(services, serviceConfig.ClientCert)
	for _, conn := range connections {
		defer conn.Close()
	}

	authnClient := authn.NewAuthNClient(connections[microservices.AuthN])
	authrClient := authr.NewAuthRClient(connections[microservices.AuthR])

	ctx := context.Background()

	err = settingservice.WatchSettings(ctx, hfClient, hfInformerFactory)
	if err != nil {
		glog.Info("watching settings failed: ", err)
	}
	gs := microservices.CreateGRPCServer(serviceConfig.ServerCert.Clone())
	ss := settingservice.NewGrpcSettingServer(hfClient, ctx)
	settingProto.RegisterSettingSvcServer(gs, ss)
	settingservice.Preinstall(ctx, ss)

	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		defer wg.Done()
		microservices.StartGRPCServer(gs, serviceConfig.EnableReflection)
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		settingServer, err := settingservice.NewSettingServer(authnClient, authrClient, ss)
		if err != nil {
			glog.Fatal(err)
		}
		microservices.StartAPIServer(settingServer)
	}()

	stopCh := signals.SetupSignalHandler()
	hfInformerFactory.Start(stopCh)
	wg.Wait()
}
