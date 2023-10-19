package main

import (
	"context"
	"flag"
	"net"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/golang/glog"
	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	rbacservice "github.com/hobbyfarm/gargantua/services/rbacsvc/v3/internal"
	rbacinstaller "github.com/hobbyfarm/gargantua/services/rbacsvc/v3/internal/rbac"
	"github.com/hobbyfarm/gargantua/v3/pkg/microservices"
	"github.com/hobbyfarm/gargantua/v3/pkg/signals"
	"github.com/hobbyfarm/gargantua/v3/pkg/util"
	"github.com/hobbyfarm/gargantua/v3/protos/authn"
	"github.com/hobbyfarm/gargantua/v3/protos/authr"
	rbacProto "github.com/hobbyfarm/gargantua/v3/protos/rbac"
	"google.golang.org/grpc/reflection"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
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
	flag.Parse()
	glog.V(2).Infof("Starting Rbac Service")
	cfg, _ := microservices.BuildClusterConfig(localMasterUrl, localKubeconfig)

	// self manage default rbac roles
	if installRBACRoles {
		ctx := context.Background()
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

	cert, err := microservices.BuildTLSCredentials(tlsCA, tlsCert, tlsKey)
	if err != nil {
		glog.Fatalf("error building cert: %v", err)
	}

	gs := microservices.CreateGRPCServer(cert)
	rs, err := rbacservice.NewGrpcRbacServer(kubeClient, namespace, kubeInformerFactory)
	if err != nil {
		glog.Fatalf("Failed to start rbac grpc server: %s", err)
	}
	rbacProto.RegisterRbacSvcServer(gs, rs)
	if enableReflection {
		reflection.Register(gs)
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

	var wg sync.WaitGroup
	wg.Add(1)

	go func() {
		defer wg.Done()
		grpcPort := os.Getenv("GRPC_PORT")
		if grpcPort == "" {
			grpcPort = "8080"
		}
		l, err := net.Listen("tcp", ":"+grpcPort)
		if err != nil {
			glog.Fatalf("Can not serve grpc: %v", err)
		}
		glog.Info("grpc rbac server listening on " + grpcPort)
		glog.Fatal(gs.Serve(l))
	}()

	go func() {
		defer wg.Done()
		r := mux.NewRouter()
		rbacServer := rbacservice.NewRbacServer(rs, authnClient, authrClient)
		rbacServer.SetupRoutes(r)
		http.Handle("/", r)
		apiPort := os.Getenv("PORT")
		if apiPort == "" {
			apiPort = "80"
		}

		glog.Info("http rbac server listening on " + apiPort)
		glog.Fatal(http.ListenAndServe(":"+apiPort, handlers.CORS(microservices.CORS_HANDLER_ALLOWED_HEADERS, microservices.CORS_HANDLER_ALLOWED_METHODS, microservices.CORS_HANDLER_ALLOWED_ORIGINS)(r)))
	}()

	stopCh := signals.SetupSignalHandler()
	kubeInformerFactory.Start(stopCh)
	wg.Wait()
}
