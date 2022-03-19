package main

import (
	"context"
	"flag"
	"github.com/golang/glog"
	"github.com/hobbyfarm/gargantua/pkg/bootstrap"
	"github.com/hobbyfarm/gargantua/pkg/signals"
	"github.com/hobbyfarm/gargantua/pkg/util"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

var (
	localMasterUrl     string
	localKubeconfig    string
	disableControllers bool
	shellServer        bool
	installCRD         bool
	port               int
)

const (
	DefaultNamespace = "gargantua"
)

func init() {
	flag.StringVar(&localKubeconfig, "kubeconfig", "", "Path to kubeconfig of local cluster. Only required if out-of-cluster.")
	flag.StringVar(&localMasterUrl, "master", "", "The address of the Kubernetes API server. Overrides any value in kubeconfig. Only required if out-of-cluster.")
	flag.BoolVar(&disableControllers, "disablecontrollers", false, "Disable the controllers")
	flag.BoolVar(&shellServer, "shellserver", false, "Be a shell server")
	flag.BoolVar(&installCRD, "installcrd", false, "Install new version of CRD")
	flag.IntVar(&port, "port", 80, "default port to listen on")
}

func main() {
	//var signal chan struct{}
	//signal = make(chan struct{})
	flag.Parse()
	stopCh := signals.SetupSignalHandler()
	ctx := context.Background()

	cfg, err := rest.InClusterConfig()

	if err != nil {
		cfg, err = clientcmd.BuildConfigFromFlags(localMasterUrl, localKubeconfig)
		if err != nil {
			glog.Fatalf("Error building kubeconfig: %s", err.Error())
		}
	}

	// default release namespace is gargantua. To override set env variable HF_NAMESPACE
	s := bootstrap.NewServer(cfg, disableControllers, shellServer, installCRD, port, util.GetReleaseNamespace())
	s.Start(ctx, stopCh)
}
