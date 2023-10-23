package microservices

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"flag"
	"fmt"
	"net"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/golang/glog"
	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	hfClientset "github.com/hobbyfarm/gargantua/v3/pkg/client/clientset/versioned"
	tls2 "github.com/hobbyfarm/gargantua/v3/pkg/tls"
	"github.com/hobbyfarm/gargantua/v3/pkg/util"
	"google.golang.org/grpc"
	"google.golang.org/grpc/connectivity"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/keepalive"
	"google.golang.org/grpc/reflection"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

// Add type MicroService based on string that is used to define constants for every service
type MicroService string

// Interface for http APIServers that share the common method "SetupRoutes".
type APIServer interface {
	SetupRoutes(r *mux.Router)
}

// ServiceConfig holds the configuration for a service
type ServiceConfig struct {
	TLSCert          string
	TLSKey           string
	TLSCA            string
	WebhookTLSCA     string
	LocalMasterUrl   string
	LocalKubeconfig  string
	EnableReflection bool
	ClientCert       credentials.TransportCredentials
	ServerCert       credentials.TransportCredentials
}

const (
	AuthN      MicroService = "authn-service"
	AuthR      MicroService = "authr-service"
	User       MicroService = "user-service"
	Rbac       MicroService = "rbac-service"
	AccessCode MicroService = "accesscode-service"
	Setting    MicroService = "setting-service"
)

const (
	defaultGrpcPort          string        = "8080"
	defaultApiPort           string        = "80"
	InitialConnectionTimeout time.Duration = 30 * time.Second
)

var CORS_ALLOWED_METHODS_ALL = [...]string{"GET", "POST", "PUT", "HEAD", "OPTIONS", "DELETE"}
var CORS_ALLOWED_ORIGINS = []string{"*"}
var CORS_ALLOWED_HEADERS = []string{"Authorization", "Content-Type"}

var CORS_HANDLER_ALLOWED_HEADERS = handlers.AllowedHeaders(CORS_ALLOWED_HEADERS)
var CORS_HANDLER_ALLOWED_ORIGINS = handlers.AllowedOrigins(CORS_ALLOWED_ORIGINS)
var CORS_HANDLER_ALLOWED_METHODS = handlers.AllowedMethods(CORS_ALLOWED_METHODS_ALL[:])

func (svc MicroService) getGRPCUrl() string {
	// Builds the connection string for the headless service.
	// Service for a grpc microservice has to be named <service>-grpc and must be headless (set .spec.ClusterIP: None)
	// Most important is the dns:/// part that leads the grpc resolver to discover multiple addresses, otherwise only the first address is used
	return fmt.Sprintf("dns:///%s-grpc.%s.svc.cluster.local:%s", string(svc), util.GetReleaseNamespace(), defaultGrpcPort)
}

/*
/  Used to create new gcpr client with options
*/
func EstablishConnection(svc MicroService, cert credentials.TransportCredentials) *grpc.ClientConn {
	url := svc.getGRPCUrl()

	// - Using round_robin loadBalancingConfig to target all of the backend services.
	// - The empty service field inside the name block for the methodConfig states that this is the default methodConfig used for all services and methods
	// - waitForReady states that the service has to be ready before event attempting to send
	//
	// All available options can be found here: https://github.com/grpc/grpc-proto/blob/master/grpc/service_config/service_config.proto
	// More about retryPolicy: https://github.com/grpc/proposal/blob/master/A6-client-retries.md
	const grpcServiceConfig = `
	{
		"loadBalancingConfig": [
			 { "round_robin": {} }
		],
		"methodConfig": 
		[
			{
				"name": [ { "service" : "" } ],
				"waitForReady": true,
				"retryPolicy": {
					"MaxAttempts": 5,
					"InitialBackoff": "0.1s",
					"MaxBackoff": "1s",
					"BackoffMultiplier": 2,
					"RetryableStatusCodes": [ "UNAVAILABLE" ]
				}
			}
		]
	}`

	// With the given context ctx, an error is thrown when the timeout is reached.
	conn, err := grpc.Dial(
		url,
		grpc.WithTransportCredentials(cert),
		grpc.WithDefaultServiceConfig(grpcServiceConfig),
	)
	if err != nil {
		glog.Fatalf("Failed to conect to service %s within %v seconds.", svc, InitialConnectionTimeout)
	}

	glog.Infof("Connection to %s is now in state %s", svc, conn.GetState())
	return conn
}

func EstablishConnections(services []MicroService, cert credentials.TransportCredentials) map[MicroService]*grpc.ClientConn {
	connections := make(map[MicroService]*grpc.ClientConn)
	var connWait sync.WaitGroup
	var connMutex sync.Mutex // protect connections map

	for _, svc := range services {
		connWait.Add(1) // Add a delta to the WaitGroup counter
		go func(microservice MicroService) {
			defer connWait.Done()
			connMutex.Lock() // Securing access against concurrent write operations
			connections[microservice] = EstablishConnection(microservice, cert)
			connMutex.Unlock()
		}(svc)
	}

	connWait.Wait()

	glog.Infof("Connections to %d services have been created.", len(connections))

	for svc, conn := range connections {
		ConnectionWatchdog(svc, conn)
	}

	return connections
}

/*
Watchdog for grpc Connection that logs state changes
Only log changes if
- moving to TransientFailure or Shutdown
- moving to any other state when there was an Error (this happens until the error is resolved and the state = READY)

Second watchdog tries to catch initial connection errors
*/
func ConnectionWatchdog(svc MicroService, conn *grpc.ClientConn) {
	glog.Infof("Starting Watchdog for connection to service %s", svc)
	hadError := false //The connection had an error in the last time, resolved by state READY
	wasReady := false //If the connection was ready at one point
	go func() {
		for {
			state := conn.GetState()
			switch state {
			case connectivity.Ready:
				if !wasReady {
					glog.Infof("Connection to %s is now in state %s", svc, state)
				}
				if hadError && !wasReady {
					glog.Infof("Resolved connection to %s (State %s)", svc, state)
					hadError = false
				}
				wasReady = true
			case connectivity.TransientFailure, connectivity.Shutdown:
				hadError = true
				fallthrough
			default:
				if hadError && wasReady {
					glog.Infof("Connection to %s is now in state %s", svc, state)
				}
			}
			//glog.V(8).Infof("Connection to %s is now in state %s", svc, state) // Enable this if log levels are used correctly
			conn.WaitForStateChange(context.Background(), state) // Wait for the next state change
		}
	}()

	go func() {
		time.Sleep(InitialConnectionTimeout)
		if conn.GetState() != connectivity.Ready && !wasReady {
			glog.Fatalf("Connection %s never came ready. Aborting after %v seconds", svc, InitialConnectionTimeout)
		}
	}()
}

/*
/  Used to create new gcpr server with options
*/
func CreateGRPCServer(c credentials.TransportCredentials) *grpc.Server {
	// MaxConnectionAge needed for trigger service rediscovery on client when scaling up instances
	// MaxConnectionAgeGrace defines the grace period for outstanding connections to complete
	// All keepalive options can be found here: https://grpc.io/docs/guides/keepalive/#keepalive-configuration-specification
	opts := []grpc.ServerOption{
		grpc.KeepaliveParams(keepalive.ServerParameters{
			MaxConnectionAge:      30 * time.Second,
			MaxConnectionAgeGrace: 10 * time.Second,
		}),
		grpc.Creds(c),
	}

	return grpc.NewServer(opts...)
}

/*
Common method that starts a microservices grpc server
*/
func StartGRPCServer(server *grpc.Server, enableReflection bool) {
	if enableReflection {
		reflection.Register(server)
	}

	grpcPort := os.Getenv("GRPC_PORT")
	if grpcPort == "" {
		grpcPort = defaultGrpcPort
	}

	l, errr := net.Listen("tcp", ":"+grpcPort)
	if errr != nil {
		glog.Fatalf("Can not serve grpc: %v", errr)
	}
	glog.Info("grpc server listening on " + grpcPort)
	glog.Fatal(server.Serve(l))
}

/*
Common method that starts a microservices API server
*/
func StartAPIServer(server APIServer) {
	r := mux.NewRouter()

	server.SetupRoutes(r)
	http.Handle("/", r)

	apiPort := os.Getenv("PORT")
	if apiPort == "" {
		apiPort = defaultApiPort
	}

	glog.Infof("http server listening on port %s", apiPort)
	glog.Fatal(http.ListenAndServe(":"+apiPort, handlers.CORS(CORS_HANDLER_ALLOWED_HEADERS, CORS_HANDLER_ALLOWED_METHODS, CORS_HANDLER_ALLOWED_ORIGINS)(r)))

}

func BuildTLSClientCredentials(caPath string) (credentials.TransportCredentials, error) {
	// Read the CA certificate from file
	caCert, err := os.ReadFile(caPath)
	if err != nil {
		glog.Errorf("could not establish connection, failed to load CA certificate: %s", err)
		return nil, fmt.Errorf("could not establish connection, failed to load CA certificate: %s", err)
	}
	caCertPool, err := x509.SystemCertPool()
	if err != nil {
		caCertPool = x509.NewCertPool()
	}
	if ok := caCertPool.AppendCertsFromPEM(caCert); !ok {
		glog.Error("could not establish connection, failed to parse CA certificate")
		return nil, fmt.Errorf("could not establish connection, failed to parse CA certificate")
	}

	// Create transport credentials with the client certificate and CA certificate pool
	creds := credentials.NewTLS(&tls.Config{
		RootCAs: caCertPool,
	})

	return creds, nil
}

func buildTLSServerCredentials(certPath string, keyPath string) (credentials.TransportCredentials, error) {
	certKeyPair, err := tls2.ReadKeyPair(certPath, keyPath)
	if err != nil {
		return nil, err
	}
	return credentials.NewTLS(&tls.Config{
		Certificates: []tls.Certificate{*certKeyPair},
	}), nil
}

func BuildClusterConfig(serviceConfig *ServiceConfig) (*rest.Config, *hfClientset.Clientset) {
	const (
		ClientGoQPS   = 100
		ClientGoBurst = 100
	)
	cfg, err := rest.InClusterConfig()
	if err != nil {
		cfg, err = clientcmd.BuildConfigFromFlags(serviceConfig.LocalMasterUrl, serviceConfig.LocalKubeconfig)
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

	return cfg, hfClient
}

// ParseFlags declares the flags and parses them, then returns a ServiceConfig struct.
func BuildServiceConfig() *ServiceConfig {
	cfg := &ServiceConfig{}

	flag.StringVar(&cfg.TLSCert, "tls-cert", "/etc/ssl/certs/tls.crt", "Path to TLS certificate for this server")
	flag.StringVar(&cfg.TLSKey, "tls-key", "/etc/ssl/certs/tls.key", "Path to TLS key for this server")
	flag.StringVar(&cfg.TLSCA, "tls-ca", "/etc/ssl/certs/ca.crt", "Path to CA cert for this server")
	flag.StringVar(&cfg.WebhookTLSCA, "webhook-ca", "/webhook-secret/ca.crt", "Path to Webhook CA")
	flag.StringVar(&cfg.LocalKubeconfig, "kubeconfig", "", "Path to kubeconfig of local cluster. Only required if out-of-cluster.")
	flag.StringVar(&cfg.LocalMasterUrl, "master", "", "The address of the Kubernetes API server. Overrides any value in kubeconfig. Only required if out-of-cluster.")
	flag.BoolVar(&cfg.EnableReflection, "enableReflection", true, "Enable reflection")

	flag.Parse()

	serverCert, err := buildTLSServerCredentials(cfg.TLSCert, cfg.TLSKey)
	if err != nil {
		glog.Fatalf("error building server cert: %v", err)
	}

	cert, err := BuildTLSClientCredentials(cfg.TLSCA)
	if err != nil {
		glog.Fatalf("error building client cert: %v", err)
	}

	cfg.ClientCert = cert
	cfg.ServerCert = serverCert

	return cfg
}
