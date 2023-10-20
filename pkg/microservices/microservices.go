package microservices

import (
	"crypto/tls"
	"crypto/x509"
	"flag"
	"fmt"
	"net"
	"os"
	"time"

	"github.com/golang/glog"
	"github.com/gorilla/handlers"
	hfClientset "github.com/hobbyfarm/gargantua/v3/pkg/client/clientset/versioned"
	tls2 "github.com/hobbyfarm/gargantua/v3/pkg/tls"
	"github.com/hobbyfarm/gargantua/v3/pkg/util"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/keepalive"
	"google.golang.org/grpc/reflection"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

type MicroService string

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
	GrpcPort string = "8080"
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
	return fmt.Sprintf("dns:///%s-grpc.%s.svc.cluster.local:%s", string(svc), util.GetReleaseNamespace(), GrpcPort)
}

/*
/  Used to create new gcpr client with options
*/
func EstablishConnection(svc MicroService, cert credentials.TransportCredentials) (*grpc.ClientConn, error) {
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

	// Add keepalive params to rediscover every x seconds
	return grpc.Dial(
		url,
		grpc.WithTransportCredentials(cert),
		grpc.WithDefaultServiceConfig(grpcServiceConfig),
	)
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

func StartGRPCServer(server *grpc.Server, enableReflection bool) {
	if enableReflection {
		reflection.Register(server)
	}

	grpcPort := os.Getenv("GRPC_PORT")
	if grpcPort == "" {
		grpcPort = GrpcPort
	}

	l, errr := net.Listen("tcp", ":"+grpcPort)
	if errr != nil {
		glog.Fatalf("Can not serve grpc: %v", errr)
	}
	glog.Info("grpc server listening on " + grpcPort)
	glog.Fatal(server.Serve(l))
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
