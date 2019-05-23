package main

import (
	"flag"
	"github.com/golang/glog"
	"github.com/gorilla/mux"
	"github.com/hobbyfarm/gargantua/pkg/accesscode"
	"github.com/hobbyfarm/gargantua/pkg/authclient"
	"github.com/hobbyfarm/gargantua/pkg/authserver"
	hfClientset "github.com/hobbyfarm/gargantua/pkg/client/clientset/versioned"
	hfInformers "github.com/hobbyfarm/gargantua/pkg/client/informers/externalversions"
	"github.com/hobbyfarm/gargantua/pkg/scenarioclient"
	"github.com/hobbyfarm/gargantua/pkg/scenarioserver"
	"github.com/hobbyfarm/gargantua/pkg/scenariosessionserver"
	"github.com/hobbyfarm/gargantua/pkg/shell"
	"github.com/hobbyfarm/gargantua/pkg/signals"
	"github.com/hobbyfarm/gargantua/pkg/vmclaimserver"
	"github.com/hobbyfarm/gargantua/pkg/vmclient"
	"github.com/hobbyfarm/gargantua/pkg/vmserver"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/clientcmd"
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

	authServer, err := authserver.NewAuthServer(hfClient, hfInformerFactory)
	if err != nil {
		glog.Fatal(err)
	}

	authClient, err := authclient.NewAuthClient(hfClient, hfInformerFactory)
	if err != nil {
		glog.Fatal(err)
	}

	acClient, err := accesscode.NewAccessCodeClient(hfClient, hfInformerFactory)
	if err != nil {
		glog.Fatal(err)
	}

	scenarioServer, err := scenarioserver.NewScenarioServer(authClient, acClient, hfClient, hfInformerFactory)

	scenarioClient, err := scenarioclient.NewScenarioClient(scenarioServer)

	ssServer, err := scenariosessionserver.NewScenarioSessionServer(authClient, scenarioClient, hfClient, hfInformerFactory)

	//ssClient, err := scenariosessionclient.NewScenarioSessionClient(ssServer)

	vmServer, err := vmserver.NewVMServer(authClient, hfClient, hfInformerFactory)

	vmClient, err := vmclient.NewVirtualMachineClient(vmServer)

	vmClaimServer, err := vmclaimserver.NewVMClaimServer(authClient, hfClient, hfInformerFactory)

	//vmClaimClient, err := vmclaimclient.NewVMClaimClient(vmClient)

	shellProxy, err := shell.NewShellProxy(authClient, vmClient)

	ssServer.SetupRoutes(r)
	authServer.SetupRoutes(r)
	scenarioServer.SetupRoutes(r)
	vmServer.SetupRoutes(r)
	shellProxy.SetupRoutes(r)
	vmClaimServer.SetupRoutes(r)

	hfInformerFactory.Start(stopCh)

	if ok := cache.WaitForCacheSync(stopCh,
		hfInformerFactory.Hobbyfarm().V1().Users().Informer().HasSynced,
		hfInformerFactory.Hobbyfarm().V1().VirtualMachines().Informer().HasSynced,
		hfInformerFactory.Hobbyfarm().V1().ScenarioSessions().Informer().HasSynced,
		hfInformerFactory.Hobbyfarm().V1().Scenarios().Informer().HasSynced,
		hfInformerFactory.Hobbyfarm().V1().VirtualMachineClaims().Informer().HasSynced,
		hfInformerFactory.Hobbyfarm().V1().AccessCodes().Informer().HasSynced,
		//hfInformerFactory.Hobbyfarm().V1().VirtualMachineTemplates().Informer().HasSynced,,
		); !ok {
		glog.Fatalf("failed to wait for caches to sync")
	}
		glog.Info("listening on 80")
	http.ListenAndServe(":80", r)
}

