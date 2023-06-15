package microservices

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"os"
	"strconv"

	"github.com/golang/glog"
	"github.com/hobbyfarm/gargantua/pkg/util"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

type MicroService struct {
	name string
}

var (
	AuthN   = MicroService{"authn-service"}
	AuthR   = MicroService{"authr-service"}
	User    = MicroService{"user-service"}
	Rbac    = MicroService{"rbac-service"}
	Unknown = MicroService{""}
)

func (m MicroService) getUrl() (string, error) {
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
	endpoints, err := clientset.CoreV1().Endpoints(util.GetReleaseNamespace()).Get(ctx, m.name, metav1.GetOptions{})
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
		return "", fmt.Errorf("no grpc port found for service %s", m.name)
	}

	// if port < 1 || port > 65535 {
	// 	return "", errors.New("Invalid port(" + strconv.Itoa(port) + ")! Port must be within the range 1 to 65535")
	// }
	return "dns://" + util.GetK8sDnsServer() + "/" + m.name + "." + util.GetReleaseNamespace() + ".svc.cluster.local:" + strconv.Itoa(int(grpcPort)), nil
}

func createMicroService(service string) (MicroService, error) {
	switch service {
	case AuthN.name:
		return AuthN, nil
	case AuthR.name:
		return AuthR, nil
	case User.name:
		return User, nil
	case Rbac.name:
		return Rbac, nil
	}

	return Unknown, fmt.Errorf("unknown service: %s", service)
}

func EstablishConnection(svcName string, caCertPath string) (*grpc.ClientConn, error) {
	svc, err := createMicroService(svcName)
	if err != nil {
		glog.Errorf("failed to create microservice %s", svcName)
		return nil, err
	}
	url, err := svc.getUrl()

	if err != nil {
		glog.Errorf("could not establish connection, failed to retrieve url for service %s", svc.name)
		return nil, fmt.Errorf("could not establish connection, failed to retrieve url for service %s", svc.name)
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
