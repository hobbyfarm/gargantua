package remote

import (
	"context"
	"github.com/hobbyfarm/gargantua/v4/pkg/stores/translators"
	"github.com/hobbyfarm/mink/pkg/strategy"
	"github.com/hobbyfarm/mink/pkg/strategy/remote"
	"github.com/hobbyfarm/mink/pkg/strategy/translation"
	"github.com/hobbyfarm/mink/pkg/types"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/apiserver/pkg/storage"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var _ strategy.CompleteStrategy = (*NamespaceScopedRemote)(nil)

type NamespaceScopedRemote struct {
	root      strategy.CompleteStrategy
	namespace string
}

func NewNamespaceScopedRemote(obj types.Object, client client.WithWatch, namespace string) *NamespaceScopedRemote {
	ogRemote := remote.NewRemote(obj, client)

	scopingTranslator := translators.NewSetNamespaceTranslator(namespace)
	return &NamespaceScopedRemote{
		root: translation.NewSimpleTranslationStrategyWithOptions(scopingTranslator, ogRemote, translation.TranslationOptions{
			AppendPublicUIDSegment: false,
		}),
		namespace: namespace,
	}
}

func (n NamespaceScopedRemote) Create(ctx context.Context, object types.Object) (types.Object, error) {
	object.SetNamespace(n.namespace)

	return n.root.Create(ctx, object)
}

func (n NamespaceScopedRemote) New() types.Object {
	return n.root.New()
}

func (n NamespaceScopedRemote) Get(ctx context.Context, _, name string) (types.Object, error) {
	return n.root.Get(ctx, n.namespace, name)
}

func (n NamespaceScopedRemote) Update(ctx context.Context, obj types.Object) (types.Object, error) {
	obj.SetNamespace(n.namespace)

	return n.root.Update(ctx, obj)
}

func (n NamespaceScopedRemote) UpdateStatus(ctx context.Context, obj types.Object) (types.Object, error) {
	obj.SetNamespace(n.namespace)

	return n.root.UpdateStatus(ctx, obj)
}

func (n NamespaceScopedRemote) List(ctx context.Context, _ string, opts storage.ListOptions) (types.ObjectList, error) {
	return n.root.List(ctx, n.namespace, opts)
}

func (n NamespaceScopedRemote) NewList() types.ObjectList {
	return n.root.NewList()
}

func (n NamespaceScopedRemote) Delete(ctx context.Context, obj types.Object) (types.Object, error) {
	obj.SetNamespace(n.namespace)

	return n.root.Delete(ctx, obj)
}

func (n NamespaceScopedRemote) Watch(ctx context.Context, _ string, opts storage.ListOptions) (<-chan watch.Event, error) {
	return n.root.Watch(ctx, n.namespace, opts)
}

func (n NamespaceScopedRemote) Destroy() {
	n.Destroy()
}

func (n NamespaceScopedRemote) Scheme() *runtime.Scheme {
	return n.root.Scheme()
}
