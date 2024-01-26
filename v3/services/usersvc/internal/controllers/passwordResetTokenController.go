package userservicecontroller

import (
	"context"
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
)

type PasswordResetTokenController struct {
	internalUserServer *userservice.GrpcUserServer

	workqueue workqueue.DelayingInterface

	prtLister   v1.PasswordResetTokenLister
	prtInformer cache.SharedIndexInformer
	prtSynced   cache.InformerSynced

	started bool

	ctx context.Context
}

func NewPasswordResetTokenController(internalUserServer *userservice.GrpcUserServer, hfInformerFactory hfInformers.SharedInformerFactory, ctx context.Context) (*PasswordResetTokenController, error) {
	prtController := PasswordResetTokenController{}
	prtController.internalUserServer = internalUserServer
	prtController.ctx = ctx
	prtController.prtSynced = hfInformerFactory.Hobbyfarm().V1().PasswordResetTokens().Informer().HasSynced

	prtController.workqueue = workqueue.NewDelayingQueueWithConfig(workqueue.DelayingQueueConfig{
		Name: "prt-queue",
	})
	prtController.prtLister = hfInformerFactory.Hobbyfarm().V1().PasswordResetTokens().Lister()

	prtInformer := hfInformerFactory.Hobbyfarm().V1().PasswordResetTokens().Informer()

	prtInformer.AddEventHandlerWithResyncPeriod(cache.ResourceEventHandlerFuncs{
		AddFunc: prtController.enqueue,
		UpdateFunc: func(old, new interface{}) {
			prtController.enqueue(new)
		},
	}, time.Minute*30)

	return &prtController, nil
}

func (s *PasswordResetTokenController) enqueue(obj interface{}) {
	if s.workqueue.ShuttingDown() || !s.started {
		return
	}

	var key string
	var err error
	if key, err = cache.MetaNamespaceKeyFunc(obj); err != nil {
		//utilruntime.HandleError(err)
		return
	}
	glog.V(8).Infof("Enqueueing PRT %s", key)
	//s.ssWorkqueue.AddRateLimited(key)
	s.workqueue.Add(key)
}

func (s *PasswordResetTokenController) Run(stopCh <-chan struct{}) error {
	defer s.workqueue.ShutDown()
	s.started = true

	glog.V(4).Infof("Starting Token controller")
	go wait.Until(s.runWorker, time.Second, stopCh)
	glog.Info("Started Token controller workers")

	<-stopCh
	s.started = false
	glog.V(4).Infof("Stopping Token controller due to stop signal")
	return nil
}

func (s *PasswordResetTokenController) runWorker() {
	glog.V(6).Infof("Starting Token worker")
	for s.processNextSession() {

	}
}

func (s *PasswordResetTokenController) processNextSession() bool {
	obj, shutdown := s.workqueue.Get()

	if shutdown {
		return false
	}

	err := func() error {
		defer s.workqueue.Done(obj)
		glog.V(8).Infof("processing token in token controller: %v", obj)
		_, objName, err := cache.SplitMetaNamespaceKey(obj.(string))
		if err != nil {
			glog.Errorf("error while splitting meta namespace key %v", err)
			return nil
		}

		err = s.reconcile(objName)

		if err != nil {
			glog.Error(err)
		}

		glog.V(8).Infof("PRT processed by PasswordResetToken controller: %v", objName)
		return nil

	}()

	if err != nil {
		return true
	}

	return true
}

func (s *PasswordResetTokenController) reconcile(token string) error {
	glog.V(4).Infof("reconciling PRT %s", token)

	passwordResetToken, err := s.prtLister.PasswordResetTokens(util2.GetReleaseNamespace()).Get(token)

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
		// requeue the session at the correct expiration time
		glog.V(4).Infof("Requeueing PRT %s", passwordResetToken.Name)
		s.workqueue.AddAfter(passwordResetToken, timeUntilExpires)
	}

	return nil
}
