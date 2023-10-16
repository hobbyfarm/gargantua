package microservices

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"os"
	"strconv"

	"github.com/golang/glog"
	"github.com/hobbyfarm/gargantua/v3/pkg/util"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

type MicroService string

const (
	AuthN MicroService 		= "authn-service"
	AuthR MicroService 		= "authr-service"
	User MicroService 		= "user-service"
	Rbac MicroService 		= "rbac-service"
	AccessCode MicroService	= "accesscode-service"
	Setting MicroService 	= "setting-service"
  )

func (svc MicroService) getUrl() (string, error) {
	// Create a Kubernetes clientset
	config, err := rest.InClusterConfig()
	if err != nil {
		return "", fmt.Errorf("error retrieving InClusterConfig")
	}
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return "", fmt.Errorf("error creating kubernetes clientset")
	}

	ctx := context.Background()

	// Get the endpoints object for the service
	endpoints, err := clientset.CoreV1().Endpoints(util.GetReleaseNamespace()).Get(ctx, string(svc), metav1.GetOptions{})
	if err != nil {
		return "", fmt.Errorf("error retrieving endpoints")
	}

	var grpcPort int32 = 0

	// Get the grpc port number
	for _, subset := range endpoints.Subsets {
		for _, port := range subset.Ports {
			if port.Name == "grpc" {
				grpcPort = port.Port
			}
		}
	}

	if grpcPort == 0 {
		return "", fmt.Errorf("no grpc port found for service %s", svc)
	}

	return string(svc) + "." + util.GetReleaseNamespace() + ".svc.cluster.local:" + strconv.Itoa(int(grpcPort)), nil
}

func EstablishConnection(svc MicroService, caCertPath string) (*grpc.ClientConn, error) {
	url, err := svc.getUrl()
	if err != nil {
		glog.Errorf("could not establish connection, failed to retrieve url for service %s", svc)
		return nil, fmt.Errorf("could not establish connection, failed to retrieve url for service %s", svc)
	}
	// Read the CA certificate from file
	caCert, err := os.ReadFile(caCertPath)
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
	return grpc.Dial(url, grpc.WithTransportCredentials(creds))
}
