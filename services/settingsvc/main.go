package main

import (
	"context"
	"flag"
	"net"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/ebauman/crder"
	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	"github.com/hobbyfarm/gargantua/v3/pkg/crd"
	"github.com/hobbyfarm/gargantua/v3/pkg/microservices"
	"github.com/hobbyfarm/gargantua/v3/pkg/signals"
	"github.com/hobbyfarm/gargantua/v3/pkg/util"

	"google.golang.org/grpc/reflection"

	"github.com/golang/glog"
	settingservice "github.com/hobbyfarm/gargantua/services/settingsvc/v3/internal"
	hfInformers "github.com/hobbyfarm/gargantua/v3/pkg/client/informers/externalversions"

	"github.com/hobbyfarm/gargantua/v3/protos/authn"
	"github.com/hobbyfarm/gargantua/v3/protos/authr"
	settingProto "github.com/hobbyfarm/gargantua/v3/protos/setting"
)

var (
	settingTLSCert   string
	settingTLSKey    string
	settingTLSCA     string
	webhookTLSCA     string
	localMasterUrl   string
	localKubeconfig  string
	enableReflection bool
)

func init() {
	flag.StringVar(&localKubeconfig, "kubeconfig", "", "Path to kubeconfig of local cluster. Only required if out-of-cluster.")
	flag.StringVar(&localMasterUrl, "master", "", "The address of the Kubernetes API server. Overrides any value in kubeconfig. Only required if out-of-cluster.")
	flag.StringVar(&settingTLSCert, "setting-tls-cert", "/etc/ssl/certs/tls.crt", "Path to TLS certificate for setting server")
	flag.StringVar(&settingTLSKey, "setting-tls-key", "/etc/ssl/certs/tls.key", "Path to TLS key for setting server")
	flag.StringVar(&settingTLSCA, "setting-tls-ca", "/etc/ssl/certs/ca.crt", "Path to CA cert for setting server")
	flag.StringVar(&webhookTLSCA, "webhook-tls-ca", "/webhook-secret/ca.crt", "Path to CA cert for webhook server")
	flag.BoolVar(&enableReflection, "enableReflection", true, "Enable reflection")
}

func main() {
	flag.Parse()

	cfg, hfClient := microservices.BuildClusterConfig(localMasterUrl, localKubeconfig)

	namespace := util.GetReleaseNamespace()

	hfInformerFactory := hfInformers.NewSharedInformerFactoryWithOptions(hfClient, time.Second*30, hfInformers.WithNamespace(namespace))

	ca, err := os.ReadFile(webhookTLSCA)
	if err != nil {
		glog.Fatalf("error reading ca certificate: %s", err.Error())
	}

	crds := settingservice.GenerateSettingCRD(string(ca), crd.ServiceReference{
		Namespace: util.GetReleaseNamespace(),
		Name:      "hobbyfarm-webhook",
	})

	glog.Info("installing/updating setting CRD")
	err = crder.InstallUpdateCRDs(cfg, crds...)
	if err != nil {
		glog.Fatalf("failed installing/updating setting crd: %s", err.Error())
	}
	glog.Info("finished installing/updating setting CRD")

	ctx := context.Background()

	cert, err := microservices.BuildTLSCredentials(settingTLSCA, settingTLSCert, settingTLSKey)
	if err != nil {
		glog.Fatalf("error building cert: %v", err)
	}

	authnConn, err := microservices.EstablishConnection(microservices.AuthN, cert)
	if err != nil {
		glog.Fatalf("failed connecting to service authn-service: %v", err)
	}
	defer authnConn.Close()

	authnClient := authn.NewAuthNClient(authnConn)

	authrConn, err := microservices.EstablishConnection(microservices.AuthR, cert)
	if err != nil {
		glog.Fatalf("failed connecting to service authr-service: %v", err)
	}
	defer authrConn.Close()

	authrClient := authr.NewAuthRClient(authrConn)

	err = settingservice.WatchSettings(ctx, hfClient, hfInformerFactory)
	if err != nil {
		glog.Info("watching settings failed: ", err)
	}
	gs := microservices.CreateGRPCServer(cert)
	ss := settingservice.NewGrpcSettingServer(hfClient, ctx)
	settingProto.RegisterSettingSvcServer(gs, ss)
	if enableReflection {
		reflection.Register(gs)
	}
	settingservice.Preinstall(ctx, ss)

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
		glog.Info("grpc setting server listening on " + grpcPort)
		glog.Fatal(gs.Serve(l))
	}()

	go func() {
		defer wg.Done()

		r := mux.NewRouter()
		settingServer, err := settingservice.NewSettingServer(authnClient, authrClient, ss)
		if err != nil {
			glog.Fatal(err)
		}
		settingServer.SetupRoutes(r)
		http.Handle("/", r)

		apiPort := os.Getenv("PORT")
		if apiPort == "" {
			apiPort = "80"
		}

		glog.Info("http setting server listening on " + apiPort)
		glog.Fatal(http.ListenAndServe(":"+apiPort, handlers.CORS(microservices.CORS_HANDLER_ALLOWED_HEADERS, microservices.CORS_HANDLER_ALLOWED_METHODS, microservices.CORS_HANDLER_ALLOWED_ORIGINS)(r)))
	}()

	stopCh := signals.SetupSignalHandler()
	hfInformerFactory.Start(stopCh)
	wg.Wait()
}
