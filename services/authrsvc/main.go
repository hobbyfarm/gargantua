package main

import (
	"crypto/tls"
	"flag"
	"net"
	"os"

	tls2 "github.com/hobbyfarm/gargantua/pkg/tls"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/reflection"

	"github.com/golang/glog"
	authrservice "github.com/hobbyfarm/gargantua/internal/authrsvc"

	authr "github.com/hobbyfarm/gargantua/protos/authr"
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
	glog.Info("grpc authr server listening on " + grpcPort)

	gs := grpc.NewServer(grpc.Creds(creds))
	as := authrservice.NewGrpcAuthRServer(authTLSCA)
	authr.RegisterAuthRServer(gs, as)
	if enableReflection {
		reflection.Register(gs)
	}

	l, errr := net.Listen("tcp", ":"+grpcPort)

	if errr != nil {
		glog.Info("Can not serve grpc")
	}
	glog.Fatal(gs.Serve(l))
}