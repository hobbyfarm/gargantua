package main

import (
	"sync"

	"github.com/hobbyfarm/gargantua/v3/pkg/microservices"

	"github.com/golang/glog"
	authnservice "github.com/hobbyfarm/gargantua/services/authnsvc/v3/internal"

	accesscodepb "github.com/hobbyfarm/gargantua/v3/protos/accesscode"
	authnpb "github.com/hobbyfarm/gargantua/v3/protos/authn"
	rbacpb "github.com/hobbyfarm/gargantua/v3/protos/rbac"
	scheduledeventpb "github.com/hobbyfarm/gargantua/v3/protos/scheduledevent"
	settingpb "github.com/hobbyfarm/gargantua/v3/protos/setting"
	userpb "github.com/hobbyfarm/gargantua/v3/protos/user"
)

var (
	serviceConfig *microservices.ServiceConfig
)

func init() {
	serviceConfig = microservices.BuildServiceConfig()
}

func main() {
	services := []microservices.MicroService{
		microservices.AccessCode,
		microservices.Rbac,
		microservices.ScheduledEvent,
		microservices.Setting,
		microservices.User,
	}
	connections := microservices.EstablishConnections(services, serviceConfig.ClientCert)
	for _, conn := range connections {
		defer conn.Close()
	}

	accesscodeClient := accesscodepb.NewAccessCodeSvcClient(connections[microservices.AccessCode])
	rbacClient := rbacpb.NewRbacSvcClient(connections[microservices.Rbac])
	scheduledEventClient := scheduledeventpb.NewScheduledEventSvcClient(connections[microservices.ScheduledEvent])
	settingClient := settingpb.NewSettingSvcClient(connections[microservices.Setting])
	userClient := userpb.NewUserSvcClient(connections[microservices.User])

	gs := microservices.CreateGRPCServer(serviceConfig.ServerCert.Clone())
	as := authnservice.NewGrpcAuthNServer(userClient)
	authnpb.RegisterAuthNServer(gs, as)

	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		defer wg.Done()
		microservices.StartGRPCServer(gs, serviceConfig.EnableReflection)
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		authServer, err := authnservice.NewAuthServer(accesscodeClient, rbacClient, scheduledEventClient, settingClient, userClient, as)
		if err != nil {
			glog.Fatal(err)
		}
		microservices.StartAPIServer(authServer)
	}()

	wg.Wait()
}
