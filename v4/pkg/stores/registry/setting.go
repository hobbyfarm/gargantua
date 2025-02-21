package registry

import (
	"context"
	"fmt"
	"github.com/hobbyfarm/gargantua/v4/pkg/apis/hobbyfarm.io/v4alpha1"
	"github.com/hobbyfarm/mink/pkg/stores"
	"github.com/hobbyfarm/mink/pkg/strategy"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/validation/field"
	"k8s.io/apiserver/pkg/registry/rest"
)

type settingValidator struct{}

func NewSettingStorage(settingStrategy strategy.CompleteStrategy) (rest.Storage, error) {
	sv := settingValidator{}

	return stores.NewBuilder(settingStrategy.Scheme(), &v4alpha1.Setting{}).
		WithValidateCreate(sv).
		WithValidateUpdate(sv).
		WithCompleteCRUD(settingStrategy).Build(), nil
}

func (sv settingValidator) Validate(ctx context.Context, obj runtime.Object) field.ErrorList {
	return sv.doValidate(ctx, obj)
}

func (sv settingValidator) ValidateUpdate(ctx context.Context, obj runtime.Object, _ runtime.Object) field.ErrorList {
	return sv.doValidate(ctx, obj)
}

func (settingValidator) doValidate(ctx context.Context, obj runtime.Object) (result field.ErrorList) {
	set := obj.(*v4alpha1.Setting)

	if err := set.Validate(set.Value); err != nil {
		result = append(result, field.Invalid(field.NewPath("value"), set.Value,
			fmt.Sprintf("setting value failed validation: %v", err.Error())))
	}

	return
}
