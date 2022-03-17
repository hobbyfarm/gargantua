package cluster

import (
	"context"
	"github.com/hobbyfarm/gargantua/tests/framework/setup"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
	"os"
	"path/filepath"
)

type ExistingCluster struct {
	config *rest.Config
	keep   bool
}

func UseExistingCluster(ctx context.Context) (Cluster, error) {

	configFile := os.Getenv("KUBECONFIG")
	if configFile == "" {
		configFile = filepath.Join(homedir.HomeDir(), ".kube", "config")
	}

	config, err := clientcmd.BuildConfigFromFlags("", configFile)
	if err != nil {
		return nil, err
	}
	e := &ExistingCluster{config: config}
	keepCluster := os.Getenv("KEEP_CLUSTER")
	if keepCluster == "true" {
		e.keep = true
	}

	return e, nil
}

func (e *ExistingCluster) Startup(ctx context.Context) (*rest.Config, error) {
	return e.config, nil
}

func (e *ExistingCluster) Shutdown(ctx context.Context) error {
	// uninstall CRDs
	if !e.keep {
		k, err := kubernetes.NewForConfig(e.config)
		if err != nil {
			return err
		}

		err = k.CoreV1().Namespaces().Delete(ctx, setup.DefaultNamespace, metav1.DeleteOptions{})
		return err
	}

	return nil
}
