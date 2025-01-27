package config

import "github.com/spf13/viper"

const (
	RemoteK8sNamespace      = "remote-k8s-namespace"
	Kubeconfig              = "kubeconfig"
	Context                 = "context"
	SkipCRDInstallation     = "skip-crd-installation"
	JWTSigningKeySecretName = "jwt-signing-key-secret-name"
	JWTSigningKeySecretKey  = "jwt-signing-key-secret-key"
)

func init() {
	// Namespace in the remote k8s cluster in which to store objects
	viper.SetDefault(RemoteK8sNamespace, "hobbyfarm")

	// Path to kubeconfig file
	viper.SetDefault(Kubeconfig, "")

	// Context in the kubeconfig file to use
	viper.SetDefault(Context, "default")

	// Whether to skip CRD installation into the remote k8s cluster
	viper.SetDefault(SkipCRDInstallation, false)

	// The name of the secret that contains the jwt signing key
	viper.SetDefault(JWTSigningKeySecretName, "jwt-signing-key")

	// The key in the data portion of the secret that contains the jwt signing key
	viper.SetDefault(JWTSigningKeySecretKey, "key")
}
