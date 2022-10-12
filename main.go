package main

import (
	"context"
	"crypto/tls"
	"flag"
	"os"

	"github.com/ebauman/crder"
	"github.com/hobbyfarm/gargantua/pkg/crd"
	"github.com/hobbyfarm/gargantua/pkg/rbac"
	"github.com/hobbyfarm/gargantua/pkg/rbacclient"
	"github.com/hobbyfarm/gargantua/pkg/rbacserver"
	tls2 "github.com/hobbyfarm/gargantua/pkg/tls"
	"github.com/hobbyfarm/gargantua/pkg/webhook/conversion"
	"github.com/hobbyfarm/gargantua/pkg/webhook/conversion/user"
	"golang.org/x/sync/errgroup"
	v1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	apiextensions "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/tools/leaderelection"
	"k8s.io/client-go/tools/leaderelection/resourcelock"

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
	"github.com/hobbyfarm/gargantua/pkg/progressserver"
	"github.com/hobbyfarm/gargantua/pkg/scenarioclient"
	"github.com/hobbyfarm/gargantua/pkg/scenarioserver"
	"github.com/hobbyfarm/gargantua/pkg/sessionserver"
	"github.com/hobbyfarm/gargantua/pkg/shell"
	"github.com/hobbyfarm/gargantua/pkg/signals"
	"github.com/hobbyfarm/gargantua/pkg/userserver"
	"github.com/hobbyfarm/gargantua/pkg/util"
	"github.com/hobbyfarm/gargantua/pkg/vmclaimserver"
	"github.com/hobbyfarm/gargantua/pkg/vmclient"
	"github.com/hobbyfarm/gargantua/pkg/vmserver"
	"github.com/hobbyfarm/gargantua/pkg/vmsetserver"
	wranglerRbac "github.com/rancher/wrangler/pkg/generated/controllers/rbac"
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
	installRBACRoles   bool
	webhookTLSCert     string
	webhookTLSKey      string
	webhookTLSCA       string
)

func init() {
	flag.StringVar(&localKubeconfig, "kubeconfig", "", "Path to kubeconfig of local cluster. Only required if out-of-cluster.")
	flag.StringVar(&localMasterUrl, "master", "", "The address of the Kubernetes API server. Overrides any value in kubeconfig. Only required if out-of-cluster.")
	flag.BoolVar(&disableControllers, "disablecontrollers", false, "Disable the controllers")
	flag.BoolVar(&shellServer, "shellserver", false, "Be a shell server")
	flag.BoolVar(&installRBACRoles, "installrbacroles", false, "Install default RBAC Roles")
	flag.StringVar(&webhookTLSCert, "webhook-tls-cert", "/webhook-secret/tls.crt", "Path to TLS certificate for webhook server")
	flag.StringVar(&webhookTLSKey, "webhook-tls-key", "/webhook-secret/tls.key", "Path to TLS key for webhook server")
	flag.StringVar(&webhookTLSCA, "webhook-tls-ca", "/webhook-secret/ca", "Path to CA cert for webhook server")
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

	var ca string
	if !shellServer {
		ca, err := os.ReadFile(webhookTLSCA)
		if err != nil {
			glog.Fatalf("error reading ca certificate: %s", err.Error())
		}

		crds := crd.GenerateCRDs(string(ca), v1.ServiceReference{
			Namespace: namespace,
			Name:      "hobbyfarm-webhook",
		})

		glog.Info("installing/updating CRDs")
		err = crder.InstallUpdateCRDs(cfg, crds...)
		if err != nil {
			glog.Fatalf("failed installing/updating crds: %s", err.Error())
		}
		glog.Info("finished installing/updating CRDs")
	}

	// self manage default rbac roles
	if installRBACRoles {
		err = rbac.Create(ctx, cfg)
		if err != nil {
			glog.Fatalf("Error installing RBAC roles: %s", err.Error())
		}
		glog.V(9).Infof("Successfully installed RBAC Roles")
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

	apiExtensionsClient, err := apiextensions.NewForConfig(cfg)
	if err != nil {
		glog.Fatalf("error building apiextensions clientset: %s", err.Error())
	}

	hfInformerFactory := hfInformers.NewSharedInformerFactoryWithOptions(hfClient, time.Second*30, hfInformers.WithNamespace(namespace))
	kubeInformerFactory := informers.NewSharedInformerFactoryWithOptions(kubeClient, time.Second*30, informers.WithNamespace(namespace))

	rbacControllerFactory := wranglerRbac.NewFactoryFromConfigOrDie(cfg)

	rbacClient, err := rbacclient.NewRbacClient(namespace, kubeInformerFactory)
	if err != nil {
		glog.Fatal(err)
	}

	authClient, err := authclient.NewAuthClient(hfClient, hfInformerFactory, rbacClient)
	if err != nil {
		glog.Fatal(err)
	}

	rbacServer := rbacserver.NewRbacServer(kubeClient, authClient, rbacClient)

	authServer, err := authserver.NewAuthServer(authClient, hfClient, ctx, rbacClient)
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

	scenarioServer, err := scenarioserver.NewScenarioServer(authClient, acClient, hfClient, hfInformerFactory, ctx, courseClient)
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

	vmServer, err := vmserver.NewVMServer(authClient, hfClient, hfInformerFactory, ctx)
	if err != nil {
		glog.Fatal(err)
	}

	vmSetServer, err := vmsetserver.NewVMSetServer(authClient, hfClient, hfInformerFactory, ctx)
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
		vmSetServer.SetupRoutes(r)
		//shellProxy.SetupRoutes(r)
		vmClaimServer.SetupRoutes(r)
		environmentServer.SetupRoutes(r)
		scheduledEventServer.SetupRoutes(r)
		userServer.SetupRoutes(r)
		vmTemplateServer.SetupRoutes(r)
		progressServer.SetupRoutes(r)
		rbacServer.SetupRoutes(r)
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

	// shell server does not serve webhook endpoint, so don't start it
	if !shellServer {
		user.Init()
		conversionRouter := mux.NewRouter()
		conversion.New(conversionRouter, apiExtensionsClient, string(ca))

		webhookPort := os.Getenv("WEBHOOK_PORT")
		if webhookPort == "" {
			webhookPort = "444"
		}
		glog.Info("webhook listening on " + webhookPort)

		wg.Add(1)
		go func() {
			defer wg.Done()

			cert, err := tls2.ReadKeyPair(webhookTLSCert, webhookTLSKey)
			if err != nil {
				glog.Fatalf("error generating x509keypair from conversion cert and key: %s", err)
			}

			server := http.Server{
				TLSConfig: &tls.Config{
					Certificates: []tls.Certificate{*cert},
				},
				Addr:    ":" + webhookPort,
				Handler: handlers.CORS(corsHeaders, corsOrigins, corsMethods)(conversionRouter),
			}

			glog.Fatal(server.ListenAndServeTLS("", ""))
		}()
	}

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
		lock, err := getLock("controller-manager", cfg)
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
					err = bootStrapControllers(kubeClient, hfClient, hfInformerFactory, kubeInformerFactory, rbacControllerFactory, ctx, stopCh)
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
	hfInformerFactory hfInformers.SharedInformerFactory, kubeInformerFactory informers.SharedInformerFactory, rbacControllerFactory *wranglerRbac.Factory,
	ctx context.Context, stopCh <-chan struct{}) error {

	g, gctx := errgroup.WithContext(ctx)
	glog.V(2).Infof("Starting controllers")
	sessionController, err := session.NewSessionController(hfClient, hfInformerFactory, gctx)
	if err != nil {
		return err
	}
	scheduledEventController, err := scheduledevent.NewScheduledEventController(hfClient, hfInformerFactory, gctx)
	if err != nil {
		return err
	}
	vmClaimController, err := vmclaimcontroller.NewVMClaimController(hfClient, hfInformerFactory, gctx)
	if err != nil {
		return err
	}
	tfpController, err := tfpcontroller.NewTerraformProvisionerController(kubeClient, hfClient, hfInformerFactory, gctx)
	if err != nil {
		return err
	}
	vmSetController, err := vmsetcontroller.NewVirtualMachineSetController(hfClient, hfInformerFactory, gctx)
	if err != nil {
		return err
	}
	dynamicBindController, err := dynamicbindcontroller.NewDynamicBindController(hfClient, hfInformerFactory, gctx)
	if err != nil {
		return err
	}

	g.Go(func() error {
		return sessionController.Run(stopCh)
	})

	g.Go(func() error {
		return scheduledEventController.Run(stopCh)
	})

	g.Go(func() error {
		return vmClaimController.Run(stopCh)
	})

	g.Go(func() error {
		return tfpController.Run(stopCh)
	})

	g.Go(func() error {
		return vmSetController.Run(stopCh)
	})

	g.Go(func() error {
		return dynamicBindController.Run(stopCh)
	})

	g.Go(func() error {
		return rbacControllerFactory.Start(ctx, 1)
	})

	hfInformerFactory.Start(stopCh)
	kubeInformerFactory.Start(stopCh)

	if err = g.Wait(); err != nil {
		glog.Errorf("error starting up the controllers: %v", err)
		return err
	}

	return nil
}

func getLock(lockName string, cfg *rest.Config) (resourcelock.Interface, error) {
	hostname, err := os.Hostname()
	if err != nil {
		return nil, err
	}

	ns := util.GetReleaseNamespace()
	return resourcelock.NewFromKubeconfig(resourcelock.ConfigMapsLeasesResourceLock, ns, lockName, resourcelock.ResourceLockConfig{Identity: hostname}, cfg, 15*time.Second)
}
