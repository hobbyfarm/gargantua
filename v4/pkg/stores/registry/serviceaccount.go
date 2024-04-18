package registry

import (
	"github.com/hobbyfarm/gargantua/v3/pkg/util"
	"github.com/hobbyfarm/mink/pkg/stores"
	"github.com/hobbyfarm/mink/pkg/strategy"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apiserver/pkg/registry/rest"
)

func NewServiceAccountStorage(serviceAccountStrategy strategy.CompleteStrategy) (rest.Storage, error) {
	return stores.NewBuilder(serviceAccountStrategy.Scheme(), &v1.ServiceAccount{}).
		WithCompleteCRUD(NewNamespaceLimitedStrategy(serviceAccountStrategy, util.GetReleaseNamespace())).Build(), nil
}
