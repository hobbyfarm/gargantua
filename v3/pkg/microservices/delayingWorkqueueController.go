package microservices

import (
	"context"
	"fmt"
	"time"

	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
)

type DelayingWorkqueueController struct {
	BaseController
}

func NewDelayingWorkqueueController(ctx context.Context, informer cache.SharedIndexInformer, reconcileFunc func(objName string) error, queueName string, resyncPeriod time.Duration) *DelayingWorkqueueController {
	dwqc := DelayingWorkqueueController{
		*NewBaseController(ctx, informer, reconcileFunc, resyncPeriod),
	}

	dwqc.Workqueue = workqueue.NewNamedDelayingQueue(queueName)

	return &dwqc
}

func (dwq DelayingWorkqueueController) GetDelayingWorkqueue() (workqueue.DelayingInterface, error) {
	delayingQueue, ok := dwq.Workqueue.(workqueue.DelayingInterface)
	if !ok {
		return nil, fmt.Errorf("workqueue is not a DelayingQueue")
	}
	return delayingQueue, nil
}
