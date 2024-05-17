package main

import (
	"context"
	"flag"
	"os"

	"github.com/hobbyfarm/gargantua/v3/pkg/accesscode"
	"github.com/hobbyfarm/gargantua/v3/pkg/authserver"
	hfClientset "github.com/hobbyfarm/gargantua/v3/pkg/client/clientset/versioned"
	hfInformers "github.com/hobbyfarm/gargantua/v3/pkg/client/informers/externalversions"
	"github.com/hobbyfarm/gargantua/v3/pkg/controllers/tfpcontroller"
	"github.com/hobbyfarm/gargantua/v3/pkg/courseclient"
	"github.com/hobbyfarm/gargantua/v3/pkg/courseserver"
	"github.com/hobbyfarm/gargantua/v3/pkg/crd"
	"github.com/hobbyfarm/gargantua/v3/pkg/environmentserver"
	"github.com/hobbyfarm/gargantua/v3/pkg/microservices"
	predefinedserviceserver "github.com/hobbyfarm/gargantua/v3/pkg/predefinedserviceserver"
	"github.com/hobbyfarm/gargantua/v3/pkg/progressserver"
	"github.com/hobbyfarm/gargantua/v3/pkg/scenarioclient"
	"github.com/hobbyfarm/gargantua/v3/pkg/scenarioserver"
	"github.com/hobbyfarm/gargantua/v3/pkg/scheduledeventserver"
	"github.com/hobbyfarm/gargantua/v3/pkg/sessionserver"
	"github.com/hobbyfarm/gargantua/v3/pkg/shell"
	"github.com/hobbyfarm/gargantua/v3/pkg/signals"
	"github.com/hobbyfarm/gargantua/v3/pkg/util"
	"github.com/hobbyfarm/gargantua/v3/pkg/vmclaimserver"
	"github.com/hobbyfarm/gargantua/v3/pkg/vmclient"
	"github.com/hobbyfarm/gargantua/v3/pkg/vmserver"
	"github.com/hobbyfarm/gargantua/v3/pkg/vmsetserver"
	"github.com/hobbyfarm/gargantua/v3/pkg/vmtemplateserver"
	"github.com/hobbyfarm/gargantua/v3/protos/authn"
	"github.com/hobbyfarm/gargantua/v3/protos/authr"
	"github.com/hobbyfarm/gargantua/v3/protos/setting"

	"github.com/ebauman/crder"
	"golang.org/x/sync/errgroup"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/tools/leaderelection"

	"net/http"
	"sync"
	"time"

	"github.com/golang/glog"
	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
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
	tlsCA              string
)

func init() {
	flag.StringVar(&localKubeconfig, "kubeconfig", "", "Path to kubeconfig of local cluster. Only required if out-of-cluster.")
	flag.StringVar(&localMasterUrl, "master", "", "The address of the Kubernetes API server. Overrides any value in kubeconfig. Only required if out-of-cluster.")
	flag.BoolVar(&disableControllers, "disablecontrollers", false, "Disable the controllers")
	flag.BoolVar(&shellServer, "shellserver", false, "Be a shell server")
	flag.StringVar(&tlsCA, "tls-ca", "/etc/ssl/certs/ca.crt", "Path to CA cert for auth servers")
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

	namespace := util.GetReleaseNamespace()

	if !shellServer {
		crds := crd.GenerateCRDs()

		glog.Info("installing/updating CRDs")
		err = crder.InstallUpdateCRDs(cfg, crds...)
		if err != nil {
			glog.Fatalf("failed installing/updating crds: %s", err.Error())
		}
		glog.Info("finished installing/updating CRDs")
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

	hfInformerFactory := hfInformers.NewSharedInformerFactoryWithOptions(hfClient, time.Second*30, hfInformers.WithNamespace(namespace))
	kubeInformerFactory := informers.NewSharedInformerFactoryWithOptions(kubeClient, time.Second*30, informers.WithNamespace(namespace))

	cert, err := microservices.BuildTLSClientCredentials(tlsCA)
	if err != nil {
		glog.Fatalf("error building cert: %v", err)
	}

	services := []microservices.MicroService{
		microservices.Setting,
		microservices.AuthN,
		microservices.AuthR,
	}
	connections := microservices.EstablishConnections(services, cert)
	for _, conn := range connections {
		defer conn.Close()
	}

	settingClient := setting.NewSettingSvcClient(connections[microservices.Setting])
	authnClient := authn.NewAuthNClient(connections[microservices.AuthN])
	authrClient := authr.NewAuthRClient(connections[microservices.AuthR])

	acClient, err := accesscode.NewAccessCodeClient(hfClient, ctx)
	if err != nil {
		glog.Fatal(err)
	}

	authServer, err := authserver.NewAuthServer(authnClient, hfClient, ctx, acClient)
	if err != nil {
		glog.Fatal(err)
	}

	courseServer, err := courseserver.NewCourseServer(authnClient, authrClient, acClient, hfClient, hfInformerFactory, ctx)
	if err != nil {
		glog.Fatal(err)
	}

	courseClient, err := courseclient.NewCourseClient(courseServer)
	if err != nil {
		glog.Fatal(err)
	}

	scenarioServer, err := scenarioserver.NewScenarioServer(authnClient, authrClient, acClient, hfClient, hfInformerFactory, ctx, courseClient)
	if err != nil {
		glog.Fatal(err)
	}

	scenarioClient, err := scenarioclient.NewScenarioClient(scenarioServer)
	if err != nil {
		glog.Fatal(err)
	}

	sessionServer, err := sessionserver.NewSessionServer(authnClient, authrClient, acClient, scenarioClient, courseClient, hfClient, hfInformerFactory, ctx)
	if err != nil {
		glog.Fatal(err)
	}

	vmServer, err := vmserver.NewVMServer(authnClient, authrClient, hfClient, hfInformerFactory, ctx)
	if err != nil {
		glog.Fatal(err)
	}

	vmSetServer, err := vmsetserver.NewVMSetServer(authnClient, authrClient, hfClient, hfInformerFactory, ctx)
	if err != nil {
		glog.Fatal(err)
	}

	vmClient, err := vmclient.NewVirtualMachineClient(vmServer)
	if err != nil {
		glog.Fatal(err)
	}

	vmClaimServer, err := vmclaimserver.NewVMClaimServer(authnClient, authrClient, hfClient, hfInformerFactory)
	if err != nil {
		glog.Fatal(err)
	}

	shellProxy, err := shell.NewShellProxy(authnClient, authrClient, vmClient, hfClient, kubeClient, ctx)
	if err != nil {
		glog.Fatal(err)
	}

	environmentServer, err := environmentserver.NewEnvironmentServer(authnClient, authrClient, hfClient, ctx)
	if err != nil {
		glog.Fatal(err)
	}

	scheduledEventServer, err := scheduledeventserver.NewScheduledEventServer(authnClient, authrClient, hfClient, ctx)
	if err != nil {
		glog.Fatal(err)
	}

	vmTemplateServer, err := vmtemplateserver.NewVirtualMachineTemplateServer(authnClient, authrClient, hfClient, ctx)
	if err != nil {
		glog.Fatal(err)
	}

	predefinedServiceServer, err := predefinedserviceserver.NewPredefinedServiceServer(authnClient, authrClient, hfClient, ctx)
	if err != nil {
		glog.Fatal(err)
	}

	progressServer, err := progressserver.NewProgressServer(authnClient, authrClient, hfClient, ctx)
	if err != nil {
		glog.Fatal(err)
	}

	if shellServer {
		glog.V(2).Infof("Starting as a shell server")
		shellProxy.SetupRoutes(r)
	} else {
		authServer.SetupRoutes(r)
		sessionServer.SetupRoutes(r)
		courseServer.SetupRoutes(r)
		scenarioServer.SetupRoutes(r)
		vmServer.SetupRoutes(r)
		vmSetServer.SetupRoutes(r)
		//shellProxy.SetupRoutes(r)
		vmClaimServer.SetupRoutes(r)
		environmentServer.SetupRoutes(r)
		scheduledEventServer.SetupRoutes(r)
		vmTemplateServer.SetupRoutes(r)
		progressServer.SetupRoutes(r)
		predefinedServiceServer.SetupRoutes(r)
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

	var wg sync.WaitGroup

	wg.Add(1)

	http.Handle("/", r)

	apiPort := os.Getenv("PORT")
	if apiPort == "" {
		apiPort = "80"
	}
	glog.Info("apiserver listening on " + apiPort)

	go func() {
		defer wg.Done()
		glog.Fatal(http.ListenAndServe(":"+apiPort, handlers.CORS(corsHeaders, corsOrigins, corsMethods)(r)))
	}()

	if !disableControllers {
		lock, err := util.GetLock("controller-manager-gargantua", cfg)
		if err != nil {
			glog.Fatal(err)
		}
		leaderelection.RunOrDie(ctx, leaderelection.LeaderElectionConfig{
			Lock:            lock,
			ReleaseOnCancel: true,
			LeaseDuration:   10 * time.Second,
			RenewDeadline:   5 * time.Second,
			RetryPeriod:     2 * time.Second,
			Callbacks: leaderelection.LeaderCallbacks{
				OnStartedLeading: func(c context.Context) {
					err = bootStrapControllers(kubeClient, hfClient, hfInformerFactory, kubeInformerFactory, acClient, settingClient, ctx, stopCh)
					if err != nil {
						glog.Fatal(err)
					}
				},
				OnStoppedLeading: func() {
					// Need to start informer factory since even when not leader to ensure api layer
					// keeps working.
					hfInformerFactory.Start(stopCh)
					glog.Info("waiting to be elected leader")
				},
				OnNewLeader: func(current_id string) {
					hfInformerFactory.Start(stopCh)
					if current_id == lock.Identity() {
						glog.Info("currently the leader")
						return
					}
					glog.Infof("current leader is %s", current_id)
				},
			},
		})
	} else {
		// default fire up hfInformer as this is still needed by the shell server
		hfInformerFactory.Start(stopCh)
		kubeInformerFactory.Start(stopCh)
	}
	wg.Wait()
}

func bootStrapControllers(kubeClient *kubernetes.Clientset, hfClient *hfClientset.Clientset,
	hfInformerFactory hfInformers.SharedInformerFactory, kubeInformerFactory informers.SharedInformerFactory, acClient *accesscode.AccessCodeClient,
	settingClient setting.SettingSvcClient, ctx context.Context, stopCh <-chan struct{}) error {

	g, gctx := errgroup.WithContext(ctx)
	glog.V(2).Infof("Starting controllers")
	tfpController, err := tfpcontroller.NewTerraformProvisionerController(kubeClient, hfClient, hfInformerFactory, gctx)
	if err != nil {
		return err
	}

	g.Go(func() error {
		return tfpController.Run(stopCh)
	})

	hfInformerFactory.Start(stopCh)
	kubeInformerFactory.Start(stopCh)

	if err = g.Wait(); err != nil {
		glog.Errorf("error starting up the controllers: %v", err)
		return err
	}

	return nil
}
