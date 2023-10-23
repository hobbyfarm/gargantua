package main

import (
	"sync"

	"github.com/hobbyfarm/gargantua/v3/pkg/microservices"

	"github.com/golang/glog"
	authnservice "github.com/hobbyfarm/gargantua/services/authnsvc/v3/internal"

	"github.com/hobbyfarm/gargantua/v3/protos/accesscode"
	"github.com/hobbyfarm/gargantua/v3/protos/authn"
	"github.com/hobbyfarm/gargantua/v3/protos/rbac"
	"github.com/hobbyfarm/gargantua/v3/protos/setting"
	"github.com/hobbyfarm/gargantua/v3/protos/user"
)

var (
	serviceConfig *microservices.ServiceConfig
)

func init() {
	serviceConfig = microservices.BuildServiceConfig()
}

// TODO: Remove rbacClient, hfClientSet etc.
func main() {
	services := []microservices.MicroService{
		microservices.AccessCode,
		microservices.User,
		microservices.Setting,
		microservices.Rbac,
	}
	connections := microservices.EstablishConnections(services, serviceConfig.ClientCert)
	for _, conn := range connections {
		defer conn.Close()
	}

	rbacClient := rbac.NewRbacSvcClient(connections[microservices.Rbac])
	accesscodeClient := accesscode.NewAccessCodeSvcClient(connections[microservices.AccessCode])
	userClient := user.NewUserSvcClient(connections[microservices.User])
	settingClient := setting.NewSettingSvcClient(connections[microservices.Setting])

	gs := microservices.CreateGRPCServer(serviceConfig.ServerCert.Clone())
	as := authnservice.NewGrpcAuthNServer(userClient)
	authn.RegisterAuthNServer(gs, as)

	var wg sync.WaitGroup
	wg.Add(1)

	go func() {
		defer wg.Done()
		microservices.StartGRPCServer(gs, serviceConfig.EnableReflection)
	}()

	go func() {
		defer wg.Done()
		authServer, err := authnservice.NewAuthServer(accesscodeClient, userClient, settingClient, rbacClient, as)
		if err != nil {
			glog.Fatal(err)
		}
		microservices.StartAPIServer(authServer)
	}()

	wg.Wait()
}
