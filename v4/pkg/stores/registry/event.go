package registry

import (
	"github.com/hobbyfarm/gargantua/v4/pkg/apis/hobbyfarm.io/v4alpha1"
	"github.com/hobbyfarm/mink/pkg/stores"
	"github.com/hobbyfarm/mink/pkg/strategy"
	"k8s.io/apiserver/pkg/registry/rest"
)

func NewEventStorage(eventStrategy strategy.CompleteStrategy) (rest.Storage, error) {
	return stores.NewBuilder(eventStrategy.Scheme(), &v4alpha1.Event{}).
		WithCompleteCRUD(eventStrategy).Build(), nil
}
