package event

import (
	"context"
	"github.com/hobbyfarm/gargantua/v4/pkg/apis/hobbyfarm.io/v4alpha1"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"time"
)

// TODO: Make this a flag

const (
	EventGarbageCollectionMinutes = 60
)

type EventGarbageCollectorController struct {
	kclient client.Client
}

func New(mgr manager.Manager) error {
	ev := &EventGarbageCollectorController{
		kclient: mgr.GetClient(),
	}

	if err := builder.
		ControllerManagedBy(mgr).
		For(&v4alpha1.Event{}).
		Named("event-gc").
		Complete(ev); err != nil {
		return err
	}

	return nil
}

func (cx EventGarbageCollectorController) Reconcile(ctx context.Context, req reconcile.Request) (reconcile.Result, error) {
	// deletion criteria are any events over 1hr old.

	ev := &v4alpha1.Event{}
	if err := cx.kclient.Get(ctx, req.NamespacedName, ev); err != nil {
		return reconcile.Result{}, client.IgnoreNotFound(err)
	}

	// TODO - Inefficient. Can we do this easier with kclient.DeleteAllOf or some alternative?
	var exp = time.Duration(EventGarbageCollectionMinutes * int64(time.Minute))
	if ev.CreationTimestamp.Add(exp).Before(time.Now()) {
		// exp has passed
		if err := cx.kclient.Delete(ctx, ev); err != nil {
			return reconcile.Result{}, err
		}
	}

	return reconcile.Result{}, nil
}
