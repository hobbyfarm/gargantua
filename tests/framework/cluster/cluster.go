package cluster

import (
	"context"
	"k8s.io/client-go/rest"
	"os"
)

// Cluster is the interface wrapper to orchestrate provisioning of a new k3d cluster or using and existing cluster
type Cluster interface {
	Startup(ctx context.Context) (*rest.Config, error)
	Shutdown(ctx context.Context) error
}

// Setup will check env variables and decide if we use existing cluster
// or launch a new k3d cluster for deploying and testing against
func Setup(ctx context.Context) (Cluster, error) {
	useExisting := os.Getenv("USE_EXISTING_CLUSTER")
	if useExisting == "true" {
		return UseExistingCluster(ctx)
	}

	return SetupK3dCluster(ctx)
}
