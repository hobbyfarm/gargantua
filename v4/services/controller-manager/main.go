package main

import (
	"fmt"
	"github.com/hobbyfarm/gargantua/v4/pkg/apis/hobbyfarm.io/v4alpha1"
	"github.com/hobbyfarm/gargantua/v4/pkg/controllers/serviceaccount"
	"github.com/rancher/lasso/pkg/controller"
	"github.com/spf13/cobra"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/clientcmd"
	"log"
)

var (
	kubeconfig  string
	kubecontext string
	namespace   string
)

var rootCmd = &cobra.Command{
	Use:   "controller-manager",
	Short: "run controllers for hobbyfarm",
	RunE:  app,
}

func init() {
	rootCmd.Flags().StringVar(&kubeconfig, "kubeconfig", "", "path to kubeconfig file")
	rootCmd.Flags().StringVar(&kubecontext, "kubecontext", "default", "kubecontext")
	rootCmd.Flags().StringVar(&namespace, "namespace", "hobbyfarm", "namespace in which to operate")
}

func app(cmd *cobra.Command, args []string) error {
	cfg, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
	if err != nil {
		return fmt.Errorf("could not connect to kubernetes cluster: %v", err.Error())
	}

	scheme := runtime.NewScheme()
	if err := v4alpha1.AddToScheme(scheme); err != nil {
		return fmt.Errorf("error adding v4alpha to scheme: %s", err.Error())
	}
	if err := corev1.AddToScheme(scheme); err != nil {
		return fmt.Errorf("error adding corev1 to scheme: %s", err.Error())
	}

	factory, err := controller.NewSharedControllerFactoryFromConfig(cfg, scheme)
	if err != nil {
		return fmt.Errorf("error building shared controller factory: %s", err.Error())
	}

	if err := serviceaccount.RegisterHandlers(factory); err != nil {
		return fmt.Errorf("error registering handlers: %s", err.Error())
	}

	if err := factory.Start(cmd.Context(), 1); err != nil {
		return fmt.Errorf("error starting controllers: %s", err.Error())
	}

	<-cmd.Context().Done()

	return cmd.Context().Err()
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		log.Fatal(err)
	}
}
