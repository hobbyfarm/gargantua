package main

import (
	"flag"
	"github.com/golang/glog"
	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	"github.com/hobbyfarm/gargantua/pkg/accesscode"
	"github.com/hobbyfarm/gargantua/pkg/authclient"
	"github.com/hobbyfarm/gargantua/pkg/authserver"
	hfClientset "github.com/hobbyfarm/gargantua/pkg/client/clientset/versioned"
	hfInformers "github.com/hobbyfarm/gargantua/pkg/client/informers/externalversions"
	"github.com/hobbyfarm/gargantua/pkg/controllers/scheduledevent"
	"github.com/hobbyfarm/gargantua/pkg/util"

	//"k8s.io/client-go/tools/cache"

	//"github.com/hobbyfarm/gargantua/pkg/controllers/environment"
	"github.com/hobbyfarm/gargantua/pkg/controllers/scenariosession"
	"github.com/hobbyfarm/gargantua/pkg/controllers/tfpcontroller"
	"github.com/hobbyfarm/gargantua/pkg/controllers/vmclaimcontroller"
	"github.com/hobbyfarm/gargantua/pkg/controllers/vmsetcontroller"
	"github.com/hobbyfarm/gargantua/pkg/scenarioclient"
	"github.com/hobbyfarm/gargantua/pkg/scenarioserver"
	"github.com/hobbyfarm/gargantua/pkg/scenariosessionserver"
	"github.com/hobbyfarm/gargantua/pkg/shell"
	"github.com/hobbyfarm/gargantua/pkg/signals"
	"github.com/hobbyfarm/gargantua/pkg/vmclaimserver"
	"github.com/hobbyfarm/gargantua/pkg/vmclient"
	"github.com/hobbyfarm/gargantua/pkg/vmserver"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	//"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/clientcmd"
	"net/http"
	"sync"
	"time"
)

var (
	localMasterUrl     string
	localKubeconfig    string
	disableControllers bool
	shellServer        bool
)

func init() {
	flag.StringVar(&localKubeconfig, "kubeconfig", "", "Path to kubeconfig of local cluster. Only required if out-of-cluster.")
	flag.StringVar(&localMasterUrl, "master", "", "The address of the Kubernetes API server. Overrides any value in kubeconfig. Only required if out-of-cluster.")
	flag.BoolVar(&disableControllers, "disablecontrollers", false, "Disable the controllers")
	flag.BoolVar(&shellServer, "shellserver", false, "Be a shell server")
}

func main() {
	//var signal chan struct{}
	//signal = make(chan struct{})

	stopCh := signals.SetupSignalHandler()

	flag.Parse()
	glog.V(2).Infof("Starting Gargantua")
	r := mux.NewRouter()

	cfg, err := rest.InClusterConfig()

	if err != nil {
		cfg, err = clientcmd.BuildConfigFromFlags(localMasterUrl, localKubeconfig)
		if err != nil {
			glog.Fatalf("Error building kubeconfig: %s", err.Error())
		}
	}

	hfClient, err := hfClientset.NewForConfig(cfg)
	if err != nil {
		glog.Fatal(err)
	}

	kubeClient, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		glog.Fatalf("Error building kubernetes clientset: %s", err.Error())
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
	if err != nil {
		glog.Fatal(err)
	}

	scenarioClient, err := scenarioclient.NewScenarioClient(scenarioServer)
	if err != nil {
		glog.Fatal(err)
	}

	ssServer, err := scenariosessionserver.NewScenarioSessionServer(authClient, scenarioClient, hfClient, hfInformerFactory)
	if err != nil {
		glog.Fatal(err)
	}

	vmServer, err := vmserver.NewVMServer(authClient, hfClient, hfInformerFactory)
	if err != nil {
		glog.Fatal(err)
	}

	vmClient, err := vmclient.NewVirtualMachineClient(vmServer)
	if err != nil {
		glog.Fatal(err)
	}

	vmClaimServer, err := vmclaimserver.NewVMClaimServer(authClient, hfClient, hfInformerFactory)
	if err != nil {
		glog.Fatal(err)
	}

	shellProxy, err := shell.NewShellProxy(authClient, vmClient, hfClient, kubeClient)
	if err != nil {
		glog.Fatal(err)
	}

	if shellServer {
		glog.V(2).Infof("Starting as a shell server")
		shellProxy.SetupRoutes(r)
	} else {
		ssServer.SetupRoutes(r)
		authServer.SetupRoutes(r)
		scenarioServer.SetupRoutes(r)
		vmServer.SetupRoutes(r)
		//shellProxy.SetupRoutes(r)
		vmClaimServer.SetupRoutes(r)
	}

	corsHeaders := handlers.AllowedHeaders([]string{"Authorization", "Content-Type"})
	corsOrigins := handlers.AllowedOrigins([]string{"*"})
	corsMethods := handlers.AllowedMethods([]string{"GET", "POST", "PUT", "HEAD", "OPTIONS", "DELETE"})
	/*
	glog.V(6).Infof("Waiting for informers to synchronize")
	if ok := cache.WaitForCacheSync(stopCh,
		hfInformerFactory.Hobbyfarm().V1().Users().Informer().HasSynced,
		hfInformerFactory.Hobbyfarm().V1().VirtualMachines().Informer().HasSynced,
		hfInformerFactory.Hobbyfarm().V1().ScenarioSessions().Informer().HasSynced,
		hfInformerFactory.Hobbyfarm().V1().Scenarios().Informer().HasSynced,
		hfInformerFactory.Hobbyfarm().V1().VirtualMachineClaims().Informer().HasSynced,
		hfInformerFactory.Hobbyfarm().V1().AccessCodes().Informer().HasSynced,
		hfInformerFactory.Hobbyfarm().V1().VirtualMachineTemplates().Informer().HasSynced,
		//hfInformerFactory.Hobbyfarm().V1().Environments().Informer().HasSynced,
		hfInformerFactory.Hobbyfarm().V1().VirtualMachineSets().Informer().HasSynced,
	); !ok {
		glog.Fatalf("failed to wait for caches to sync")
	}
	glog.V(6).Infof("Informers have synchronized")
	*/

	http.Handle("/", r)

	var wg sync.WaitGroup
	if !disableControllers {
		/*
		environmentController, err := environment.NewEnvironmentController(hfClient, hfInformerFactory)
		if err != nil {
			glog.Fatal(err)
		}
		*/
		glog.V(2).Infof("Starting controllers")
		scenarioSessionController, err := scenariosession.NewScenarioSessionController(hfClient, hfInformerFactory)
		if err != nil {
			glog.Fatal(err)
		}
		scheduledEventController, err := scheduledevent.NewScheduledEventController(hfClient, hfInformerFactory)
		if err != nil {
			glog.Fatal(err)
		}
		vmClaimController, err := vmclaimcontroller.NewVMClaimController(hfClient, hfInformerFactory)
		if err != nil {
			glog.Fatal(err)
		}
		tfpController, err := tfpcontroller.NewTerraformProvisionerController(kubeClient, hfClient, hfInformerFactory)
		if err != nil {
			glog.Fatal(err)
		}
		vmSetController, err := vmsetcontroller.NewVirtualMachineSetController(hfClient, hfInformerFactory)
		if err != nil {
			glog.Fatal(err)
		}

		wg.Add(5)
		/*
		go func() {
			defer wg.Done()
			environmentController.Run(stopCh)
		}()
		*/
		go func() {
			defer wg.Done()
			scenarioSessionController.Run(stopCh)
		}()

		go func() {
			defer wg.Done()
			scheduledEventController.Run(stopCh)
		}()

		go func() {
			defer wg.Done()
			vmClaimController.Run(stopCh)
		}()

		go func() {
			defer wg.Done()
			tfpController.Run(stopCh)
		}()

		go func() {
			defer wg.Done()
			vmSetController.Run(stopCh)
		}()
	}

	hfInformerFactory.Start(stopCh)
	glog.Info("listening on 80")

	wg.Add(1)

	go func() {
		defer wg.Done()
		glog.Fatal(http.ListenAndServe(":80", handlers.CORS(corsHeaders, corsOrigins, corsMethods)(r)))
	}()

	wg.Wait()

}
