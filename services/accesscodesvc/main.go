package main

import (
	"context"
	"flag"
	"net"
	"os"
	"sync"

	"github.com/ebauman/crder"
	"github.com/hobbyfarm/gargantua/v3/pkg/microservices"

	"google.golang.org/grpc/reflection"

	"github.com/golang/glog"
	accesscodeservice "github.com/hobbyfarm/gargantua/services/accesscodesvc/v3/internal"
	accessCodeProto "github.com/hobbyfarm/gargantua/v3/protos/accesscode"
)

var (
	acTLSCert        string
	acTLSKey         string
	acTLSCA          string
	localMasterUrl   string
	localKubeconfig  string
	enableReflection bool
)

func init() {
	flag.StringVar(&acTLSCert, "ac-tls-cert", "/etc/ssl/certs/tls.crt", "Path to TLS certificate for access code server")
	flag.StringVar(&acTLSKey, "ac-tls-key", "/etc/ssl/certs/tls.key", "Path to TLS key for access code server")
	flag.StringVar(&acTLSCA, "ac-tls-ca", "/etc/ssl/certs/ca.crt", "Path to CA cert for access code server")
	flag.StringVar(&localKubeconfig, "kubeconfig", "", "Path to kubeconfig of local cluster. Only required if out-of-cluster.")
	flag.StringVar(&localMasterUrl, "master", "", "The address of the Kubernetes API server. Overrides any value in kubeconfig. Only required if out-of-cluster.")
	flag.BoolVar(&enableReflection, "enableReflection", true, "Enable reflection")
}

func main() {
	flag.Parse()

	cfg, hfClient := microservices.BuildClusterConfig(localMasterUrl, localKubeconfig)

	crds := accesscodeservice.GenerateAccessCodeCRD()

	glog.Info("installing/updating access code CRDs")
	err := crder.InstallUpdateCRDs(cfg, crds...)
	if err != nil {
		glog.Fatalf("failed installing/updating access code CRDs: %s", err.Error())
	}
	glog.Info("finished installing/updating access code CRDs")

	ctx := context.Background()

	cert, err := microservices.BuildTLSCredentials(acTLSCA, acTLSCert, acTLSKey)
	if err != nil {
		glog.Fatalf("error building cert: %v", err)
	}

	gs := microservices.CreateGRPCServer(cert)
	as := accesscodeservice.NewGrpcAccessCodeServer(hfClient, ctx)
	accessCodeProto.RegisterAccessCodeSvcServer(gs, as)
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
