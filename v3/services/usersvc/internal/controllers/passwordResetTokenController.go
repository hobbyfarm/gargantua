package userservicecontroller

import (
	"context"
	"fmt"
	"time"

	userservice "github.com/hobbyfarm/gargantua/services/usersvc/v3/internal"
	hfInformers "github.com/hobbyfarm/gargantua/v3/pkg/client/informers/externalversions"
	v1 "github.com/hobbyfarm/gargantua/v3/pkg/client/listers/hobbyfarm.io/v1"
	util2 "github.com/hobbyfarm/gargantua/v3/pkg/util"

	"github.com/golang/glog"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
)

const (
	TOKEN_MAX_VALID_DURATION = time.Hour * 1
	WORKQUEUE_NAME           = "workqueue-token"
)

type PasswordResetTokenController struct {
	internalUserServer *userservice.GrpcUserServer

	workqueue workqueue.DelayingInterface

	prtLister              v1.PasswordResetTokenLister
	prtInformer            cache.SharedIndexInformer
	prtHandlerRegistration cache.ResourceEventHandlerRegistration

	started bool

	ctx context.Context
}

func NewPasswordResetTokenController(hfInformerFactory hfInformers.SharedInformerFactory, ctx context.Context) (*PasswordResetTokenController, error) {
	prtController := PasswordResetTokenController{}
	prtController.ctx = ctx
	prtController.prtLister = hfInformerFactory.Hobbyfarm().V1().PasswordResetTokens().Lister()
	prtController.prtInformer = hfInformerFactory.Hobbyfarm().V1().PasswordResetTokens().Informer()

	return &prtController, nil
}

func (c *PasswordResetTokenController) enqueue(obj interface{}) {
	glog.V(4).Infof("Trying to enqueue PRT %v", obj) // TODO remove

	if !c.started || c.workqueue == nil || c.workqueue.ShuttingDown() {
		glog.V(4).Infof("token is not being added to the workqueue while controller is not started")
		return
	}

	var key string
	var err error
	if key, err = cache.MetaNamespaceKeyFunc(obj); err != nil {
		glog.V(4).Infof("Error enquing token %s: %v", key, err)
		return
	}
	glog.V(4).Infof("Enqueueing PRT %s", key)
	c.workqueue.Add(key)
}

func (c *PasswordResetTokenController) Run(stopCh <-chan struct{}) error {
	defer c.stopWorker()

	glog.V(4).Infof("Starting Token controller")

	c.workqueue = workqueue.NewDelayingQueueWithConfig(workqueue.DelayingQueueConfig{
		Name: WORKQUEUE_NAME,
	})
	defer c.workqueue.ShutDown()

	reg, err := c.prtInformer.AddEventHandlerWithResyncPeriod(cache.ResourceEventHandlerFuncs{
		AddFunc: c.enqueue,
		UpdateFunc: func(old, new interface{}) {
			c.enqueue(new)
		},
		DeleteFunc: c.enqueue,
	}, time.Minute*30)

	if err != nil {
		glog.V(4).Infof("Event Handler could not be started. Aborting controller start")
		return err
	}

	c.prtHandlerRegistration = reg
	defer c.prtInformer.RemoveEventHandler(c.prtHandlerRegistration)

	glog.Info("Waiting for informer caches to sync")
	if ok := cache.WaitForCacheSync(stopCh, c.prtInformer.HasSynced); !ok {
		glog.V(4).Infof("Error: failed to wait for informer caches to sync")
		return fmt.Errorf("failed to wait for informer caches to sync")
	}

	go wait.Until(c.runWorker, time.Second, stopCh)
	c.started = true
	glog.Info("Started Token controller worker")
	<-stopCh
	glog.V(4).Infof("Stopping Token controller due to stop signal")
	return nil
}

func (c *PasswordResetTokenController) runWorker() {
	glog.V(6).Infof("Starting Token worker")
	for c.processNextToken() {

	}
}

func (c *PasswordResetTokenController) processNextToken() bool {
	obj, shutdown := c.workqueue.Get()

	if shutdown {
		return false
	}

	err := func() error {
		defer c.workqueue.Done(obj)
		glog.V(8).Infof("processing token in token controller: %v", obj)
		_, objName, err := cache.SplitMetaNamespaceKey(obj.(string))
		if err != nil {
			glog.Errorf("error while splitting meta namespace key %v", err)
			return nil
		}

		err = c.reconcile(objName)

		if err != nil {
			glog.Error(err)
		}

		glog.V(8).Infof("token processed by Token controller: %v", objName)
		return nil

	}()

	if err != nil {
		return true
	}

	return true
}

func (c *PasswordResetTokenController) reconcile(token string) error {
	glog.V(4).Infof("reconciling token %s", token)

	passwordResetToken, err := c.prtLister.PasswordResetTokens(util2.GetReleaseNamespace()).Get(token)

	if err != nil {
		return err
	}

	now := time.Now()

	expires, err := time.Parse(time.UnixDate, passwordResetToken.Spec.Timestamp)
	expires = expires.Add(TOKEN_MAX_VALID_DURATION)

	if err != nil {
		return err
	}

	timeUntilExpires := expires.Sub(now)

	if timeUntilExpires < 0 {
		glog.V(4).Infof("PRT %s seems to old, can be deleted", passwordResetToken.Name)
	} else {
		// requeue the token at the correct expiration time
		glog.V(4).Infof("Requeueing PRT %s", passwordResetToken.Name)
		c.workqueue.AddAfter(passwordResetToken, timeUntilExpires)
	}

	return nil
}

func (c *PasswordResetTokenController) stopWorker() {
	c.started = false
}
