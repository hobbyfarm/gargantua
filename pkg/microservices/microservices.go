package microservices

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"os"

	"github.com/golang/glog"
	"github.com/hobbyfarm/gargantua/v3/pkg/util"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
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

func (svc MicroService) getGRPCUrl() string {
	return string(svc) + "." + util.GetReleaseNamespace() + ".svc.cluster.local:8080"
}

func EstablishConnection(svc MicroService, caCertPath string) (*grpc.ClientConn, error) {
	url := svc.getGRPCUrl()

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
