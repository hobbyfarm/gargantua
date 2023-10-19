package main

import (
	"crypto/tls"
	"flag"
	"net"
	"net/http"
	"os"
	"sync"

	"github.com/hobbyfarm/gargantua/v3/pkg/microservices"
	tls2 "github.com/hobbyfarm/gargantua/v3/pkg/tls"

	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/reflection"

	"github.com/golang/glog"
	authnservice "github.com/hobbyfarm/gargantua/services/authnsvc/v3/internal"

	"github.com/hobbyfarm/gargantua/v3/protos/accesscode"
	"github.com/hobbyfarm/gargantua/v3/protos/authn"
	"github.com/hobbyfarm/gargantua/v3/protos/rbac"
	"github.com/hobbyfarm/gargantua/v3/protos/setting"
	"github.com/hobbyfarm/gargantua/v3/protos/user"
)

var (
	localMasterUrl   string
	localKubeconfig  string
	installRBACRoles bool
	authTLSCert      string
	authTLSKey       string
	authTLSCA        string
	enableReflection bool
)

func init() {
	flag.StringVar(&localKubeconfig, "kubeconfig", "", "Path to kubeconfig of local cluster. Only required if out-of-cluster.")
	flag.StringVar(&localMasterUrl, "master", "", "The address of the Kubernetes API server. Overrides any value in kubeconfig. Only required if out-of-cluster.")
	flag.BoolVar(&installRBACRoles, "installrbacroles", false, "Install default RBAC Roles")
	flag.StringVar(&authTLSCert, "auth-tls-cert", "/etc/ssl/certs/tls.crt", "Path to TLS certificate for authn server")
	flag.StringVar(&authTLSKey, "auth-tls-key", "/etc/ssl/certs/tls.key", "Path to TLS key for authn server")
	flag.StringVar(&authTLSCA, "auth-tls-ca", "/etc/ssl/certs/ca.crt", "Path to CA cert for authn server")
	flag.BoolVar(&enableReflection, "enableReflection", true, "Enable reflection")
}

// TODO: Remove rbacClient, hfClientSet etc.
func main() {
	flag.Parse()
	glog.V(2).Infof("Starting Authentication Service")

	cert, err := tls2.ReadKeyPair(authTLSCert, authTLSKey)
	if err != nil {
		glog.Fatalf("error generating x509keypair from conversion cert and key: %s", err)
	}

	creds := credentials.NewTLS(&tls.Config{
		Certificates: []tls.Certificate{*cert},
	})

	accesscodeConn, err := microservices.EstablishConnection(microservices.AccessCode, authTLSCA)
	if err != nil {
		glog.Fatalf("failed connecting to service accesscode-service: %v", err)
	}
	defer accesscodeConn.Close()

	accesscodeClient := accesscode.NewAccessCodeSvcClient(accesscodeConn)

	userConn, err := microservices.EstablishConnection(microservices.User, authTLSCA)
	if err != nil {
		glog.Fatalf("failed connecting to service user-service: %v", err)
	}
	defer userConn.Close()

	userClient := user.NewUserSvcClient(userConn)

	settingConn, err := microservices.EstablishConnection(microservices.Setting, authTLSCA)
	if err != nil {
		glog.Fatalf("failed connecting to service setting-service: %v", err)
	}
	defer settingConn.Close()

	settingClient := setting.NewSettingSvcClient(settingConn)

	rbacConn, err := microservices.EstablishConnection(microservices.Rbac, authTLSCA)
	if err != nil {
		glog.Fatalf("failed connecting to service rbac-service: %v", err)
	}
	defer rbacConn.Close()

	rbacClient := rbac.NewRbacSvcClient(rbacConn)

	gs := microservices.CreateGRPCServer(creds)
	as := authnservice.NewGrpcAuthNServer(userClient)
	authn.RegisterAuthNServer(gs, as)
	if enableReflection {
		reflection.Register(gs)
	}

	var wg sync.WaitGroup
	wg.Add(1)

	go func() {
		defer wg.Done()

		grpcPort := os.Getenv("GRPC_PORT")
		if grpcPort == "" {
			grpcPort = "8080"
		}
		l, errr := net.Listen("tcp", ":"+grpcPort)

		if errr != nil {
			glog.Infof("Can not serve grpc: %v", errr)
			return
		}

		glog.Info("grpc auth server listening on " + grpcPort)
		glog.Fatal(gs.Serve(l))
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
