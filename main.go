package main

import (
	"context"
	"flag"
	"os"

	"github.com/hobbyfarm/gargantua/pkg/scheduledeventserver"
	"github.com/hobbyfarm/gargantua/pkg/vmtemplateserver"

	"net/http"
	"sync"
	"time"

	"github.com/golang/glog"
	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	"github.com/hobbyfarm/gargantua/pkg/accesscode"
	"github.com/hobbyfarm/gargantua/pkg/authclient"
	"github.com/hobbyfarm/gargantua/pkg/authserver"
	hfClientset "github.com/hobbyfarm/gargantua/pkg/client/clientset/versioned"
	hfInformers "github.com/hobbyfarm/gargantua/pkg/client/informers/externalversions"
	"github.com/hobbyfarm/gargantua/pkg/controllers/dynamicbindcontroller"
	"github.com/hobbyfarm/gargantua/pkg/controllers/scheduledevent"
	"github.com/hobbyfarm/gargantua/pkg/controllers/session"
	"github.com/hobbyfarm/gargantua/pkg/controllers/tfpcontroller"
	"github.com/hobbyfarm/gargantua/pkg/controllers/vmclaimcontroller"
	"github.com/hobbyfarm/gargantua/pkg/controllers/vmsetcontroller"
	"github.com/hobbyfarm/gargantua/pkg/courseclient"
	"github.com/hobbyfarm/gargantua/pkg/courseserver"
	"github.com/hobbyfarm/gargantua/pkg/environmentserver"
	"github.com/hobbyfarm/gargantua/pkg/scenarioclient"
	"github.com/hobbyfarm/gargantua/pkg/scenarioserver"
	"github.com/hobbyfarm/gargantua/pkg/sessionserver"
	"github.com/hobbyfarm/gargantua/pkg/shell"
	"github.com/hobbyfarm/gargantua/pkg/signals"
	"github.com/hobbyfarm/gargantua/pkg/userserver"
	"github.com/hobbyfarm/gargantua/pkg/vmclaimserver"
	"github.com/hobbyfarm/gargantua/pkg/vmclient"
	"github.com/hobbyfarm/gargantua/pkg/vmserver"
	"github.com/hobbyfarm/gargantua/pkg/util"
	"github.com/hobbyfarm/gargantua/pkg/progressserver"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"

	"k8s.io/client-go/tools/clientcmd"
)

const (
	ClientGoQPS   = 100
	ClientGoBurst = 100
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
	ctx := context.Background()
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

	cfg.QPS = ClientGoQPS
	cfg.Burst = ClientGoBurst

	hfClient, err := hfClientset.NewForConfig(cfg)
	if err != nil {
		glog.Fatal(err)
	}

	kubeClient, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		glog.Fatalf("Error building kubernetes clientset: %s", err.Error())
	}

	namespace := util.GetReleaseNamespace()
	hfInformerFactory := hfInformers.NewSharedInformerFactoryWithOptions(hfClient, time.Second*30, hfInformers.WithNamespace(namespace))

	authClient, err := authclient.NewAuthClient(hfClient, hfInformerFactory)
	if err != nil {
		glog.Fatal(err)
	}

	authServer, err := authserver.NewAuthServer(authClient, hfClient, ctx)
	if err != nil {
		glog.Fatal(err)
	}

	acClient, err := accesscode.NewAccessCodeClient(hfClient, ctx)
	if err != nil {
		glog.Fatal(err)
	}

	courseServer, err := courseserver.NewCourseServer(authClient, acClient, hfClient, hfInformerFactory, ctx)
	if err != nil {
		glog.Fatal(err)
	}

	courseClient, err := courseclient.NewCourseClient(courseServer)
	if err != nil {
		glog.Fatal(err)
	}

	scenarioServer, err := scenarioserver.NewScenarioServer(authClient, acClient, hfClient, hfInformerFactory, ctx)
	if err != nil {
		glog.Fatal(err)
	}

	scenarioClient, err := scenarioclient.NewScenarioClient(scenarioServer)
	if err != nil {
		glog.Fatal(err)
	}

	sessionServer, err := sessionserver.NewSessionServer(authClient, acClient, scenarioClient, courseClient, hfClient, hfInformerFactory, ctx)
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

	shellProxy, err := shell.NewShellProxy(authClient, vmClient, hfClient, kubeClient, ctx)
	if err != nil {
		glog.Fatal(err)
	}

	environmentServer, err := environmentserver.NewEnvironmentServer(authClient, hfClient, ctx)
	if err != nil {
		glog.Fatal(err)
	}

	scheduledEventServer, err := scheduledeventserver.NewScheduledEventServer(authClient, hfClient, ctx)
	if err != nil {
		glog.Fatal(err)
	}

	userServer, err := userserver.NewUserServer(authClient, hfClient, ctx)
	if err != nil {
		glog.Fatal(err)
	}

	vmTemplateServer, err := vmtemplateserver.NewVirtualMachineTemplateServer(authClient, hfClient, ctx)
	if err != nil {
		glog.Fatal(err)
	}

	progressServer, err := progressserver.NewProgressServer(authClient, hfClient, ctx)
	if err != nil {
		glog.Fatal(err)
	}

	if shellServer {
		glog.V(2).Infof("Starting as a shell server")
		shellProxy.SetupRoutes(r)
	} else {
		sessionServer.SetupRoutes(r)
		authServer.SetupRoutes(r)
		courseServer.SetupRoutes(r)
		scenarioServer.SetupRoutes(r)
		vmServer.SetupRoutes(r)
		//shellProxy.SetupRoutes(r)
		vmClaimServer.SetupRoutes(r)
		environmentServer.SetupRoutes(r)
		scheduledEventServer.SetupRoutes(r)
		userServer.SetupRoutes(r)
		vmTemplateServer.SetupRoutes(r)
		progressServer.SetupRoutes(r)
	}

	corsHeaders := handlers.AllowedHeaders([]string{"Authorization", "Content-Type"})
	corsOrigins := handlers.AllowedOrigins([]string{"*"})
	corsMethods := handlers.AllowedMethods([]string{"GET", "POST", "PUT", "HEAD", "OPTIONS", "DELETE"})
	/*
		glog.V(6).Infof("Waiting for informers to synchronize")
		if ok := cache.WaitForCacheSync(stopCh,
			hfInformerFactory.Hobbyfarm().V1().Users().Informer().HasSynced,
			hfInformerFactory.Hobbyfarm().V1().VirtualMachines().Informer().HasSynced,
			hfInformerFactory.Hobbyfarm().V1().Sessions().Informer().HasSynced,
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
		sessionController, err := session.NewSessionController(hfClient, hfInformerFactory, ctx)
		if err != nil {
			glog.Fatal(err)
		}
		scheduledEventController, err := scheduledevent.NewScheduledEventController(hfClient, hfInformerFactory, ctx)
		if err != nil {
			glog.Fatal(err)
		}
		vmClaimController, err := vmclaimcontroller.NewVMClaimController(hfClient, hfInformerFactory, ctx)
		if err != nil {
			glog.Fatal(err)
		}
		tfpController, err := tfpcontroller.NewTerraformProvisionerController(kubeClient, hfClient, hfInformerFactory, ctx)
		if err != nil {
			glog.Fatal(err)
		}
		vmSetController, err := vmsetcontroller.NewVirtualMachineSetController(hfClient, hfInformerFactory, ctx)
		if err != nil {
			glog.Fatal(err)
		}

		dynamicBindController, err := dynamicbindcontroller.NewDynamicBindController(hfClient, hfInformerFactory, ctx)
		if err != nil {
			glog.Fatal(err)
		}

		wg.Add(6)
		/*
			go func() {
				defer wg.Done()
				environmentController.Run(stopCh)
			}()
		*/
		go func() {
			defer wg.Done()
			sessionController.Run(stopCh)
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

		go func() {
			defer wg.Done()
			dynamicBindController.Run(stopCh)
		}()
	}

	hfInformerFactory.Start(stopCh)

	wg.Add(1)

	port := os.Getenv("PORT")
	if port == "" {
		port = "80"
	}
	glog.Info("listening on " + port)

	go func() {
		defer wg.Done()
		glog.Fatal(http.ListenAndServe(":"+port, handlers.CORS(corsHeaders, corsOrigins, corsMethods)(r)))
	}()

	wg.Wait()

}
