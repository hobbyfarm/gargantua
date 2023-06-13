package main

import (
	"context"
	"crypto/tls"
	"flag"
	"net"
	"net/http"
	"os"

	"github.com/ebauman/crder"
	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	hfClientset "github.com/hobbyfarm/gargantua/pkg/client/clientset/versioned"
	tls2 "github.com/hobbyfarm/gargantua/pkg/tls"
	"github.com/hobbyfarm/gargantua/pkg/util"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/reflection"

	"github.com/golang/glog"
	userservice "github.com/hobbyfarm/gargantua/internal/usersvc"

	usr "github.com/hobbyfarm/gargantua/protos/user"
	v1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
)

var (
	userTLSCert     string
	userTLSKey      string
	userTLSCA       string
	localMasterUrl  string
	localKubeconfig string
	webhookTLSCA    string
)

func init() {
	flag.StringVar(&localKubeconfig, "kubeconfig", "", "Path to kubeconfig of local cluster. Only required if out-of-cluster.")
	flag.StringVar(&localMasterUrl, "master", "", "The address of the Kubernetes API server. Overrides any value in kubeconfig. Only required if out-of-cluster.")
	flag.StringVar(&userTLSCert, "user-tls-cert", "/etc/ssl/certs/tls.crt", "Path to TLS certificate for user server")
	flag.StringVar(&userTLSKey, "user-tls-key", "/etc/ssl/certs/tls.key", "Path to TLS key for user server")
	flag.StringVar(&userTLSCA, "user-tls-ca", "/etc/ssl/certs/ca.crt", "Path to CA cert for user server")
	flag.StringVar(&webhookTLSCA, "webhook-tls-ca", "/webhook-secret/ca.crt", "Path to CA cert for webhook server")
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

	ca, err := os.ReadFile(webhookTLSCA)
	if err != nil {
		glog.Fatalf("error reading ca certificate: %s", err.Error())
	}

	crds := userservice.GenerateUserCRD(string(ca), v1.ServiceReference{
		Namespace: util.GetReleaseNamespace(),
		Name:      "hobbyfarm-webhook",
	})

	glog.Info("installing/updating user CRD")
	err = crder.InstallUpdateCRDs(cfg, crds...)
	if err != nil {
		glog.Fatalf("failed installing/updating user crd: %s", err.Error())
	}
	glog.Info("finished installing/updating user CRD")

	corsHeaders := handlers.AllowedHeaders([]string{"Authorization", "Content-Type"})
	corsOrigins := handlers.AllowedOrigins([]string{"*"})
	corsMethods := handlers.AllowedMethods([]string{"GET", "POST", "PUT", "HEAD", "OPTIONS", "DELETE"})

	ctx := context.Background()

	cert, err := tls2.ReadKeyPair(userTLSCert, userTLSKey)
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
	us := userservice.NewGrpcUserServer(hfClient, ctx)
	usr.RegisterUserSvcServer(gs, us)
	reflection.Register(gs)

	go func() {
		glog.Info("grpc user server listening on " + grpcPort)
		l, errr := net.Listen("tcp", ":"+grpcPort)

		if errr != nil {
			glog.Info("Can not serve grpc")
		}
		glog.Fatal(gs.Serve(l))
	}()

	r := mux.NewRouter()
	userServer, err := userservice.NewUserServer(userTLSCA, us)
	if err != nil {
		glog.Fatal(err)
	}
	userServer.SetupRoutes(r)
	http.Handle("/", r)
	apiPort := os.Getenv("PORT")
	if apiPort == "" {
		apiPort = "80"
	}
	glog.Info("http user server listening on " + apiPort)

	glog.Fatal(http.ListenAndServe(":"+apiPort, handlers.CORS(corsHeaders, corsOrigins, corsMethods)(r)))
}