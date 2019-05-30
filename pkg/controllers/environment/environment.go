package environment

import (
	"fmt"
	"github.com/golang/glog"
	hfv1 "github.com/hobbyfarm/gargantua/pkg/apis/hobbyfarm.io/v1"
	hfClientset "github.com/hobbyfarm/gargantua/pkg/client/clientset/versioned"
	hfInformers "github.com/hobbyfarm/gargantua/pkg/client/informers/externalversions"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/retry"
	"k8s.io/client-go/util/workqueue"
	"time"
)

type EnvironmentController struct {
	hfClientSet *hfClientset.Clientset
	vmWorkqueue workqueue.RateLimitingInterface
	//hfInformerFactory *hfInformers.SharedInformerFactory

	vmIndexer cache.Indexer
	vmTemplateIndexer cache.Indexer

	vmSynced cache.InformerSynced

}
const (
	vmEnvironmentIndex = "vm.vmclaim.controllers.hobbyfarm.io/environment-index"
)

func NewEnvironmentController(hfClientSet *hfClientset.Clientset, hfInformerFactory hfInformers.SharedInformerFactory) (*EnvironmentController, error) {
	envController := EnvironmentController{}
	envController.hfClientSet = hfClientSet
	envController.vmSynced = hfInformerFactory.Hobbyfarm().V1().VirtualMachines().Informer().HasSynced
	envController.vmWorkqueue = workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "VM")
	//vmClaimController.hfInformerFactory = hfInformerFactory

	envController.vmTemplateIndexer = hfInformerFactory.Hobbyfarm().V1().VirtualMachineTemplates().Informer().GetIndexer()

	vmInformer := hfInformerFactory.Hobbyfarm().V1().VirtualMachines().Informer()

	vmInformer.AddEventHandlerWithResyncPeriod(cache.ResourceEventHandlerFuncs{
		AddFunc: envController.enqueueVM,
		UpdateFunc: func(old, new interface{}) {
			envController.enqueueVM(new)
		},
		DeleteFunc: envController.enqueueVM,
	}, time.Second * 30)

	return &envController, nil
}


func (e *EnvironmentController) enqueueVM(obj interface{}) {
	var key string
	var err error
	if key, err = cache.MetaNamespaceKeyFunc(obj); err != nil {
		//utilruntime.HandleError(err)
		return
	}
	glog.V(8).Infof("Enqueueing vm %s", key)
	e.vmWorkqueue.AddRateLimited(key)
}

func (e *EnvironmentController) Run(stopCh <-chan struct{}) error {
	defer e.vmWorkqueue.ShutDown()

	glog.V(4).Infof("Starting environment controller")
	glog.Info("Waiting for informer caches to sync")
	if ok := cache.WaitForCacheSync(stopCh, e.vmSynced); !ok {
		return fmt.Errorf("failed to wait for caches to sync")
	}
	glog.Info("Starting environment controller worker")
	go wait.Until(e.runEnvWorker, time.Second, stopCh)
	glog.Info("Started environment controller worker")
	//if ok := cache.WaitForCacheSync(stopCh, )
	<-stopCh
	return nil
}

func (e *EnvironmentController) runEnvWorker() {
	for e.processNextVM() {

	}
}

// @TODO: Need to handle delete reconciliation of an environment
func (e *EnvironmentController) processNextVM() bool {
	obj, shutdown := e.vmWorkqueue.Get()

	glog.V(4).Infof("processing VM")

	if shutdown {
		return false
	}

	glog.V(4).Infof("gmm %v", obj)

	err := func() error {
		defer e.vmWorkqueue.Done(obj)
		glog.V(4).Infof("processing vm in env controller: %v", obj)
		_, objName, err := cache.SplitMetaNamespaceKey(obj.(string)) // this is actually not necessary because VM's are not namespaced yet...
		if err != nil {
			glog.Errorf("error while splitting meta namespace key %v", err)
			//e.vmWorkqueue.AddRateLimited(obj)
			return nil
		}

		vm, err := e.hfClientSet.HobbyfarmV1().VirtualMachines().Get(objName, metav1.GetOptions{})
		if err != nil {
			glog.Errorf("error while retrieving virtual machine %s, likely deleted %v", objName, err)
			e.vmWorkqueue.Forget(obj)
			return nil
		}

		err = e.reconcileEnvironment(vm.Status.EnvironmentId)

		if err != nil {
			glog.Error(err)
		}
		e.vmWorkqueue.Forget(obj)
		glog.V(4).Infof("vm processed by endpoint controller %v", objName)

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
			return fmt.Errorf("Error retrieving latest version of Environment %s: %v", environmentId, getErr)
		}

		vms, err := e.hfClientSet.HobbyfarmV1().VirtualMachines().List(metav1.ListOptions{})

		if err != nil {
			return fmt.Errorf("error while retrieving vms for environment id %s %v", environmentId, err)
		}

		allocatedCPU := 0
		allocatedMemory := 0
		allocatedStorage := 0

		available := make(map[string]int)
		for _, vm := range vms.Items {
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
				if !vm.Status.Allocated {
					if val, ok := available[vm.Spec.VirtualMachineTemplateId]; ok {
						available[vm.Spec.VirtualMachineTemplateId] = val + 1
					} else {
						available[vm.Spec.VirtualMachineTemplateId] = 1
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