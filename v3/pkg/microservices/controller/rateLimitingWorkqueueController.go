package microservices

import (
	"context"
	"fmt"
	"time"

	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
)

type RateLimitingWorkqueueController struct {
	BaseController
}

func NewRateLimitingWorkqueueController(ctx context.Context, informer cache.SharedIndexInformer, reconcileFunc func(objName string) error, name string, resyncPeriod time.Duration, rateLimiter workqueue.RateLimiter) *RateLimitingWorkqueueController {
	rlwq := &RateLimitingWorkqueueController{
		*newBaseController(name, ctx, informer, resyncPeriod),
	}

	rlwq.SetWorkqueue(workqueue.NewRateLimitingQueueWithConfig(rateLimiter, workqueue.RateLimitingQueueConfig{Name: name}))

	return rlwq
}

func (rlwq *RateLimitingWorkqueueController) GetRateLimitingWorkqueue() (workqueue.RateLimitingInterface, error) {
	rateLimitingQueue, ok := rlwq.GetWorkqueue().(workqueue.RateLimitingInterface)
	if !ok {
		return nil, fmt.Errorf("Workqueue is not a DelayingQueue")
	}
	return rateLimitingQueue, nil
}
