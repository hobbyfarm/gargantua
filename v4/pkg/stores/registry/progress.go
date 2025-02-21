package registry

import (
	"github.com/hobbyfarm/gargantua/v4/pkg/apis/hobbyfarm.io/v4alpha1"
	"github.com/hobbyfarm/mink/pkg/stores"
	"github.com/hobbyfarm/mink/pkg/strategy"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apiserver/pkg/registry/rest"
)

func NewProgressStatusStorage(scheme *runtime.Scheme, storage strategy.StatusUpdater) rest.Storage {
	return stores.NewStatus(scheme, storage)
}

func NewProgressStorage(progressStrategy strategy.CompleteStrategy) (rest.Storage, error) {
	return stores.NewBuilder(progressStrategy.Scheme(), &v4alpha1.Progress{}).
		WithCompleteCRUD(progressStrategy).Build(), nil
}
