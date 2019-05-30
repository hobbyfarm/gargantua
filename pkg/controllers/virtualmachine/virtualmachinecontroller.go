package virtualmachine

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

type VirtualMachineController struct {
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

func NewVirtualMachineController(hfClientSet *hfClientset.Clientset, hfInformerFactory hfInformers.SharedInformerFactory) (*VirtualMachineController, error) {
	vmController := VirtualMachineController{}
	vmController.hfClientSet = hfClientSet
	vmController.vmSynced = hfInformerFactory.Hobbyfarm().V1().VirtualMachines().Informer().HasSynced
	vmController.vmWorkqueue = workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "VM")
	//vmClaimController.hfInformerFactory = hfInformerFactory

	vmController.vmTemplateIndexer = hfInformerFactory.Hobbyfarm().V1().VirtualMachineTemplates().Informer().GetIndexer()

	vmInformer := hfInformerFactory.Hobbyfarm().V1().VirtualMachines().Informer()

	vmInformer.AddEventHandlerWithResyncPeriod(cache.ResourceEventHandlerFuncs{
		AddFunc: vmController.enqueueVM,
		UpdateFunc: func(old, new interface{}) {
			vmController.enqueueVM(new)
		},
		DeleteFunc: vmController.enqueueVM,
	}, time.Second * 30)

	return &vmController, nil
}


func (v *VirtualMachineController) enqueueVM(obj interface{}) {
	var key string
	var err error
	if key, err = cache.MetaNamespaceKeyFunc(obj); err != nil {
		//utilruntime.HandleError(err)
		return
	}
	glog.V(8).Infof("Enqueueing vm %s", key)
	v.vmWorkqueue.AddRateLimited(key)
}

func (v *VirtualMachineController) Run(stopCh <-chan struct{}) error {
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

func (v *VirtualMachineController) runEnvWorker() {
	for v.processNextVM() {

	}
}

// @TODO: Need to handle delete reconciliation of an environment
func (v *VirtualMachineController) processNextVM() bool {
	obj, shutdown := v.vmWorkqueue.Get()

	glog.V(4).Infof("processing VM")

	if shutdown {
		return false
	}
	err := func() error {
		defer e.vmWorkqueue.Done(obj)
		glog.V(4).Infof("processing vm in env controller: %v", obj)
		_, objName, err := cache.SplitMetaNamespaceKey(obj.(string)) // this is actually not necessary because VM's are not namespaced yet...
		if err != nil {
			glog.Errorf("error while splitting meta namespace key %v", err)
			//e.vmWorkqueue.AddRateLimited(obj)
			return nil
		}

		vm, err := v.hfClientSet.HobbyfarmV1().VirtualMachines().Get(objName, metav1.GetOptions{})
		if err != nil {
			glog.Errorf("error while retrieving virtual machine %s, likely deleted %v", objName, err)
			v.vmWorkqueue.Forget(obj)
			return nil
		}

		//err = v.reconcileEnvironment(vm.Status.EnvironmentId)

		if err != nil {
			glog.Error(err)
		}
		v.vmWorkqueue.Forget(obj)
		glog.V(4).Infof("vm processed by endpoint controller %v", objName)

		return nil

	}()

	if err != nil {
		return true
	}

	return true
}