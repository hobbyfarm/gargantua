package registry

import (
	"github.com/hobbyfarm/gargantua/v4/pkg/apis/hobbyfarm.io/v4alpha1"
	"github.com/hobbyfarm/mink/pkg/stores"
	"github.com/hobbyfarm/mink/pkg/strategy"
	"k8s.io/apiserver/pkg/registry/rest"
)

func NewPredefinedServiceStorage(predefinedServiceStrategy strategy.CompleteStrategy) (rest.Storage, error) {
	return stores.NewBuilder(predefinedServiceStrategy.Scheme(), &v4alpha1.PredefinedService{}).
		WithCompleteCRUD(predefinedServiceStrategy).Build(), nil
}
