package main

import (
	"sync"

	"github.com/hobbyfarm/gargantua/v3/pkg/microservices"

	authrservice "github.com/hobbyfarm/gargantua/services/authrsvc/v3/internal"

	authrpb "github.com/hobbyfarm/gargantua/v3/protos/authr"
	rbacpb "github.com/hobbyfarm/gargantua/v3/protos/rbac"
)

var (
	serviceConfig *microservices.ServiceConfig
)

func init() {
	serviceConfig = microservices.BuildServiceConfig()
}

func main() {
	services := []microservices.MicroService{
		microservices.Rbac,
	}
	connections := microservices.EstablishConnections(services, serviceConfig.ClientCert)
	for _, conn := range connections {
		defer conn.Close()
	}

	rbacClient := rbacpb.NewRbacSvcClient(connections[microservices.Rbac])
	gs := microservices.CreateGRPCServer(serviceConfig.ServerCert.Clone())
	as := authrservice.NewGrpcAuthRServer(rbacClient)
	authrpb.RegisterAuthRServer(gs, as)

	var wg sync.WaitGroup
	wg.Add(1)

	go func() {
		defer wg.Done()
		microservices.StartGRPCServer(gs, serviceConfig.EnableReflection)
	}()

	wg.Wait()
}
