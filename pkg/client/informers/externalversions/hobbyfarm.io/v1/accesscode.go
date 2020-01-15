// Code generated by informer-gen. DO NOT EDIT.

package v1

import (
	time "time"

	hobbyfarmiov1 "github.com/hobbyfarm/gargantua/pkg/apis/hobbyfarm.io/v1"
	versioned "github.com/hobbyfarm/gargantua/pkg/client/clientset/versioned"
	internalinterfaces "github.com/hobbyfarm/gargantua/pkg/client/informers/externalversions/internalinterfaces"
	v1 "github.com/hobbyfarm/gargantua/pkg/client/listers/hobbyfarm.io/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	runtime "k8s.io/apimachinery/pkg/runtime"
	watch "k8s.io/apimachinery/pkg/watch"
	cache "k8s.io/client-go/tools/cache"
)

// AccessCodeInformer provides access to a shared informer and lister for
// AccessCodes.
type AccessCodeInformer interface {
	Informer() cache.SharedIndexInformer
	Lister() v1.AccessCodeLister
}

type accessCodeInformer struct {
	factory          internalinterfaces.SharedInformerFactory
	tweakListOptions internalinterfaces.TweakListOptionsFunc
}

// NewAccessCodeInformer constructs a new informer for AccessCode type.
// Always prefer using an informer factory to get a shared informer instead of getting an independent
// one. This reduces memory footprint and number of connections to the server.
func NewAccessCodeInformer(client versioned.Interface, resyncPeriod time.Duration, indexers cache.Indexers) cache.SharedIndexInformer {
	return NewFilteredAccessCodeInformer(client, resyncPeriod, indexers, nil)
}

// NewFilteredAccessCodeInformer constructs a new informer for AccessCode type.
// Always prefer using an informer factory to get a shared informer instead of getting an independent
// one. This reduces memory footprint and number of connections to the server.
func NewFilteredAccessCodeInformer(client versioned.Interface, resyncPeriod time.Duration, indexers cache.Indexers, tweakListOptions internalinterfaces.TweakListOptionsFunc) cache.SharedIndexInformer {
	return cache.NewSharedIndexInformer(
		&cache.ListWatch{
			ListFunc: func(options metav1.ListOptions) (runtime.Object, error) {
				if tweakListOptions != nil {
					tweakListOptions(&options)
				}
				return client.HobbyfarmV1().AccessCodes().List(options)
			},
			WatchFunc: func(options metav1.ListOptions) (watch.Interface, error) {
				if tweakListOptions != nil {
					tweakListOptions(&options)
				}
				return client.HobbyfarmV1().AccessCodes().Watch(options)
			},
		},
		&hobbyfarmiov1.AccessCode{},
		resyncPeriod,
		indexers,
	)
}

func (f *accessCodeInformer) defaultInformer(client versioned.Interface, resyncPeriod time.Duration) cache.SharedIndexInformer {
	return NewFilteredAccessCodeInformer(client, resyncPeriod, cache.Indexers{cache.NamespaceIndex: cache.MetaNamespaceIndexFunc}, f.tweakListOptions)
}

func (f *accessCodeInformer) Informer() cache.SharedIndexInformer {
	return f.factory.InformerFor(&hobbyfarmiov1.AccessCode{}, f.defaultInformer)
}

func (f *accessCodeInformer) Lister() v1.AccessCodeLister {
	return v1.NewAccessCodeLister(f.Informer().GetIndexer())
}
