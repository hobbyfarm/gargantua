package userservicecontroller

import (
	"context"
	"time"

	hfInformers "github.com/hobbyfarm/gargantua/v3/pkg/client/informers/externalversions"
	informerV1 "github.com/hobbyfarm/gargantua/v3/pkg/client/listers/hobbyfarm.io/v1"
	controllers "github.com/hobbyfarm/gargantua/v3/pkg/microservices/controller"
	"github.com/hobbyfarm/gargantua/v3/pkg/util"

	"github.com/golang/glog"
)

const (
	NAME = "token-controller"
)

type TokenController struct {
	controllers.DelayingWorkqueueController
	controllers.Reconciler
	tokenLister informerV1.PasswordResetTokenLister
}

func NewPasswordResetTokenController(hfInformerFactory hfInformers.SharedInformerFactory, ctx context.Context) (*TokenController, error) {
	tokenController := &TokenController{
		DelayingWorkqueueController: *controllers.NewDelayingWorkqueueController(
			ctx,
			hfInformerFactory.Hobbyfarm().V1().PasswordResetTokens().Informer(),
			NAME,
			30*time.Minute),
	}

	tokenController.tokenLister = hfInformerFactory.Hobbyfarm().V1().PasswordResetTokens().Lister()
	tokenController.SetReconciler(tokenController)

	return tokenController, nil
}

func (dwq *TokenController) Reconcile(objName string) error {
	glog.V(4).Infof("reconciling token inside specific token controller %s", objName)
	token, err := dwq.tokenLister.PasswordResetTokens(util.GetReleaseNamespace()).Get(objName)

	if err != nil {
		return err
	}

	tokenDuration, err := util.GetDurationWithDays(token.Spec.Duration)
	if err != nil {
		return err
	}

	tokenDurationParsed, err := time.ParseDuration(tokenDuration)
	if err != nil {
		return err
	}

	now := time.Now()

	expires, err := time.Parse(time.UnixDate, token.Spec.Timestamp)
	expires = expires.Add(tokenDurationParsed)

	if err != nil {
		return err
	}

	timeUntilExpires := expires.Sub(now)

	if timeUntilExpires < 0 {
		glog.V(4).Infof("Token %s seems to old, can be deleted", token.Name)
	} else {
		// requeue the token at the correct expiration time
		glog.V(4).Infof("Requeueing token %s as the duration is not reached", token.Name)
		delayingWorkqueue, err := dwq.GetDelayingWorkqueue()
		if err != nil {
			return err
		}
		delayingWorkqueue.AddAfter(token, timeUntilExpires)
	}

	return nil
}
