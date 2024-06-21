package main

import (
	"context"
	"flag"
	"os"

	hfClientset "github.com/hobbyfarm/gargantua/v3/pkg/client/clientset/versioned"
	"github.com/hobbyfarm/gargantua/v3/pkg/crd"
	"github.com/hobbyfarm/gargantua/v3/pkg/microservices"
	predefinedserviceserver "github.com/hobbyfarm/gargantua/v3/pkg/predefinedserviceserver"
	"github.com/hobbyfarm/gargantua/v3/pkg/shell"
	authnpb "github.com/hobbyfarm/gargantua/v3/protos/authn"
	authrpb "github.com/hobbyfarm/gargantua/v3/protos/authr"
	vmpb "github.com/hobbyfarm/gargantua/v3/protos/vm"
	vmtemplatepb "github.com/hobbyfarm/gargantua/v3/protos/vmtemplate"

	"github.com/ebauman/crder"

	"net/http"
	"sync"

	"github.com/golang/glog"
	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"

	"k8s.io/client-go/tools/clientcmd"
)

const (
	ClientGoQPS   = 100
	ClientGoBurst = 100
)

var (
	localMasterUrl  string
	localKubeconfig string
	shellServer     bool
	tlsCA           string
)

func init() {
	flag.StringVar(&localKubeconfig, "kubeconfig", "", "Path to kubeconfig of local cluster. Only required if out-of-cluster.")
	flag.StringVar(&localMasterUrl, "master", "", "The address of the Kubernetes API server. Overrides any value in kubeconfig. Only required if out-of-cluster.")
	flag.BoolVar(&shellServer, "shellserver", false, "Be a shell server")
	flag.StringVar(&tlsCA, "tls-ca", "/etc/ssl/certs/ca.crt", "Path to CA cert for auth servers")
}

func main() {
	ctx := context.Background()
	flag.Parse()
	glog.V(2).Infof("Starting Gargantua")
	r := mux.NewRouter()

	cfg, err := rest.InClusterConfig()

	if err != nil {
		cfg, err = clientcmd.BuildConfigFromFlags(localMasterUrl, localKubeconfig)
		if err != nil {
			glog.Fatalf("Error building kubeconfig: %s", err.Error())
		}
	}

	if !shellServer {
		crds := crd.GenerateCRDs()

		glog.Info("installing/updating CRDs")
		err = crder.InstallUpdateCRDs(cfg, crds...)
		if err != nil {
			glog.Fatalf("failed installing/updating crds: %s", err.Error())
		}
		glog.Info("finished installing/updating CRDs")
	}

	cfg.QPS = ClientGoQPS
	cfg.Burst = ClientGoBurst

	hfClient, err := hfClientset.NewForConfig(cfg)
	if err != nil {
		glog.Fatal(err)
	}

	kubeClient, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		glog.Fatalf("Error building kubernetes clientset: %s", err.Error())
	}

	cert, err := microservices.BuildTLSClientCredentials(tlsCA)
	if err != nil {
		glog.Fatalf("error building cert: %v", err)
	}

	services := []microservices.MicroService{
		microservices.AuthN,
		microservices.AuthR,
		microservices.VM,
		microservices.VMTemplate,
	}
	connections := microservices.EstablishConnections(services, cert)
	for _, conn := range connections {
		defer conn.Close()
	}

	authnClient := authnpb.NewAuthNClient(connections[microservices.AuthN])
	authrClient := authrpb.NewAuthRClient(connections[microservices.AuthR])
	vmClient := vmpb.NewVMSvcClient(connections[microservices.VM])
	vmTemplateClient := vmtemplatepb.NewVMTemplateSvcClient(connections[microservices.VMTemplate])

	shellProxy := shell.NewShellProxy(authnClient, authrClient, vmClient, vmTemplateClient, kubeClient)

	predefinedServiceServer, err := predefinedserviceserver.NewPredefinedServiceServer(authnClient, authrClient, hfClient, ctx)
	if err != nil {
		glog.Fatal(err)
	}

	if shellServer {
		glog.V(2).Infof("Starting as a shell server")
		shellProxy.SetupRoutes(r)
	} else {
		predefinedServiceServer.SetupRoutes(r)
	}

	corsHeaders := handlers.AllowedHeaders([]string{"Authorization", "Content-Type"})
	corsOrigins := handlers.AllowedOrigins([]string{"*"})
	corsMethods := handlers.AllowedMethods([]string{"GET", "POST", "PUT", "HEAD", "OPTIONS", "DELETE"})

	var wg sync.WaitGroup

	wg.Add(1)

	http.Handle("/", r)

	apiPort := os.Getenv("PORT")
	if apiPort == "" {
		apiPort = "80"
	}
	glog.Info("apiserver listening on " + apiPort)

	go func() {
		defer wg.Done()
		glog.Fatal(http.ListenAndServe(":"+apiPort, handlers.CORS(corsHeaders, corsOrigins, corsMethods)(r)))
	}()

	wg.Wait()
}
