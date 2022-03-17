package cluster

import (
	"context"
	"fmt"
	"github.com/docker/go-connections/nat"
	cliutil "github.com/rancher/k3d/v5/cmd/util"
	k3dCluster "github.com/rancher/k3d/v5/pkg/client"
	"github.com/rancher/k3d/v5/pkg/config"
	conf "github.com/rancher/k3d/v5/pkg/config/v1alpha4"
	"github.com/rancher/k3d/v5/pkg/runtimes"
	"github.com/rancher/k3d/v5/pkg/types"
	"github.com/rancher/wrangler/pkg/yaml"
	"github.com/sirupsen/logrus"
	"io/ioutil"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"os"
	"strconv"
	"time"
)

type K3dCluster struct {
	config        *rest.Config
	keep          bool
	clusterConfig *conf.ClusterConfig
	name          string
}

const (
	defaultConfig = `# k3d configuration file
apiVersion: k3d.io/v1alpha4 
kind: Simple 
metadata:
  name: gargantua-integration 
servers: 1
image: rancher/k3s:v1.20.4-k3s1
`
)

func SetupK3dCluster(ctx context.Context) (Cluster, error) {
	var k3dConfig string
	k := &K3dCluster{}
	k3dTemplate := os.Getenv("USE_K3D_CONFIG")
	if k3dTemplate != "" {
		configFile, err := ioutil.ReadFile(k3dTemplate)
		if err != nil {
			return nil, err
		}
		k3dConfig = string(configFile)
	} else {
		k3dConfig = defaultConfig
	}

	keepCluster := os.Getenv("KEEP_CLUSTER")
	if keepCluster == "true" {
		k.keep = true
	}
	simpleCfg := &conf.SimpleConfig{}
	err := yaml.Unmarshal([]byte(k3dConfig), simpleCfg)
	if err != nil {
		return nil, err
	}

	applyDefaults(simpleCfg)

	k.name = simpleCfg.Name
	clusterConfig, err := config.TransformSimpleToClusterConfig(ctx, runtimes.SelectedRuntime, *simpleCfg)
	if err != nil {
		return nil, err
	}

	if err := config.ValidateClusterConfig(ctx, runtimes.SelectedRuntime, *clusterConfig); err != nil {
		return nil, err
	}

	clusterConfig, err = config.ProcessClusterConfig(*clusterConfig)
	if err != nil {
		return nil, err
	}

	if err := config.ValidateClusterConfig(ctx, runtimes.SelectedRuntime, *clusterConfig); err != nil {
		return nil, err
	}

	k.clusterConfig = clusterConfig

	return k, nil
}

func (c *K3dCluster) Startup(ctx context.Context) (*rest.Config, error) {
	logrus.Infof("checking if cluster %s exists", c.clusterConfig.Cluster.Name)
	if _, err := k3dCluster.ClusterGet(ctx, runtimes.SelectedRuntime, &c.clusterConfig.Cluster); err == nil {
		return nil, fmt.Errorf("failed to create cluster %s because a cluster with that name already exists", c.clusterConfig.Cluster.Name)
	}
	if err := k3dCluster.ClusterRun(ctx, runtimes.SelectedRuntime, c.clusterConfig); err != nil {
		logrus.Info("cluster creation failed, trying to roll back")
		if err := k3dCluster.ClusterDelete(ctx, runtimes.SelectedRuntime, &c.clusterConfig.Cluster, types.ClusterDeleteOpts{SkipRegistryCheck: true}); err != nil {
			return nil, err
		}
	}

	// wait for cluster to be ready

	cluster, err := k3dCluster.ClusterGet(ctx, runtimes.SelectedRuntime, &c.clusterConfig.Cluster)
	if err != nil {
		return nil, fmt.Errorf("error querying cluster state %v", err)
	}

	keepChecking := true
	for keepChecking {
		nodesReady := false
		for _, node := range cluster.Nodes {
			nodesReady = node.State.Running
		}

		time.Sleep(10 * time.Second)
		if nodesReady {
			break
		}
	}

	file, err := ioutil.TempFile("/tmp", "kc")
	if err != nil {
		return nil, fmt.Errorf("error creating a tmp file %v", err)
	}
	defer os.Remove(file.Name())

	if _, err := k3dCluster.KubeconfigGetWrite(ctx, runtimes.SelectedRuntime, &c.clusterConfig.Cluster, file.Name(), &k3dCluster.WriteKubeConfigOptions{UpdateExisting: true, OverwriteExisting: true}); err != nil {
		return nil, fmt.Errorf("error fetching and updating kubeconfig at %s with error: %v", file.Name(), err)
	}
	// cluster create. fetch kubeconfig and return *rest.Config
	restConfig, err := clientcmd.BuildConfigFromFlags("", file.Name())
	return restConfig, err
}

func (c *K3dCluster) Shutdown(ctx context.Context) error {
	var err error
	if !c.keep {
		err = k3dCluster.ClusterDelete(ctx, runtimes.SelectedRuntime,
			&types.Cluster{Name: c.name}, types.ClusterDeleteOpts{SkipRegistryCheck: true})
	}

	return err
}

// expose api server via a random port if one is not specified
func applyDefaults(cfg *conf.SimpleConfig) {

	exposeAPI := &types.ExposureOpts{
		PortMapping: nat.PortMapping{
			Binding: nat.PortBinding{
				HostIP:   cfg.ExposeAPI.HostIP,
				HostPort: cfg.ExposeAPI.HostPort,
			},
		},
		Host: cfg.ExposeAPI.Host,
	}

	// Set to random port if port is empty string
	if len(exposeAPI.Binding.HostPort) == 0 {
		var freePort string
		port, err := cliutil.GetFreePort()
		freePort = strconv.Itoa(port)
		if err != nil || port == 0 {
			logrus.Warnf("Failed to get random free port: %+v", err)
			logrus.Warnf("Falling back to internal port %s (may be blocked though)...", types.DefaultAPIPort)
			freePort = types.DefaultAPIPort
		}
		exposeAPI.Binding.HostPort = freePort
	}

	cfg.ExposeAPI = conf.SimpleExposureOpts{
		Host:     exposeAPI.Host,
		HostIP:   exposeAPI.Binding.HostIP,
		HostPort: exposeAPI.Binding.HostPort,
	}

}
