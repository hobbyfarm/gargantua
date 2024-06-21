package main

import (
	"context"
	"flag"
	"sync"
	"time"

	"github.com/golang/glog"
	rbacservice "github.com/hobbyfarm/gargantua/services/rbacsvc/v3/internal"
	rbacinstaller "github.com/hobbyfarm/gargantua/services/rbacsvc/v3/internal/rbac"
	"github.com/hobbyfarm/gargantua/v3/pkg/microservices"
	"github.com/hobbyfarm/gargantua/v3/pkg/signals"
	"github.com/hobbyfarm/gargantua/v3/pkg/util"
	authnpb "github.com/hobbyfarm/gargantua/v3/protos/authn"
	authrpb "github.com/hobbyfarm/gargantua/v3/protos/authr"
	rbacpb "github.com/hobbyfarm/gargantua/v3/protos/rbac"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
)

var (
	installRBACRoles bool
	serviceConfig    *microservices.ServiceConfig
)

func init() {
	flag.BoolVar(&installRBACRoles, "installrbacroles", false, "Install default RBAC Roles")
	serviceConfig = microservices.BuildServiceConfig()
}

func main() {
	cfg, _, _ := microservices.BuildClusterConfig(serviceConfig)

	// self manage default rbac roles
	if installRBACRoles {
		ctx := context.Background()
		err := rbacinstaller.Create(ctx, cfg)
		if err != nil {
			glog.Fatalf("Error installing RBAC roles: %s", err.Error())
		}
		glog.V(9).Infof("Successfully installed RBAC Roles")
	}

	namespace := util.GetReleaseNamespace()
	kubeClient, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		glog.Fatalf("Error building kubernetes clientset: %s", err.Error())
	}
	kubeInformerFactory := informers.NewSharedInformerFactoryWithOptions(kubeClient, time.Second*30, informers.WithNamespace(namespace))

	gs := microservices.CreateGRPCServer(serviceConfig.ServerCert.Clone())
	rs, err := rbacservice.NewGrpcRbacServer(kubeClient, namespace, kubeInformerFactory)
	if err != nil {
		glog.Fatalf("Failed to start rbac grpc server: %s", err)
	}
	rbacpb.RegisterRbacSvcServer(gs, rs)

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

	var wg sync.WaitGroup
	// only add 1 to our wait group since our service should stop (and restart) as soon as one of the go routines terminates
	wg.Add(1)

	go func() {
		defer wg.Done()
		microservices.StartGRPCServer(gs, serviceConfig.EnableReflection)
	}()

	go func() {
		defer wg.Done()
		rbacServer := rbacservice.NewRbacServer(rs, authnClient, authrClient)
		microservices.StartAPIServer(rbacServer)
	}()

	stopCh := signals.SetupSignalHandler()
	kubeInformerFactory.Start(stopCh)
	wg.Wait()
}
