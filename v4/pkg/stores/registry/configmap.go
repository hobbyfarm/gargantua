package registry

import (
	"github.com/hobbyfarm/mink/pkg/stores"
	"github.com/hobbyfarm/mink/pkg/strategy"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apiserver/pkg/registry/rest"
)

func NewConfigMapStorage(configMapStrategy strategy.CompleteStrategy) (rest.Storage, error) {
	return stores.NewBuilder(configMapStrategy.Scheme(), &v1.ConfigMap{}).
		WithCompleteCRUD(configMapStrategy).Build(), nil
}
