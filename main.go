package main

import (
	"flag"
	"github.com/golang/glog"
	"github.com/gorilla/mux"
	"github.com/hobbyfarm/gargantua/pkg/auth"
	hfClientset "github.com/hobbyfarm/gargantua/pkg/client/clientset/versioned"
	hfInformers "github.com/hobbyfarm/gargantua/pkg/client/informers/externalversions"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/clientcmd"
	"github.com/hobbyfarm/gargantua/pkg/signals"
	"net/http"
	"time"
)

var (
	localMasterUrl  string
	localKubeconfig string
)

func init() {
	flag.StringVar(&localKubeconfig, "kubeconfig", "", "Path to kubeconfig of local cluster. Only required if out-of-cluster.")
	flag.StringVar(&localMasterUrl, "master", "", "The address of the Kubernetes API server. Overrides any value in kubeconfig. Only required if out-of-cluster.")
}

func main() {
	//var signal chan struct{}
	//signal = make(chan struct{})

	stopCh := signals.SetupSignalHandler()

	flag.Parse()
	glog.V(2).Infof("Starting")
	r := mux.NewRouter()

	cfg, err := clientcmd.BuildConfigFromFlags(localMasterUrl, localKubeconfig)
	if err != nil {
		glog.Fatalf("Error building kubeconfig: %s", err.Error())
	}

	hfClient, err := hfClientset.NewForConfig(cfg)
	if err != nil {
		glog.Fatal(err)
	}

	hfInformerFactory := hfInformers.NewSharedInformerFactory(hfClient, time.Second*30)

	a, err := auth.NewAuth(hfInformerFactory)
	if err != nil {
		glog.Fatal(err)
	}

	a.Setup(r)

	hfInformerFactory.Start(stopCh)

	if ok := cache.WaitForCacheSync(stopCh, hfInformerFactory.Hobbyfarm().V1().Users().Informer().HasSynced); !ok {
		glog.Fatalf("failed to wait for caches to sync")
	}

	http.ListenAndServe(":80", r)
}

