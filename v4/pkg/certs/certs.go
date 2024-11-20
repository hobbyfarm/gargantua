package certs

import (
	"bytes"
	cryptorand "crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"math/big"
	"math/rand"
	"net"
	"time"
)

func GenerateHFCACertificate() (cert []byte, key []byte, err error) {
	ca := &x509.Certificate{
		SerialNumber: big.NewInt(rand.Int63()),
		Subject: pkix.Name{
			Organization: []string{"hobbyfarm"},
		},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().AddDate(10, 0, 0),
		IsCA:                  true,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth, x509.ExtKeyUsageServerAuth},
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign,
		BasicConstraintsValid: true,
	}

	caPrivKey, err := rsa.GenerateKey(cryptorand.Reader, 4096)
	if err != nil {
		return nil, nil, err
	}

	caBytes, err := x509.CreateCertificate(cryptorand.Reader, ca, ca, &caPrivKey.PublicKey, caPrivKey)
	if err != nil {
		return nil, nil, err
	}

	caBuffer := new(bytes.Buffer)
	if err = pem.Encode(caBuffer, &pem.Block{
		Type:  "CERTIFICATE",
		Bytes: caBytes,
	}); err != nil {
		return nil, nil, err
	}

	keyBuffer := new(bytes.Buffer)
	if err = pem.Encode(keyBuffer, &pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(caPrivKey),
	}); err != nil {
		return nil, nil, err
	}

	return caBuffer.Bytes(), keyBuffer.Bytes(), nil
}

func SignServingCertificate(commonName string, dnsNames []string, ips []net.IP, ca *x509.Certificate, privKey *rsa.PrivateKey) (cert []byte, key []byte, err error) {
	crt := &x509.Certificate{
		SerialNumber: big.NewInt(rand.Int63()),
		Subject: pkix.Name{
			Organization: []string{"hobbyfarm"},
			CommonName:   commonName,
		},
		NotBefore:   time.Now(),
		NotAfter:    time.Now().AddDate(10, 0, 0),
		ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth, x509.ExtKeyUsageServerAuth},
		KeyUsage:    x509.KeyUsageDigitalSignature,
		DNSNames:    dnsNames,
		IPAddresses: ips,
	}

	return signCert(crt, ca, privKey)
}

func SignAuthCertificate(username string, groups []string, ca *x509.Certificate, privKey *rsa.PrivateKey) (cert []byte, key []byte, err error) {
	crt := &x509.Certificate{
		SerialNumber: big.NewInt(rand.Int63()),
		Subject: pkix.Name{
			Organization: groups,
			CommonName:   username,
		},
		NotBefore:   time.Now(),
		NotAfter:    time.Now().AddDate(10, 0, 0),
		ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth, x509.ExtKeyUsageServerAuth},
		KeyUsage:    x509.KeyUsageDigitalSignature,
	}

	return signCert(crt, ca, privKey)
}

func signCert(certificate *x509.Certificate, caCert *x509.Certificate, caPrivKey *rsa.PrivateKey) (cert []byte, key []byte, err error) {
	privateKey, err := rsa.GenerateKey(cryptorand.Reader, 4096)
	if err != nil {
		return nil, nil, err
	}

	certBytes, err := x509.CreateCertificate(cryptorand.Reader, certificate, caCert, &privateKey.PublicKey, caPrivKey)
	if err != nil {
		return nil, nil, err
	}

	certPem, err := pemEncode(certBytes)
	if err != nil {
		return nil, nil, err
	}

	keyPem, err := pemEncode(privateKey)
	if err != nil {
		return nil, nil, err
	}

	return certPem, keyPem, nil
}

func pemEncode(in any) ([]byte, error) {
	buf := new(bytes.Buffer)

	switch typedIn := in.(type) {
	case *rsa.PrivateKey:
		if err := pem.Encode(buf, &pem.Block{
			Type:  "RSA PRIVATE KEY",
			Bytes: x509.MarshalPKCS1PrivateKey(typedIn),
		}); err != nil {
			return nil, err
		}
	case []byte:
		if err := pem.Encode(buf, &pem.Block{
			Type:  "CERTIFICATE",
			Bytes: typedIn,
		}); err != nil {
			return nil, err
		}
	default:
		return nil, fmt.Errorf("unsupported type: %T", typedIn)
	}

	return buf.Bytes(), nil
}
