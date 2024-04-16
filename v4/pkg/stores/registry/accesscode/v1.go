package accesscode

import (
	v1 "github.com/hobbyfarm/gargantua/v3/pkg/apis/hobbyfarm.io/v1"
	"github.com/hobbyfarm/mink/pkg/stores"
	"github.com/hobbyfarm/mink/pkg/strategy"
	"k8s.io/apiserver/pkg/registry/rest"
)

func NewV1Storage(v1Storage strategy.CompleteStrategy) (rest.Storage, error) {
	return stores.NewBuilder(v1Storage.Scheme(), &v1.AccessCode{}).
		WithCompleteCRUD(v1Storage).Build(), nil
}
