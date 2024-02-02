package microservices

import (
	"context"
	"fmt"
	"time"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
)

type RateLimitingWorkqueueController struct {
	DistributedController
}

func NewRateLimitingWorkqueueController(ctx context.Context, informer cache.SharedIndexInformer, kubeClient *kubernetes.Clientset, reconcileFunc func(objName string) error, name string, resyncPeriod time.Duration, rateLimiter workqueue.RateLimiter) *RateLimitingWorkqueueController {
	rlwq := &RateLimitingWorkqueueController{
		*NewDistributedController(ctx, informer, kubeClient, name, resyncPeriod),
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
