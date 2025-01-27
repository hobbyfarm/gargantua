package registry

import (
	"context"
	"github.com/hobbyfarm/gargantua/v4/pkg/apis/hobbyfarm.io/v4alpha1"
	"github.com/hobbyfarm/mink/pkg/stores"
	"github.com/hobbyfarm/mink/pkg/strategy"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/validation/field"
	"k8s.io/apiserver/pkg/registry/rest"
)

type scenarioValidator struct {
}

func NewScenarioStorage(scenarioStrategy strategy.CompleteStrategy) (rest.Storage, error) {
	var sv = scenarioValidator{}
	return stores.NewBuilder(scenarioStrategy.Scheme(), &v4alpha1.Scenario{}).
		WithCompleteCRUD(scenarioStrategy).
		WithValidateCreate(sv).
		WithValidateUpdate(sv).Build(), nil
}

func (sv scenarioValidator) Validate(ctx context.Context, obj runtime.Object) field.ErrorList {
	return sv.doValidate(ctx, obj)
}

func (sv scenarioValidator) ValidateUpdate(ctx context.Context, obj runtime.Object, _ runtime.Object) field.ErrorList {
	return sv.doValidate(ctx, obj)
}

func (sv scenarioValidator) doValidate(ctx context.Context, obj runtime.Object) (result field.ErrorList) {
	scenario := obj.(*v4alpha1.Scenario)

	errs := ValidatePauseKeepaliveDurations(scenario.Spec.PauseDuration, scenario.Spec.KeepaliveDuration)

	if len(errs) > 0 {
		result = append(result, errs...)
	}

	return
}
