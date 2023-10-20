package main

import (
	"sync"

	"github.com/hobbyfarm/gargantua/v3/pkg/microservices"

	"github.com/golang/glog"
	authrservice "github.com/hobbyfarm/gargantua/services/authrsvc/v3/internal"

	"github.com/hobbyfarm/gargantua/v3/protos/authr"
	"github.com/hobbyfarm/gargantua/v3/protos/rbac"
)

var (
	serviceConfig *microservices.ServiceConfig
)

func init() {
	serviceConfig = microservices.BuildServiceConfig()
}

func main() {
	rbacConn, err := microservices.EstablishConnection(microservices.Rbac, serviceConfig.ClientCert.Clone())
	if err != nil {
		glog.Fatalf("failed connecting to service rbac-service: %v", err)
	}
	defer rbacConn.Close()

	rbacClient := rbac.NewRbacSvcClient(rbacConn)

	gs := microservices.CreateGRPCServer(serviceConfig.ServerCert.Clone())
	as := authrservice.NewGrpcAuthRServer(rbacClient)
	authr.RegisterAuthRServer(gs, as)

	var wg sync.WaitGroup
	wg.Add(1)

	go func() {
		defer wg.Done()
		microservices.StartGRPCServer(gs, serviceConfig.EnableReflection)
	}()

	wg.Wait()
}
