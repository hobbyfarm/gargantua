package userservicecontroller

import (
	"context"
	"time"

	hfInformers "github.com/hobbyfarm/gargantua/v3/pkg/client/informers/externalversions"
	informerV1 "github.com/hobbyfarm/gargantua/v3/pkg/client/listers/hobbyfarm.io/v1"
	"github.com/hobbyfarm/gargantua/v3/pkg/microservices"

	"github.com/golang/glog"
)

const (
	WORKQUEUE_NAME = "workqueue-token"
)

type TokenController struct {
	microservices.DelayingWorkqueueController
	prtLister informerV1.PasswordResetTokenLister
}

func NewPasswordResetTokenController(hfInformerFactory hfInformers.SharedInformerFactory, ctx context.Context) (*TokenController, error) {
	tokenController := TokenController{
		DelayingWorkqueueController: *microservices.NewDelayingWorkqueueController(
			ctx,
			hfInformerFactory.Hobbyfarm().V1().PasswordResetTokens().Informer(),
			ReconcileTokenFunc,
			WORKQUEUE_NAME,
			30*time.Minute),
	}
	tokenController.prtLister = hfInformerFactory.Hobbyfarm().V1().PasswordResetTokens().Lister()

	return &tokenController, nil
}

func ReconcileTokenFunc(objName string) error {
	glog.V(4).Infof("reconciling token inside specific token controller %s", objName)

	/*
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
	*/

	return nil
}
