package cert

import (
	"crypto/x509"
	kx509 "k8s.io/apiserver/pkg/authentication/request/x509"
	"k8s.io/client-go/util/cert"
)

func NewCertAuthenticator(caCertBundle string) (*kx509.Authenticator, error) {
	pool, err := cert.NewPool(caCertBundle)
	if err != nil {
		return nil, err
	}

	return kx509.New(x509.VerifyOptions{
		Roots: pool,
	}, kx509.CommonNameUserConversion), nil
}
