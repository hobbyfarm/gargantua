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

type BaseController struct {
	Workqueue                   workqueue.Interface
	Started                     bool
	Context                     context.Context
	Informer                    cache.SharedIndexInformer
	InformerHandlerRegistration cache.ResourceEventHandlerRegistration
	ResyncPeriod                time.Duration
	ReconcileFunc               func(objName string) error
}

// Should not be instiantiated in its own, use specific implementation of delayingWorkqueueController or RateLimitingWorkqueueController
func NewBaseController(ctx context.Context, informer cache.SharedIndexInformer, reconcileFunc func(objName string) error, resyncPeriod time.Duration) *BaseController {
	return &BaseController{
		Workqueue:     workqueue.New(),
		Context:       ctx,
		Informer:      informer,
		ResyncPeriod:  resyncPeriod,
		ReconcileFunc: reconcileFunc,
	}
}

func (c *BaseController) AddEventHandlerWithResyncPeriod() error {
	glog.Info("Add EventHandlerWithResyncPeriod")
	reg, err := c.Informer.AddEventHandlerWithResyncPeriod(cache.ResourceEventHandlerFuncs{
		AddFunc: c.enqueue,
		UpdateFunc: func(old, new interface{}) {
			c.enqueue(new)
		},
		DeleteFunc: c.enqueue,
	}, c.ResyncPeriod)

	if err != nil {
		glog.V(4).Infof("Event Handler could not be started. Aborting controller start")
		return err
	}

	c.InformerHandlerRegistration = reg
	return nil
}

// Override this method if you need to wait for other informers aswell
func (c *BaseController) WaitForCacheSync(stopCh <-chan struct{}) error {
	// Wait for the caches to be synced before starting workers
	glog.Info("Waiting for informer caches to sync")
	if ok := cache.WaitForCacheSync(stopCh, c.Informer.HasSynced); !ok {
		return fmt.Errorf("failed to wait for caches to sync")
	}
	return nil
}

func (c *BaseController) Run(stopCh <-chan struct{}) error {
	defer c.Workqueue.ShutDown()
	defer c.stopWorker()

	glog.Info("Starting controller")
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

	go wait.Until(c.runWorker, time.Second, stopCh)

	<-stopCh
	glog.Info("Stopping base controller")
	return nil
}

func (c *BaseController) enqueue(obj interface{}) {
	if !c.Started || c.Workqueue == nil || c.Workqueue.ShuttingDown() {
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
	c.Workqueue.Add(key)
}

func (c *BaseController) runWorker() {
	for c.processNextWorkItem() {
	}
}

func (c *BaseController) processNextWorkItem() bool {
	obj, shutdown := c.Workqueue.Get()

	if shutdown {
		return false
	}

	err := func() error {
		defer c.Workqueue.Done(obj)
		glog.V(8).Infof("processing next token in queue: %v", obj)
		_, objName, err := cache.SplitMetaNamespaceKey(obj.(string))
		if err != nil {
			glog.Errorf("error while splitting meta namespace key %v", err)
			return nil
		}

		err = c.ReconcileFunc(objName)

		if err != nil {
			glog.Error(err)
		}

		glog.Infof("Successfully processed: %s", objName)
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
