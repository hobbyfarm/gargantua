package main

import (
	"context"
	"crypto/tls"
	"flag"
	"net"
	"net/http"
	"os"
	"time"

	"github.com/golang/glog"
	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	rbacservice "github.com/hobbyfarm/gargantua/services/rbacsvc/internal"
	rbacinstaller "github.com/hobbyfarm/gargantua/services/rbacsvc/internal/rbac"
	"github.com/hobbyfarm/gargantua/pkg/signals"
	tls2 "github.com/hobbyfarm/gargantua/pkg/tls"
	"github.com/hobbyfarm/gargantua/pkg/util"
	rbacProto "github.com/hobbyfarm/gargantua/protos/rbac"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/reflection"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/leaderelection"
)

var (
	tlsCert          string
	tlsKey           string
	tlsCA            string
	localMasterUrl   string
	localKubeconfig  string
	installRBACRoles bool
	enableReflection bool
)

func init() {
	flag.StringVar(&localKubeconfig, "kubeconfig", "", "Path to kubeconfig of local cluster. Only required if out-of-cluster.")
	flag.StringVar(&localMasterUrl, "master", "", "The address of the Kubernetes API server. Overrides any value in kubeconfig. Only required if out-of-cluster.")
	flag.StringVar(&tlsCert, "user-tls-cert", "/etc/ssl/certs/tls.crt", "Path to TLS certificate for user server")
	flag.StringVar(&tlsKey, "user-tls-key", "/etc/ssl/certs/tls.key", "Path to TLS key for user server")
	flag.StringVar(&tlsCA, "user-tls-ca", "/etc/ssl/certs/ca.crt", "Path to CA cert for user server")
	flag.BoolVar(&installRBACRoles, "installrbacroles", false, "Install default RBAC Roles")
	flag.BoolVar(&enableReflection, "enableReflection", true, "Enable reflection")
}

func main() {
	stopCh := signals.SetupSignalHandler()
	ctx := context.Background()
	flag.Parse()
	glog.V(2).Infof("Starting Rbac Service")
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

	// self manage default rbac roles
	if installRBACRoles {
		err := rbacinstaller.Create(ctx, cfg)
		if err != nil {
			glog.Fatalf("Error installing RBAC roles: %s", err.Error())
		}
		glog.V(9).Infof("Successfully installed RBAC Roles")
	}

	namespace := util.GetReleaseNamespace()
	kubeClient, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		glog.Fatalf("Error building kubernetes clientset: %s", err.Error())
	}
	kubeInformerFactory := informers.NewSharedInformerFactoryWithOptions(kubeClient, time.Second*30, informers.WithNamespace(namespace))

	lock, err := util.GetLock("controller-manager-rbac", cfg)
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
				kubeInformerFactory.Start(stopCh)
			},
			OnStoppedLeading: func() {
				// Need to start informer factory since even when not leader to ensure api layer
				// keeps working.
				kubeInformerFactory.Start(stopCh)
			},
			OnNewLeader: func(current_id string) {
				kubeInformerFactory.Start(stopCh)
				if current_id == lock.Identity() {
					glog.Info("currently the leader")
					return
				}
				glog.Infof("current leader is %s", current_id)
			},
		},
	}

	cert, err := tls2.ReadKeyPair(tlsCert, tlsKey)
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
	rs, err := rbacservice.NewGrpcRbacServer(kubeClient, namespace, kubeInformerFactory)
	if err != nil {
		glog.Fatalf("Failed to start rbac grpc server: %s", err)
	}
	rbacProto.RegisterRbacSvcServer(gs, rs)
	if enableReflection {
		reflection.Register(gs)
	}

	go func() {
		glog.Info("grpc rbac server listening on " + grpcPort)
		l, err := net.Listen("tcp", ":"+grpcPort)

		if err != nil {
			glog.Info("Can not serve grpc")
		}
		glog.Fatal(gs.Serve(l))
	}()

	r := mux.NewRouter()
	corsHeaders := handlers.AllowedHeaders([]string{"Authorization", "Content-Type"})
	corsOrigins := handlers.AllowedOrigins([]string{"*"})
	corsMethods := handlers.AllowedMethods([]string{"GET", "POST", "PUT", "HEAD", "OPTIONS", "DELETE"})
	rbacServer := rbacservice.NewRbacServer(rs, tlsCA)
	rbacServer.SetupRoutes(r)
	http.Handle("/", r)
	apiPort := os.Getenv("PORT")
	if apiPort == "" {
		apiPort = "80"
	}
	glog.Info("http rbac server listening on " + apiPort)

	go func() {
		glog.Fatal(http.ListenAndServe(":"+apiPort, handlers.CORS(corsHeaders, corsOrigins, corsMethods)(r)))
	}()

	// Run leader election
	leaderelection.RunOrDie(ctx, leaderElectionConfig)
}
