package microservices

import (
	"context"
	"fmt"
	"time"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
)

type DelayingWorkqueueController struct {
	ShardedController
}

func NewDelayingWorkqueueController(ctx context.Context, informer cache.SharedIndexInformer, kubeClient *kubernetes.Clientset, name string, resyncPeriod time.Duration, queue workqueue.DelayingInterface) *DelayingWorkqueueController {
	dwqc := &DelayingWorkqueueController{
		*NewShardedController(ctx, informer, kubeClient, name, resyncPeriod),
	}
	if queue != nil {
		dwqc.SetWorkqueue(queue)
	} else {
		dwqc.SetWorkqueue(workqueue.NewDelayingQueueWithConfig(workqueue.DelayingQueueConfig{Name: name}))
	}

	return dwqc
}

func (dwq *DelayingWorkqueueController) GetDelayingWorkqueue() (workqueue.DelayingInterface, error) {
	delayingQueue, ok := dwq.GetWorkqueue().(workqueue.DelayingInterface)
	if !ok {
		return nil, fmt.Errorf("workqueue is not a DelayingQueue")
	}
	return delayingQueue, nil
}
