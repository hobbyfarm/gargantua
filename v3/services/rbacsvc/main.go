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
	"github.com/hobbyfarm/gargantua/v3/protos/authn"
	"github.com/hobbyfarm/gargantua/v3/protos/authr"
	rbacProto "github.com/hobbyfarm/gargantua/v3/protos/rbac"
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
	rbacProto.RegisterRbacSvcServer(gs, rs)

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

	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		defer wg.Done()
		microservices.StartGRPCServer(gs, serviceConfig.EnableReflection)
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		rbacServer := rbacservice.NewRbacServer(rs, authnClient, authrClient)
		microservices.StartAPIServer(rbacServer)
	}()

	stopCh := signals.SetupSignalHandler()
	kubeInformerFactory.Start(stopCh)
	wg.Wait()
}
