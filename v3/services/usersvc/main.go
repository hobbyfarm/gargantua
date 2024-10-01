package main

import (
	"sync"
	"time"

	"github.com/hobbyfarm/gargantua/v3/pkg/crd"
	"github.com/hobbyfarm/gargantua/v3/pkg/microservices"
	"github.com/hobbyfarm/gargantua/v3/pkg/signals"
	"github.com/hobbyfarm/gargantua/v3/pkg/util"

	"github.com/golang/glog"
	userservice "github.com/hobbyfarm/gargantua/services/usersvc/v3/internal"
	userservicecontroller "github.com/hobbyfarm/gargantua/services/usersvc/v3/internal/controllers"
	hfInformers "github.com/hobbyfarm/gargantua/v3/pkg/client/informers/externalversions"

	authnpb "github.com/hobbyfarm/gargantua/v3/protos/authn"
	authrpb "github.com/hobbyfarm/gargantua/v3/protos/authr"
	rbacpb "github.com/hobbyfarm/gargantua/v3/protos/rbac"
	sessionpb "github.com/hobbyfarm/gargantua/v3/protos/session"
	userpb "github.com/hobbyfarm/gargantua/v3/protos/user"
)

var (
	serviceConfig *microservices.ServiceConfig
)

func init() {
	serviceConfig = microservices.BuildServiceConfig()
}

func main() {
	cfg, hfClient, kubeClient := microservices.BuildClusterConfig(serviceConfig)

	namespace := util.GetReleaseNamespace()
	hfInformerFactory := hfInformers.NewSharedInformerFactoryWithOptions(hfClient, time.Second*30, hfInformers.WithNamespace(namespace))

	crd.InstallCrdsWithServiceReference(userservice.UserCRDInstaller{}, cfg, "user", serviceConfig.WebhookTLSCA)

	services := []microservices.MicroService{
		microservices.Rbac,
		microservices.AuthN,
		microservices.AuthR,
		microservices.Session,
	}
	connections := microservices.EstablishConnections(services, serviceConfig.ClientCert)
	for _, conn := range connections {
		defer conn.Close()
	}

	rbacClient := rbacpb.NewRbacSvcClient(connections[microservices.Rbac])
	authnClient := authnpb.NewAuthNClient(connections[microservices.AuthN])
	authrClient := authrpb.NewAuthRClient(connections[microservices.AuthR])
	sessionClient := sessionpb.NewSessionSvcClient(connections[microservices.Session])

	gs := microservices.CreateGRPCServer(serviceConfig.ServerCert)
	us, err := userservice.NewGrpcUserServer(hfClient, hfInformerFactory, sessionClient, rbacClient)

	if err != nil {
		glog.Fatalf("starting grpc user server failed: %v", err)
	}

	userpb.RegisterUserSvcServer(gs, us)

	passwordResetTokenController, err := userservicecontroller.NewPasswordResetTokenController(hfInformerFactory, kubeClient, hfClient, ctx)
	if err != nil {
		glog.Fatalf("creating passwordResetTokenController failed: %v", err)
	}
	passwordResetTokenController.SetWorkerThreadCount(microservices.GetWorkerThreadCount())

	var wg sync.WaitGroup
	// only add 1 to our wait group since our service should stop (and restart) as soon as one of the go routines terminates
	wg.Add(1)

	go func() {
		defer wg.Done()
		microservices.StartGRPCServer(gs, serviceConfig.EnableReflection)
	}()

	go func() {
		defer wg.Done()

		userServer := userservice.NewUserServer(authnClient, authrClient, rbacClient, us)
		microservices.StartAPIServer(userServer)
	}()

	go func() {
		defer wg.Done()
		glog.Info("Starting controllers")
		stopControllersCh := make(chan struct{})
		err := passwordResetTokenController.RunSharded(stopControllersCh, os.Getenv("STATEFULSET_NAME"))
		if err != nil {
			glog.Errorf("Error starting up the controllers: %v", err)
		}
	}()

	stopInformerFactoryCh := signals.SetupSignalHandler()
	hfInformerFactory.Start(stopInformerFactoryCh)

	wg.Wait()
}
