package microservices

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
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
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

type MicroService string

const (
	AuthN      MicroService = "authn-service"
	AuthR      MicroService = "authr-service"
	User       MicroService = "user-service"
	Rbac       MicroService = "rbac-service"
	AccessCode MicroService = "accesscode-service"
	Setting    MicroService = "setting-service"
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
	return fmt.Sprintf("dns:///%s-grpc.%s.svc.cluster.local:%s", string(svc), util.GetReleaseNamespace(), "8080")
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

func BuildTLSCredentials(caPath string, certPath string, keyPath string) (credentials.TransportCredentials, error) {
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

	// Certificate is not always needed, so omit it.
	if certPath != "" && keyPath != "" {
		keyPair, err := tls2.ReadKeyPair(certPath, keyPath)
		if err != nil {
			glog.Fatalf("error generating x509keypair from conversion cert and key: %s", err)
		}
		creds = credentials.NewTLS(&tls.Config{
			RootCAs:      caCertPool,
			Certificates: []tls.Certificate{*keyPair},
		})
	}

	return creds, nil
}

func BuildClusterConfig(localMasterUrl string, localKubeconfig string) (*rest.Config, *hfClientset.Clientset) {
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

	return cfg, hfClient
}
