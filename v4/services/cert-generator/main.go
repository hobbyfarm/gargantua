package main

import (
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"github.com/hobbyfarm/gargantua/v4/pkg/authentication/user"
	"github.com/hobbyfarm/gargantua/v4/pkg/certs"
	"github.com/spf13/cobra"
	"log"
	"os"
	"path/filepath"
	"strings"
)

var (
	outputdirectory string
)

func init() {
	rootCmd.Flags().StringVarP(&outputdirectory, "outputdirectory", "o", ".", "output directory")
}

var rootCmd = &cobra.Command{
	Use:   "cert-generator",
	Short: "generate certificates for hobbyfarm apiserver and core services",
	RunE:  app,
}

func app(cmd *cobra.Command, args []string) error {
	cert, key, err := certs.GenerateHFCACertificate()
	if err != nil {
		return err
	}

	if err = os.WriteFile(filepath.Join(outputdirectory, "hf-ca-cert.pem"), []byte(cert), 0644); err != nil {
		return err
	}

	if err = os.WriteFile(filepath.Join(outputdirectory, "hf-ca-key.pem"), []byte(key), 0644); err != nil {
		return err
	}

	// ugh
	certBlock, _ := pem.Decode(cert)
	keyBlock, _ := pem.Decode(key)

	caKey, _ := x509.ParsePKCS1PrivateKey(keyBlock.Bytes)
	caCert, _ := x509.ParseCertificate(certBlock.Bytes)

	for _, u := range user.WellKnownUsers {
		userCert, userKey, err := certs.SignAuthCertificate(u, []string{user.SuperuserGroup}, caCert, caKey)
		if err != nil {
			return err
		}

		outPath := removeColonsFilepath(filepath.Join(outputdirectory, u+"-cert.pem"))
		if err = os.WriteFile(outPath, userCert, 0644); err != nil {
			return err
		}

		outPath = removeColonsFilepath(filepath.Join(outputdirectory, u+"-key.pem"))
		if err = os.WriteFile(outPath, userKey, 0644); err != nil {
			return err
		}
	}

	fmt.Println("certificates generated")

	return nil
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		log.Fatal(err)
	}
}

func removeColonsFilepath(path string) string {
	return strings.Replace(path, ":", "-", -1)
}
