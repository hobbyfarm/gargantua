package tls

import (
	"crypto/tls"
	"os"
)

func ReadKeyPair(certPath string, keyPath string) (*tls.Certificate, error) {
	cert, err := os.ReadFile(certPath)
	if err != nil {
		return nil, err
	}

	key, err := os.ReadFile(keyPath)
	if err != nil {
		return nil, err
	}

	crt, err := tls.X509KeyPair(cert, key)
	return &crt, err
}
