package registry

import (
	"github.com/hobbyfarm/gargantua/v3/pkg/util"
	"github.com/hobbyfarm/mink/pkg/stores"
	"github.com/hobbyfarm/mink/pkg/strategy"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apiserver/pkg/registry/rest"
)

func NewSecretStorage(secretStrategy strategy.CompleteStrategy) (rest.Storage, error) {
	return stores.NewBuilder(secretStrategy.Scheme(), &v1.Secret{}).
		WithCompleteCRUD(NewNamespaceLimitedStrategy(secretStrategy, util.GetReleaseNamespace())).
		Build(), nil
}
