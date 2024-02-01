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

func NewDelayingWorkqueueController(ctx context.Context, informer cache.SharedIndexInformer, name string, resyncPeriod time.Duration) *DelayingWorkqueueController {
	dwqc := &DelayingWorkqueueController{
		*newBaseController(name, ctx, informer, resyncPeriod),
	}

	dwqc.SetWorkqueue(workqueue.NewDelayingQueueWithConfig(workqueue.DelayingQueueConfig{Name: name}))

	return dwqc
}

func (dwq *DelayingWorkqueueController) GetDelayingWorkqueue() (workqueue.DelayingInterface, error) {
	delayingQueue, ok := dwq.GetWorkqueue().(workqueue.DelayingInterface)
	if !ok {
		return nil, fmt.Errorf("Workqueue is not a DelayingQueue")
	}
	return delayingQueue, nil
}
