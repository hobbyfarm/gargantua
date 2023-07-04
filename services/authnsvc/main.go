package main

import (
	"crypto/tls"
	"flag"
	"net"
	"net/http"
	"os"

	tls2 "github.com/hobbyfarm/gargantua/pkg/tls"

	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/reflection"

	"github.com/golang/glog"
	authnservice "github.com/hobbyfarm/gargantua/internal/authnsvc"

	authn "github.com/hobbyfarm/gargantua/protos/authn"
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
	r := mux.NewRouter()

	cert, err := tls2.ReadKeyPair(authTLSCert, authTLSKey)
	if err != nil {
		glog.Fatalf("error generating x509keypair from conversion cert and key: %s", err)
	}

	creds := credentials.NewTLS(&tls.Config{
		Certificates: []tls.Certificate{*cert},
	})

	grpcPort := os.Getenv("GRPC_PORT")
	if grpcPort == "" {
		grpcPort = "8080"
	}
	glog.Info("grpc auth server listening on " + grpcPort)

	gs := grpc.NewServer(grpc.Creds(creds))
	as := authnservice.NewGrpcAuthNServer(authTLSCA)
	authn.RegisterAuthNServer(gs, as)
	if enableReflection {
		reflection.Register(gs)
	}

	go func() {
		l, errr := net.Listen("tcp", ":"+grpcPort)

		if errr != nil {
			glog.Info("Can not serve grpc")
		}
		glog.Fatal(gs.Serve(l))
	}()

	authServer, err := authnservice.NewAuthServer(authTLSCA, as)
	if err != nil {
		glog.Fatal(err)
	}
	authServer.SetupRoutes(r)

	corsHeaders := handlers.AllowedHeaders([]string{"Authorization", "Content-Type"})
	corsOrigins := handlers.AllowedOrigins([]string{"*"})
	corsMethods := handlers.AllowedMethods([]string{"GET", "POST", "PUT", "HEAD", "OPTIONS", "DELETE"})

	http.Handle("/", r)

	apiPort := os.Getenv("PORT")
	if apiPort == "" {
		apiPort = "80"
	}
	glog.Info("http auth server listening on " + apiPort)

	glog.Fatal(http.ListenAndServe(":"+apiPort, handlers.CORS(corsHeaders, corsOrigins, corsMethods)(r)))
}
