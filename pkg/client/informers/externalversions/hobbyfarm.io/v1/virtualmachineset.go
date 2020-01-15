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

// VirtualMachineSetInformer provides access to a shared informer and lister for
// VirtualMachineSets.
type VirtualMachineSetInformer interface {
	Informer() cache.SharedIndexInformer
	Lister() v1.VirtualMachineSetLister
}

type virtualMachineSetInformer struct {
	factory          internalinterfaces.SharedInformerFactory
	tweakListOptions internalinterfaces.TweakListOptionsFunc
}

// NewVirtualMachineSetInformer constructs a new informer for VirtualMachineSet type.
// Always prefer using an informer factory to get a shared informer instead of getting an independent
// one. This reduces memory footprint and number of connections to the server.
func NewVirtualMachineSetInformer(client versioned.Interface, resyncPeriod time.Duration, indexers cache.Indexers) cache.SharedIndexInformer {
	return NewFilteredVirtualMachineSetInformer(client, resyncPeriod, indexers, nil)
}

// NewFilteredVirtualMachineSetInformer constructs a new informer for VirtualMachineSet type.
// Always prefer using an informer factory to get a shared informer instead of getting an independent
// one. This reduces memory footprint and number of connections to the server.
func NewFilteredVirtualMachineSetInformer(client versioned.Interface, resyncPeriod time.Duration, indexers cache.Indexers, tweakListOptions internalinterfaces.TweakListOptionsFunc) cache.SharedIndexInformer {
	return cache.NewSharedIndexInformer(
		&cache.ListWatch{
			ListFunc: func(options metav1.ListOptions) (runtime.Object, error) {
				if tweakListOptions != nil {
					tweakListOptions(&options)
				}
				return client.HobbyfarmV1().VirtualMachineSets().List(options)
			},
			WatchFunc: func(options metav1.ListOptions) (watch.Interface, error) {
				if tweakListOptions != nil {
					tweakListOptions(&options)
				}
				return client.HobbyfarmV1().VirtualMachineSets().Watch(options)
			},
		},
		&hobbyfarmiov1.VirtualMachineSet{},
		resyncPeriod,
		indexers,
	)
}

func (f *virtualMachineSetInformer) defaultInformer(client versioned.Interface, resyncPeriod time.Duration) cache.SharedIndexInformer {
	return NewFilteredVirtualMachineSetInformer(client, resyncPeriod, cache.Indexers{cache.NamespaceIndex: cache.MetaNamespaceIndexFunc}, f.tweakListOptions)
}

func (f *virtualMachineSetInformer) Informer() cache.SharedIndexInformer {
	return f.factory.InformerFor(&hobbyfarmiov1.VirtualMachineSet{}, f.defaultInformer)
}

func (f *virtualMachineSetInformer) Lister() v1.VirtualMachineSetLister {
	return v1.NewVirtualMachineSetLister(f.Informer().GetIndexer())
}
