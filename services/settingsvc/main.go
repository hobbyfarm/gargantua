package main

import (
	"context"
	"crypto/tls"
	"flag"
	"net"
	"net/http"
	"os"
	"time"

	"github.com/ebauman/crder"
	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	hfClientset "github.com/hobbyfarm/gargantua/v3/pkg/client/clientset/versioned"
	"github.com/hobbyfarm/gargantua/v3/pkg/crd"
	"github.com/hobbyfarm/gargantua/v3/pkg/microservices"
	"github.com/hobbyfarm/gargantua/v3/pkg/signals"
	tls2 "github.com/hobbyfarm/gargantua/v3/pkg/tls"
	"github.com/hobbyfarm/gargantua/v3/pkg/util"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/leaderelection"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
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
	stopCh := signals.SetupSignalHandler()
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

	namespace := util.GetReleaseNamespace()

	hfClient, err := hfClientset.NewForConfig(cfg)
	if err != nil {
		glog.Fatal(err)
	}

	hfInformerFactory := hfInformers.NewSharedInformerFactoryWithOptions(hfClient, time.Second*30, hfInformers.WithNamespace(namespace))

	lock, err := util.GetLock("controller-manager-setting", cfg)
	if err != nil {
		glog.Fatal(err)
	}

	// Creating the leader election config
	leaderElectionConfig := leaderelection.LeaderElectionConfig{
		Lock:            lock,
		ReleaseOnCancel: true,
		LeaseDuration:   10 * time.Second,
		RenewDeadline:   5 * time.Second,
		RetryPeriod:     2 * time.Second,
		Callbacks: leaderelection.LeaderCallbacks{
			OnStartedLeading: func(ctx context.Context) {
				// Start informer
				hfInformerFactory.Start(stopCh)
			},
			OnStoppedLeading: func() {
				// Need to start informer factory since even when not leader to ensure api layer
				// keeps working.
				hfInformerFactory.Start(stopCh)
			},
			OnNewLeader: func(current_id string) {
				hfInformerFactory.Start(stopCh)
				if current_id == lock.Identity() {
					glog.Info("currently the leader")
					return
				}
				glog.Infof("current leader is %s", current_id)
			},
		},
	}

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

	corsHeaders := handlers.AllowedHeaders([]string{"Authorization", "Content-Type"})
	corsOrigins := handlers.AllowedOrigins([]string{"*"})
	corsMethods := handlers.AllowedMethods([]string{"GET", "POST", "PUT", "HEAD", "OPTIONS", "DELETE"})

	ctx := context.Background()

	cert, err := tls2.ReadKeyPair(settingTLSCert, settingTLSKey)
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

	settingservice.WatchSettings(ctx, hfClient, hfInformerFactory)
	gs := grpc.NewServer(grpc.Creds(creds))
	ss := settingservice.NewGrpcSettingServer(hfClient, ctx)
	settingProto.RegisterSettingSvcServer(gs, ss)
	if enableReflection {
		reflection.Register(gs)
	}
	settingservice.Preinstall(ctx, ss)

	authnConn, err := microservices.EstablishConnection(microservices.AuthN, settingTLSCA)
	if err != nil {
		glog.Fatalf("failed connecting to service authn-service: %v", err)
	}
	defer authnConn.Close()

	authnClient := authn.NewAuthNClient(authnConn)

	authrConn, err := microservices.EstablishConnection(microservices.AuthR, settingTLSCA)
	if err != nil {
		glog.Fatalf("failed connecting to service authn-service: %v", err)
	}
	defer authrConn.Close()

	authrClient := authr.NewAuthRClient(authrConn)

	go func() {
		glog.Info("grpc setting server listening on " + grpcPort)
		l, errr := net.Listen("tcp", ":"+grpcPort)

		if errr != nil {
			glog.Info("Can not serve grpc")
		}
		glog.Fatal(gs.Serve(l))
	}()

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

	go func() {
		glog.Fatal(http.ListenAndServe(":"+apiPort, handlers.CORS(corsHeaders, corsOrigins, corsMethods)(r)))
	}()

	// Run leader election
	leaderelection.RunOrDie(ctx, leaderElectionConfig)
}
