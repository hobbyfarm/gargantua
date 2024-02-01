package microservices

import (
	"context"
	"fmt"
	"time"

	"github.com/golang/glog"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
)

// Reconciler required Reconcile method.
type Reconciler interface {
	Reconcile(objName string) error
}

type BaseController struct {
	name                        string
	workqueue                   workqueue.Interface
	Started                     bool
	Context                     context.Context
	Informer                    cache.SharedIndexInformer              // The informer to attach to
	InformerHandlerRegistration cache.ResourceEventHandlerRegistration // We save the Registration here to unregister when shutting down
	ResyncPeriod                time.Duration
	reconciler                  Reconciler
	threads                     int
}

// Should not be instiantiated on its own, use specific implementation of delayingWorkqueueController or RateLimitingWorkqueueController
func newBaseController(name string, ctx context.Context, informer cache.SharedIndexInformer, resyncPeriod time.Duration) *BaseController {
	baseController := BaseController{
		name:         name,
		Context:      ctx,
		Informer:     informer,
		ResyncPeriod: resyncPeriod,
		threads:      1,
	}
	return &baseController
}

func (c *BaseController) AddEventHandlerWithResyncPeriod() error {
	glog.V(4).Infof("Add EventHandlerWithResyncPeriod")
	reg, err := c.Informer.AddEventHandlerWithResyncPeriod(cache.ResourceEventHandlerFuncs{
		AddFunc: c.enqueue,
		UpdateFunc: func(old, new interface{}) {
			c.enqueue(new)
		},
		DeleteFunc: c.enqueue,
	}, c.ResyncPeriod)

	if err != nil {
		glog.Errorf("Event Handler could not be started. Aborting controller start")
		return err
	}

	c.InformerHandlerRegistration = reg
	return nil
}

// Override this method if you need to wait for other informers aswell
func (c *BaseController) WaitForCacheSync(stopCh <-chan struct{}) error {
	// Wait for the caches to be synced before starting workers
	glog.V(4).Info("Waiting for informer caches to sync")
	if ok := cache.WaitForCacheSync(stopCh, c.Informer.HasSynced); !ok {
		return fmt.Errorf("failed to wait for caches to sync")
	}
	return nil
}

func (c *BaseController) Run(stopCh <-chan struct{}) error {
	if c.workqueue == nil {
		return fmt.Errorf("Workqueue not instantiated.")
	}
	if c.reconciler == nil {
		return fmt.Errorf("Reconciler not instantiated. Call SetReconciler(r Reconciler) to set it")
	}

	defer c.workqueue.ShutDown()
	defer c.stopWorker()

	glog.Infof("Starting controller: %s", c.name)
	c.Started = true

	err := c.AddEventHandlerWithResyncPeriod()
	if err != nil {
		return err
	}
	defer c.Informer.RemoveEventHandler(c.InformerHandlerRegistration)

	err = c.WaitForCacheSync(stopCh)
	if err != nil {
		return err
	}

	glog.Infof("Starting %d worker threads for %s", c.threads, c.name)
	for i := 0; i < c.threads; i++ {
		go wait.Until(c.runWorker, time.Second, stopCh)
	}

	<-stopCh
	glog.Info("Stopping base controller")
	return nil
}

func (c *BaseController) enqueue(obj interface{}) {
	if !c.Started || c.workqueue == nil || c.workqueue.ShuttingDown() {
		glog.V(4).Infof("Object is not being added to the workqueue while controller is not started")
		return
	}

	var key string
	var err error
	if key, err = cache.MetaNamespaceKeyFunc(obj); err != nil {
		glog.V(4).Infof("Error enquing %s: %v", key, err)
		return
	}
	glog.V(4).Infof("Enqueueing: %s", key)
	c.workqueue.Add(key)
}

func (c *BaseController) runWorker() {
	glog.V(4).Infof("Starting worker thread for %s", c.name)
	for c.processNextWorkItem() {
	}
}

func (c *BaseController) processNextWorkItem() bool {
	obj, shutdown := c.workqueue.Get()

	if shutdown {
		return false
	}

	err := func() error {
		defer c.workqueue.Done(obj)
		glog.V(8).Infof("processing next token in queue: %v", obj)
		_, objName, err := cache.SplitMetaNamespaceKey(obj.(string))
		if err != nil {
			glog.Errorf("error while splitting meta namespace key %v", err)
			return nil
		}

		err = c.reconciler.Reconcile(objName)

		if err != nil {
			glog.Error(err)
		}

		glog.V(8).Infof("Successfully processed: %s", objName)
		return nil
	}()

	if err != nil {
		return true
	}

	return true
}

func (c *BaseController) stopWorker() {
	c.Started = false
}

func (c *BaseController) SetReconciler(r Reconciler) {
	c.reconciler = r
}

func (c *BaseController) SetWorkqueue(w workqueue.Interface) {
	c.workqueue = w
}

func (c *BaseController) GetWorkqueue() workqueue.Interface {
	return c.workqueue
}

func (c *BaseController) SetWorkerThreadCount(threads int) {
	c.threads = threads
}
