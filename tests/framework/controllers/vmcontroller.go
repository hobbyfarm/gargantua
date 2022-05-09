package controllers

import (
	"context"
	"fmt"
	"github.com/golang/glog"
	hfv1 "github.com/hobbyfarm/gargantua/pkg/apis/hobbyfarm.io/v1"
	hfClientset "github.com/hobbyfarm/gargantua/pkg/client/clientset/versioned"
	hfInformers "github.com/hobbyfarm/gargantua/pkg/client/informers/externalversions"
	hfListers "github.com/hobbyfarm/gargantua/pkg/client/listers/hobbyfarm.io/v1"
	"github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
	"time"
)

// Stub VM Controller to patch VM objects and mark them ready for testing
type VMController struct {
	hfClientSet hfClientset.Interface
	vmWorkqueue workqueue.RateLimitingInterface
	vmSynced    cache.InformerSynced
	vmLister    hfListers.VirtualMachineLister
	ctx         context.Context
}

func NewVMController(hfClientSet hfClientset.Interface, hfInformerFactory hfInformers.SharedInformerFactory, ctx context.Context) (*VMController, error) {
	v := VMController{}
	v.hfClientSet = hfClientSet
	v.ctx = ctx
	v.vmWorkqueue = workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "vms")
	v.vmSynced = hfInformerFactory.Hobbyfarm().V1().VirtualMachines().Informer().HasSynced
	v.vmLister = hfInformerFactory.Hobbyfarm().V1().VirtualMachines().Lister()

	hfInformerFactory.Hobbyfarm().V1().VirtualMachines().Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: v.enqueueVM,
		UpdateFunc: func(old, new interface{}) {
			v.enqueueVM(new)
		},
		DeleteFunc: v.enqueueVM,
	})

	return &v, nil
}

func (v *VMController) enqueueVM(obj interface{}) {
	var key string
	var err error
	if key, err = cache.MetaNamespaceKeyFunc(obj); err != nil {
		glog.Error(err)
		return
	}
	v.vmWorkqueue.AddRateLimited(key)
}

func (v *VMController) Run(stopCh <-chan struct{}) error {
	defer v.vmWorkqueue.ShutDown()

	glog.V(4).Infof("Starting VM Provisioner controller")
	glog.Info("Waiting for informer caches to sync")

	if ok := cache.WaitForCacheSync(stopCh, v.vmSynced); !ok {
		return fmt.Errorf("failed to wait for caches to sync")
	}

	glog.Info("Starting VM controller worker")
	go wait.Until(v.runVMWorker, time.Second, stopCh)
	glog.Info("Started VM controller worker")
	<-stopCh
	glog.Info("Shutting down VM controller worker")
	return nil
}

func (v *VMController) runVMWorker() {
	for v.processNextVM() {
	}
}

func (v *VMController) processNextVM() bool {
	obj, shutdown := v.vmWorkqueue.Get()
	if shutdown {
		return false
	}
	err := func(obj interface{}) error {
		defer v.vmWorkqueue.Done(obj)
		var key string
		var ok bool
		if key, ok = obj.(string); !ok {
			v.vmWorkqueue.Forget(obj)
			glog.Errorf("expected string in workqueue but got %#v", obj)
			return nil
		}

		namespace, name, err := cache.SplitMetaNamespaceKey(key)
		if err != nil {
			glog.Errorf("invalid resource key: %s", key)
			return nil
		}

		vm, err := v.vmLister.VirtualMachines(namespace).Get(name)
		if err != nil {
			if errors.IsNotFound(err) {
				glog.Errorf("VM %s in workqueue doesnt exist", key)
			}

			return err
		}
		glog.V(9).Infof("process VirtualMachine %s in namespace %s", vm.Name, vm.Namespace)
		err = v.handleProvision(vm)
		if err != nil {
			v.vmWorkqueue.AddRateLimited(key)
			return err
		}

		v.vmWorkqueue.Forget(key)
		return nil
	}(obj)

	if err != nil {
		glog.Errorf("error during processing VM object %v", err)
	}

	return true
}

func (v *VMController) handleProvision(vm *hfv1.VirtualMachine) error {
	status := vm.Status.DeepCopy()
	if status.Status == hfv1.VmStatusRunning {
		return nil // already been processed so lets just ignore it
	}
	logrus.Infof("processing virtualmachine %s", vm.Name)
	glog.V(5).Infof("processing virtualmachine %s", vm.Name)
	status.Status = hfv1.VmStatusRunning
	status.PrivateIP = "127.0.0.1"
	status.PublicIP = "127.0.0.1"
	if vm.Labels == nil {
		vm.Labels = make(map[string]string)
	}
	vm.Labels["ready"] = "true"
	vm.Status = *status
	_, err := v.hfClientSet.HobbyfarmV1().VirtualMachines(vm.Namespace).Update(context.TODO(), vm, metav1.UpdateOptions{})
	return err
}
