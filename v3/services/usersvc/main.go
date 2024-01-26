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
	"golang.org/x/sync/errgroup"

	"github.com/golang/glog"
	userservice "github.com/hobbyfarm/gargantua/services/usersvc/v3/internal"
	userservicecontroller "github.com/hobbyfarm/gargantua/services/usersvc/v3/internal/controllers"
	hfInformers "github.com/hobbyfarm/gargantua/v3/pkg/client/informers/externalversions"

	"github.com/hobbyfarm/gargantua/v3/protos/authn"
	"github.com/hobbyfarm/gargantua/v3/protos/authr"
	"github.com/hobbyfarm/gargantua/v3/protos/rbac"
	"github.com/hobbyfarm/gargantua/v3/protos/user"
)

var (
	serviceConfig *microservices.ServiceConfig
)

func init() {
	serviceConfig = microservices.BuildServiceConfig()
}

func main() {
	cfg, hfClient := microservices.BuildClusterConfig(serviceConfig)

	namespace := util.GetReleaseNamespace()
	hfInformerFactory := hfInformers.NewSharedInformerFactoryWithOptions(hfClient, time.Second*30, hfInformers.WithNamespace(namespace))

	ca, err := os.ReadFile(serviceConfig.WebhookTLSCA)
	if err != nil {
		glog.Fatalf("error reading ca certificate: %s", err.Error())
	}

	crds := userservice.GenerateUserCRD(string(ca), crd.ServiceReference{
		Namespace: util.GetReleaseNamespace(),
		Name:      "hobbyfarm-webhook",
	})

	glog.Info("installing/updating user CRD")
	err = crder.InstallUpdateCRDs(cfg, crds...)
	if err != nil {
		glog.Fatalf("failed installing/updating user crd: %s", err.Error())
	}
	glog.Info("finished installing/updating user CRD")

	services := []microservices.MicroService{
		microservices.Rbac,
		microservices.AuthN,
		microservices.AuthR,
	}
	connections := microservices.EstablishConnections(services, serviceConfig.ClientCert)
	for _, conn := range connections {
		defer conn.Close()
	}

	rbacClient := rbac.NewRbacSvcClient(connections[microservices.Rbac])
	authnClient := authn.NewAuthNClient(connections[microservices.AuthN])
	authrClient := authr.NewAuthRClient(connections[microservices.AuthR])

	gs := microservices.CreateGRPCServer(serviceConfig.ServerCert)
	ctx := context.Background()
	us, err := userservice.NewGrpcUserServer(hfClient, hfInformerFactory, ctx)

	if err != nil {
		glog.Fatalf("starting grpc user server failed: %v", err)
	}

	user.RegisterUserSvcServer(gs, us)

	var wg sync.WaitGroup
	wg.Add(1)

	go func() {
		defer wg.Done()
		microservices.StartGRPCServer(gs, serviceConfig.EnableReflection)
	}()

	go func() {
		defer wg.Done()

		userServer, err := userservice.NewUserServer(authnClient, authrClient, rbacClient, us)
		if err != nil {
			glog.Fatalf("Error creating userserver: %v", err)
		}
		microservices.StartAPIServer(userServer)
	}()

	stopControllersCh := make(chan struct{}, 1)
	g, gctx := errgroup.WithContext(ctx)
	passwordResetTokenController, err := userservicecontroller.NewPasswordResetTokenController(us, hfInformerFactory, gctx)
	if err != nil {
		glog.Fatalf("starting grpc user server failed: %v", err)
	}

	microservices.ElectLeaderOrDie(microservices.User, cfg, gctx, stopControllersCh, func(c context.Context) {
		glog.Info("Started being the leader. Starting controllers")
		g.Go(func() error {
			return passwordResetTokenController.Run(stopControllersCh)
		})
		if err != nil {
			glog.Fatal(err)
		}
	})

	stopInformerFactoryCh := signals.SetupSignalHandler()
	hfInformerFactory.Start(stopInformerFactoryCh)
	wg.Wait()
}