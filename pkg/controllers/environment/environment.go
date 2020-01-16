package environment

import (
	"fmt"
	"github.com/golang/glog"
	hfv1 "github.com/hobbyfarm/gargantua/pkg/apis/hobbyfarm.io/v1"
	hfClientset "github.com/hobbyfarm/gargantua/pkg/client/clientset/versioned"
	hfInformers "github.com/hobbyfarm/gargantua/pkg/client/informers/externalversions"
	hfListers "github.com/hobbyfarm/gargantua/pkg/client/listers/hobbyfarm.io/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/retry"
	"k8s.io/client-go/util/workqueue"
	"k8s.io/klog"
	"time"
)

type EnvironmentController struct {
	hfClientSet  *hfClientset.Clientset
	envWorkqueue workqueue.RateLimitingInterface
	vmWorkqueue  workqueue.RateLimitingInterface
	//hfInformerFactory *hfInformers.SharedInformerFactory

	vmTemplateIndexer cache.Indexer

	vmLister  hfListers.VirtualMachineLister
	envLister hfListers.EnvironmentLister

	vmSynced  cache.InformerSynced
	envSynced cache.InformerSynced
}

const (
	vmEnvironmentIndex = "vm.vmclaim.controllers.hobbyfarm.io/environment-index"
)

func NewEnvironmentController(hfClientSet *hfClientset.Clientset, hfInformerFactory hfInformers.SharedInformerFactory) (*EnvironmentController, error) {
	envController := EnvironmentController{}
	envController.hfClientSet = hfClientSet
	envController.vmSynced = hfInformerFactory.Hobbyfarm().V1().VirtualMachines().Informer().HasSynced
	envController.envSynced = hfInformerFactory.Hobbyfarm().V1().Environments().Informer().HasSynced
	envController.envWorkqueue = workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "Environment")

	envController.vmTemplateIndexer = hfInformerFactory.Hobbyfarm().V1().VirtualMachineTemplates().Informer().GetIndexer()
	envController.vmLister = hfInformerFactory.Hobbyfarm().V1().VirtualMachines().Lister()
	envController.envLister = hfInformerFactory.Hobbyfarm().V1().Environments().Lister()
	envInformer := hfInformerFactory.Hobbyfarm().V1().Environments().Informer()
	vmInformer := hfInformerFactory.Hobbyfarm().V1().VirtualMachines().Informer()

	vmInformer.AddEventHandlerWithResyncPeriod(cache.ResourceEventHandlerFuncs{
		AddFunc: envController.handleVM,
		UpdateFunc: func(old, new interface{}) {
			envController.handleVM(new)
		},
		DeleteFunc: envController.handleVM,
	}, time.Second*30)

	envInformer.AddEventHandlerWithResyncPeriod(cache.ResourceEventHandlerFuncs{
		AddFunc: envController.enqueueEnv,
		UpdateFunc: func(old, new interface{}) {
			envController.enqueueEnv(new)
		},
		DeleteFunc: envController.enqueueEnv,
	}, time.Second*30)

	return &envController, nil
}

func (e *EnvironmentController) enqueueEnv(obj interface{}) {
	var key string
	var err error
	if key, err = cache.MetaNamespaceKeyFunc(obj); err != nil {
		//utilruntime.HandleError(err)
		return
	}
	glog.V(8).Infof("Enqueueing env %s", key)
	e.envWorkqueue.AddRateLimited(key)
}

func (e *EnvironmentController) Run(stopCh <-chan struct{}) error {
	defer e.envWorkqueue.ShutDown()

	glog.V(4).Infof("Starting environment controller")
	glog.Info("Waiting for informer caches to sync")
	if ok := cache.WaitForCacheSync(stopCh, e.vmSynced, e.envSynced); !ok {
		return fmt.Errorf("failed to wait for vm and environment caches to sync")
	}
	glog.Info("Starting environment controller workers")
	go wait.Until(e.runEnvWorker, time.Second, stopCh)
	glog.Info("Started environment controller workers")
	//if ok := cache.WaitForCacheSync(stopCh, )
	<-stopCh
	return nil
}

func (e *EnvironmentController) runEnvWorker() {
	glog.V(6).Infof("Starting environment worker")
	for e.processNextEnvironment() {

	}
}

func (e *EnvironmentController) handleVM(obj interface{}) {
	var object metav1.Object
	var ok bool
	if object, ok = obj.(metav1.Object); !ok {
		tombstone, ok := obj.(cache.DeletedFinalStateUnknown)
		if !ok {
			glog.Errorf("error decoding object, invalid type")
			return
		}
		object, ok = tombstone.Obj.(metav1.Object)
		if !ok {
			glog.Errorf("error decoding object tombstone, invalid type")
			return
		}
		klog.V(4).Infof("Recovered deleted object '%s' from tombstone", object.GetName())
	}
	klog.V(4).Infof("Processing object: %s", object.GetName())
	if ownerRef := metav1.GetControllerOf(object); ownerRef != nil {
		// If this object is not owned by a Foo, we should not do anything more
		// with it.
		if ownerRef.Kind != "Environment" {
			return
		}

		env, err := e.envLister.Get(ownerRef.Name)
		if err != nil {
			klog.V(4).Infof("ignoring orphaned object '%s' of foo '%s'", object.GetSelfLink(), ownerRef.Name)
			return
		}

		e.enqueueEnv(env)
		return
	}
}

func (e *EnvironmentController) processNextEnvironment() bool {
	obj, shutdown := e.envWorkqueue.Get()

	if shutdown {
		return false
	}

	err := func() error {
		defer e.envWorkqueue.Done(obj)
		glog.V(4).Infof("processing env in env controller: %v", obj)
		_, objName, err := cache.SplitMetaNamespaceKey(obj.(string)) // this is actually not necessary because VM's are not namespaced yet...
		if err != nil {
			glog.Errorf("error while splitting meta namespace key %v", err)
			//e.vmWorkqueue.AddRateLimited(obj)
			return nil
		}

		err = e.reconcileEnvironment(objName)

		if err != nil {
			glog.Error(err)
		}
		e.envWorkqueue.Forget(obj)
		glog.V(4).Infof("env processed by endpoint controller %v", objName)

		return nil

	}()

	if err != nil {
		return true
	}

	return true
}

func (e *EnvironmentController) reconcileEnvironment(environmentId string) error {
	glog.V(4).Infof("reconciling environment %s", environmentId)
	retryErr := retry.RetryOnConflict(retry.DefaultRetry, func() error {
		result, getErr := e.hfClientSet.HobbyfarmV1().Environments().Get(environmentId, metav1.GetOptions{})
		if getErr != nil {
			return fmt.Errorf("error retrieving latest version of Environment %s: %v", environmentId, getErr)
		}

		vms, err := e.vmLister.List(labels.NewSelector())

		if err != nil {
			return fmt.Errorf("error while retrieving vms for environment id %s %v", environmentId, err)
		}

		allocatedCPU := 0
		allocatedMemory := 0
		allocatedStorage := 0

		available := make(map[string]int)
		for _, vm := range vms {
			if vm.Status.EnvironmentId != environmentId {
				continue
			}
			vmTemplateObj, exists, err := e.vmTemplateIndexer.GetByKey(vm.Spec.VirtualMachineTemplateId)

			if err != nil {
				glog.Errorf("error while getting vm template from indexer %v", err)
			}

			if exists {
				vmTemplate := vmTemplateObj.(*hfv1.VirtualMachineTemplate)
				allocatedCPU = allocatedCPU + vmTemplate.Spec.Resources.CPU
				allocatedMemory = allocatedMemory + vmTemplate.Spec.Resources.Memory
				allocatedStorage = allocatedStorage + vmTemplate.Spec.Resources.Storage
				if vm.Status.Status == hfv1.VmStatusRunning {
					if !vm.Status.Allocated {
						if val, ok := available[vm.Spec.VirtualMachineTemplateId]; ok {
							available[vm.Spec.VirtualMachineTemplateId] = val + 1
						} else {
							available[vm.Spec.VirtualMachineTemplateId] = 1
						}
					}
				}
			}

		}

		result.Status.Used.CPU = allocatedCPU
		result.Status.Used.Memory = allocatedMemory
		result.Status.Used.Storage = allocatedStorage

		result.Status.AvailableCount = available

		_, updateErr := e.hfClientSet.HobbyfarmV1().Environments().Update(result)
		glog.V(4).Infof("updated result for environment")

		return updateErr
	})
	if retryErr != nil {
		return fmt.Errorf("Error updating Environment: %s, %v", environmentId, retryErr)
	}
	return nil
}
