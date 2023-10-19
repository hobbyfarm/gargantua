package main

import (
	"crypto/tls"
	"flag"
	"net/http"
	"os"

	"github.com/golang/glog"
	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	"github.com/hobbyfarm/gargantua/services/conversionsvc/v3/internal/conversion"
	"github.com/hobbyfarm/gargantua/services/conversionsvc/v3/internal/conversion/user"
	"github.com/hobbyfarm/gargantua/services/conversionsvc/v3/internal/validation"
	"github.com/hobbyfarm/gargantua/v3/pkg/microservices"
	tls2 "github.com/hobbyfarm/gargantua/v3/pkg/tls"
	apiextensions "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
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

	cfg, hfClient := microservices.BuildClusterConfig(localMasterUrl, localKubeconfig)

	ca, err := os.ReadFile(webhookTLSCA)
	if err != nil {
		glog.Fatalf("error reading ca certificate: %s", err.Error())
	}

	apiExtensionsClient, err := apiextensions.NewForConfig(cfg)
	if err != nil {
		glog.Fatalf("error building apiextensions clientset: %s", err.Error())
	}

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
		Handler: handlers.CORS(microservices.CORS_HANDLER_ALLOWED_HEADERS, microservices.CORS_HANDLER_ALLOWED_METHODS, microservices.CORS_HANDLER_ALLOWED_ORIGINS)(conversionRouter),
	}

	glog.Fatal(server.ListenAndServeTLS("", ""))
}
