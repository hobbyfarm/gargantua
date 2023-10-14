package main

import (
	"crypto/tls"
	"flag"
	"net/http"
	"os"

	"github.com/golang/glog"
	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	"github.com/hobbyfarm/gargantua/internal/webhook/conversion"
	"github.com/hobbyfarm/gargantua/internal/webhook/conversion/user"
	"github.com/hobbyfarm/gargantua/internal/webhook/validation"
	hfClientset "github.com/hobbyfarm/gargantua/pkg/client/clientset/versioned"
	tls2 "github.com/hobbyfarm/gargantua/pkg/tls"
	apiextensions "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

var (
	webhookTLSCert  string
	webhookTLSKey   string
	webhookTLSCA    string
	localMasterUrl  string
	localKubeconfig string
)

func init() {
	flag.StringVar(&localKubeconfig, "kubeconfig", "", "Path to kubeconfig of local cluster. Only required if out-of-cluster.")
	flag.StringVar(&localMasterUrl, "master", "", "The address of the Kubernetes API server. Overrides any value in kubeconfig. Only required if out-of-cluster.")
	flag.StringVar(&webhookTLSCert, "webhook-tls-cert", "/etc/ssl/certs/tls.crt", "Path to TLS certificate for webhook server")
	flag.StringVar(&webhookTLSKey, "webhook-tls-key", "/etc/ssl/certs/tls.key", "Path to TLS key for webhook server")
	flag.StringVar(&webhookTLSCA, "webhook-tls-ca", "/etc/ssl/certs/ca.crt", "Path to CA cert for webhook server")
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

	apiExtensionsClient, err := apiextensions.NewForConfig(cfg)
	if err != nil {
		glog.Fatalf("error building apiextensions clientset: %s", err.Error())
	}

	corsHeaders := handlers.AllowedHeaders([]string{"Authorization", "Content-Type"})
	corsOrigins := handlers.AllowedOrigins([]string{"*"})
	corsMethods := handlers.AllowedMethods([]string{"GET", "POST", "PUT", "HEAD", "OPTIONS", "DELETE"})

	user.Init()
	conversionRouter := mux.NewRouter()
	conversion.New(conversionRouter, apiExtensionsClient, string(ca))

	validationEndpoints := conversionRouter.PathPrefix("/validation").Subrouter()
	validation.SetupValidationServer(hfClient, validationEndpoints)

	webhookPort := os.Getenv("WEBHOOK_PORT")
	if webhookPort == "" {
		webhookPort = "444"
	}
	glog.Info("webhook listening on " + webhookPort)

	cert, err := tls2.ReadKeyPair(webhookTLSCert, webhookTLSKey)
	if err != nil {
		glog.Fatalf("error generating x509keypair from conversion cert and key: %s", err)
	}

	server := http.Server{
		TLSConfig: &tls.Config{
			Certificates: []tls.Certificate{*cert},
		},
		Addr:    ":" + webhookPort,
		Handler: handlers.CORS(corsHeaders, corsOrigins, corsMethods)(conversionRouter),
	}

	glog.Fatal(server.ListenAndServeTLS("", ""))
}
