package main

import (
	"flag"
	"net"
	"os"
	"sync"

	"github.com/hobbyfarm/gargantua/v3/pkg/microservices"

	"google.golang.org/grpc/reflection"

	"github.com/golang/glog"
	authrservice "github.com/hobbyfarm/gargantua/services/authrsvc/v3/internal"

	"github.com/hobbyfarm/gargantua/v3/protos/authr"
	"github.com/hobbyfarm/gargantua/v3/protos/rbac"
)

var (
	authTLSCert      string
	authTLSKey       string
	authTLSCA        string
	enableReflection bool
)

func init() {
	flag.StringVar(&authTLSCert, "auth-tls-cert", "/etc/ssl/certs/tls.crt", "Path to TLS certificate for authr server")
	flag.StringVar(&authTLSKey, "auth-tls-key", "/etc/ssl/certs/tls.key", "Path to TLS key for authr server")
	flag.StringVar(&authTLSCA, "auth-tls-ca", "/etc/ssl/certs/ca.crt", "Path to CA cert for authr server")
	flag.BoolVar(&enableReflection, "enableReflection", true, "Enable reflection")
}

func main() {
	flag.Parse()

	cert, err := microservices.BuildTLSCredentials(authTLSCA, authTLSCert, authTLSKey)
	if err != nil {
		glog.Fatalf("error building cert: %v", err)
	}

	rbacConn, err := microservices.EstablishConnection(microservices.Rbac, cert)
	if err != nil {
		glog.Fatalf("failed connecting to service rbac-service: %v", err)
	}
	defer rbacConn.Close()

	rbacClient := rbac.NewRbacSvcClient(rbacConn)

	gs := microservices.CreateGRPCServer(cert)
	as := authrservice.NewGrpcAuthRServer(rbacClient)
	authr.RegisterAuthRServer(gs, as)
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
			glog.Fatalf("Can not serve grpc: %v", errr)
		}
		glog.Info("grpc rbac server listening on " + grpcPort)
		glog.Fatal(gs.Serve(l))
	}()

	wg.Wait()
}
