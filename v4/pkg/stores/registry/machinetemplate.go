package registry

import (
	"github.com/hobbyfarm/gargantua/v4/pkg/apis/hobbyfarm.io/v4alpha1"
	"github.com/hobbyfarm/mink/pkg/stores"
	"github.com/hobbyfarm/mink/pkg/strategy"
	"k8s.io/apiserver/pkg/registry/rest"
)

func NewMachineTemplateStorage(
	machineTemplateStrategy strategy.CompleteStrategy,
) (rest.Storage, error) {
	return stores.NewBuilder(machineTemplateStrategy.Scheme(), &v4alpha1.MachineTemplate{}).
		WithCompleteCRUD(machineTemplateStrategy).Build(), nil
}
