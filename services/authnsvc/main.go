package main

import (
	"net/http"
	"os"
	"sync"

	"github.com/hobbyfarm/gargantua/v3/pkg/microservices"

	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"

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
	accesscodeConn, err := microservices.EstablishConnection(microservices.AccessCode, serviceConfig.ClientCert.Clone())
	if err != nil {
		glog.Fatalf("failed connecting to service accesscode-service: %v", err)
	}
	defer accesscodeConn.Close()

	accesscodeClient := accesscode.NewAccessCodeSvcClient(accesscodeConn)

	userConn, err := microservices.EstablishConnection(microservices.User, serviceConfig.ClientCert.Clone())
	if err != nil {
		glog.Fatalf("failed connecting to service user-service: %v", err)
	}
	defer userConn.Close()

	userClient := user.NewUserSvcClient(userConn)

	settingConn, err := microservices.EstablishConnection(microservices.Setting, serviceConfig.ClientCert.Clone())
	if err != nil {
		glog.Fatalf("failed connecting to service setting-service: %v", err)
	}
	defer settingConn.Close()

	settingClient := setting.NewSettingSvcClient(settingConn)

	rbacConn, err := microservices.EstablishConnection(microservices.Rbac, serviceConfig.ClientCert.Clone())
	if err != nil {
		glog.Fatalf("failed connecting to service rbac-service: %v", err)
	}
	defer rbacConn.Close()

	rbacClient := rbac.NewRbacSvcClient(rbacConn)

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

		r := mux.NewRouter()
		authServer.SetupRoutes(r)

		http.Handle("/", r)
		apiPort := os.Getenv("PORT")
		if apiPort == "" {
			apiPort = "80"
		}
		glog.Info("http auth server listening on " + apiPort)
		glog.Fatal(http.ListenAndServe(":"+apiPort, handlers.CORS(microservices.CORS_HANDLER_ALLOWED_HEADERS, microservices.CORS_HANDLER_ALLOWED_METHODS, microservices.CORS_HANDLER_ALLOWED_ORIGINS)(r)))
	}()

	wg.Wait()
}
