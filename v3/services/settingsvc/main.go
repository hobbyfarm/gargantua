package main

import (
	"context"
	"sync"
	"time"

	"github.com/hobbyfarm/gargantua/v3/pkg/crd"
	"github.com/hobbyfarm/gargantua/v3/pkg/microservices"
	"github.com/hobbyfarm/gargantua/v3/pkg/signals"
	"github.com/hobbyfarm/gargantua/v3/pkg/util"

	"github.com/golang/glog"
	settingservice "github.com/hobbyfarm/gargantua/services/settingsvc/v3/internal"
	hfInformers "github.com/hobbyfarm/gargantua/v3/pkg/client/informers/externalversions"

	authnpb "github.com/hobbyfarm/gargantua/v3/protos/authn"
	authrpb "github.com/hobbyfarm/gargantua/v3/protos/authr"
	settingpb "github.com/hobbyfarm/gargantua/v3/protos/setting"
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

	crd.InstallCrdsWithServiceReference(settingservice.SettingCRDInstaller{}, cfg, "setting", serviceConfig.WebhookTLSCA)

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

	ctx := context.Background()

	err := settingservice.WatchSettings(ctx, hfClient, hfInformerFactory)
	if err != nil {
		glog.Info("watching settings failed: ", err)
	}
	gs := microservices.CreateGRPCServer(serviceConfig.ServerCert.Clone())
	ss := settingservice.NewGrpcSettingServer(hfClient, hfInformerFactory)
	settingpb.RegisterSettingSvcServer(gs, ss)
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
