package main

import (
	"fmt"
	"github.com/ebauman/crder"
	"github.com/hobbyfarm/gargantua/v4/pkg/crd"
	"github.com/hobbyfarm/gargantua/v4/pkg/scheme"
	server2 "github.com/hobbyfarm/gargantua/v4/pkg/server"
	"github.com/spf13/cobra"
	"k8s.io/client-go/tools/clientcmd"
	"log"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var (
	kubeconfig     string
	kubecontext    string
	skipcrdinstall bool
	namespace      string
)

func init() {
	rootCmd.Flags().StringVar(&kubeconfig, "kubeconfig", "", "path to kubeconfig file, uses in-cluster if not set")
	rootCmd.Flags().StringVar(&kubecontext, "context", "default", "kube context")
	rootCmd.Flags().BoolVar(&skipcrdinstall, "skip-crd-installation", false, "skip installation of CRDs into remote cluster")
	rootCmd.Flags().StringVar(&namespace, "namespace", "hobbyfarm", "namespace in which to store objects in remote cluster")
}

var rootCmd = &cobra.Command{
	Use:   "apiserver",
	Short: "run apiserver for hobbyfarm",
	RunE:  app,
}

func app(cmd *cobra.Command, args []string) error {
	cfg, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
	if err != nil {
		return fmt.Errorf("could not connect to kubernetes cluster: %v", err.Error())
	}

	kClient, err := client.NewWithWatch(cfg, client.Options{
		Scheme: scheme.Scheme,
	})
	if err != nil {
		return fmt.Errorf("could not build client: %v", err.Error())
	}

	if !skipcrdinstall {
		crds := crd.GenerateCRDs()
		if err := crder.InstallUpdateCRDs(cfg, crds...); err != nil {
			return fmt.Errorf("error installing/updating crds: %v", err.Error())
		}
	}

	server, err := server2.NewKubernetesServer(kClient, namespace)
	if err != nil {
		return fmt.Errorf("could not build server: %v", err.Error())
	}

	if err := server.Run(cmd.Context()); err != nil {
		return err
	}

	<-cmd.Context().Done()

	return cmd.Context().Err()
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		log.Fatal(err)
	}
}
