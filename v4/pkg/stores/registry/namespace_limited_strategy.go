package registry

import (
	"context"
	"github.com/hobbyfarm/mink/pkg/strategy"
	"github.com/hobbyfarm/mink/pkg/types"
	"k8s.io/apimachinery/pkg/runtime"
	watch "k8s.io/apimachinery/pkg/watch"
	"k8s.io/apiserver/pkg/storage"
)

type namespaceLimitedStrategy struct {
	client    strategy.CompleteStrategy
	namespace string
}

func (n namespaceLimitedStrategy) Create(ctx context.Context, object types.Object) (types.Object, error) {
	object.SetNamespace(n.namespace)

	return n.client.Create(ctx, object)
}

func (n namespaceLimitedStrategy) New() types.Object {
	return n.client.New()
}

func (n namespaceLimitedStrategy) Get(ctx context.Context, namespace, name string) (types.Object, error) {
	namespace = n.namespace

	return n.client.Get(ctx, namespace, name)
}

func (n namespaceLimitedStrategy) Update(ctx context.Context, obj types.Object) (types.Object, error) {
	obj.SetNamespace(n.namespace)

	return n.client.Update(ctx, obj)
}

func (n namespaceLimitedStrategy) UpdateStatus(ctx context.Context, obj types.Object) (types.Object, error) {
	obj.SetNamespace(n.namespace)

	return n.client.UpdateStatus(ctx, obj)
}

func (n namespaceLimitedStrategy) List(ctx context.Context, namespace string, opts storage.ListOptions) (types.ObjectList, error) {
	namespace = n.namespace

	return n.client.List(ctx, namespace, opts)
}

func (n namespaceLimitedStrategy) NewList() types.ObjectList {
	return n.client.NewList()
}

func (n namespaceLimitedStrategy) Delete(ctx context.Context, obj types.Object) (types.Object, error) {
	obj.SetNamespace(n.namespace)

	return n.client.Delete(ctx, obj)
}

func (n namespaceLimitedStrategy) Watch(ctx context.Context, namespace string, opts storage.ListOptions) (<-chan watch.Event, error) {
	namespace = n.namespace

	return n.client.Watch(ctx, namespace, opts)
}

func (n namespaceLimitedStrategy) Destroy() {
	n.client.Destroy()
}

func (n namespaceLimitedStrategy) Scheme() *runtime.Scheme {
	return n.client.Scheme()
}

func NewNamespaceLimitedStrategy(client strategy.CompleteStrategy, namespace string) strategy.CompleteStrategy {
	return namespaceLimitedStrategy{client: client, namespace: namespace}
}
