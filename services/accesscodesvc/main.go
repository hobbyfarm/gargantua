package main

import (
	"context"
	"crypto/tls"
	"flag"
	"net"
	"os"

	"github.com/ebauman/crder"
	hfClientset "github.com/hobbyfarm/gargantua/v3/pkg/client/clientset/versioned"
	tls2 "github.com/hobbyfarm/gargantua/v3/pkg/tls"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/reflection"

	"github.com/golang/glog"
	accesscodeservice "github.com/hobbyfarm/gargantua/v3/services/accesscodesvc/internal"
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

	const (
		ClientGoQPS   = 100
		ClientGoBurst = 100
	)
	cfg, err := rest.InClusterConfig()
	if err != nil {
		cfg, err = clientcmd.BuildConfigFromFlags(localMasterUrl, localKubeconfig)
		if err != nil {
			glog.Fatalf("Error building kubeconfig: %s", err.Error())
		}
	}
	cfg.QPS = ClientGoQPS
	cfg.Burst = ClientGoBurst

	hfClient, err := hfClientset.NewForConfig(cfg)
	if err != nil {
		glog.Fatal(err)
	}

	crds := accesscodeservice.GenerateAccessCodeCRD()

	glog.Info("installing/updating access code CRDs")
	err = crder.InstallUpdateCRDs(cfg, crds...)
	if err != nil {
		glog.Fatalf("failed installing/updating access code CRDs: %s", err.Error())
	}
	glog.Info("finished installing/updating access code CRDs")

	ctx := context.Background()

	cert, err := tls2.ReadKeyPair(acTLSCert, acTLSKey)
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

	gs := grpc.NewServer(grpc.Creds(creds))
	as := accesscodeservice.NewGrpcAccessCodeServer(hfClient, ctx)
	accessCodeProto.RegisterAccessCodeSvcServer(gs, as)
	if enableReflection {
		reflection.Register(gs)
	}

	glog.Info("grpc access code server listening on " + grpcPort)
	l, errr := net.Listen("tcp", ":"+grpcPort)

	if errr != nil {
		glog.Info("Can not serve grpc")
	}
	glog.Fatal(gs.Serve(l))
}
