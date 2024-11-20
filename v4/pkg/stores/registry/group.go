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

type groupValidator struct{}

func NewGroupStorage(groupStrategy strategy.CompleteStrategy) (rest.Storage, error) {
	var gv = &groupValidator{}

	return stores.NewBuilder(groupStrategy.Scheme(), &v4alpha1.Group{}).
		WithCompleteCRUD(groupStrategy).
		WithValidateCreate(gv).
		WithValidateUpdate(gv).Build(), nil
}

func (gv groupValidator) ValidateUpdate(ctx context.Context, new runtime.Object, _ runtime.Object) (result field.ErrorList) {
	return gv.doValidate(ctx, new)
}

func (gv groupValidator) Validate(ctx context.Context, obj runtime.Object) field.ErrorList {
	return gv.doValidate(ctx, obj)
}

func (gv groupValidator) doValidate(ctx context.Context, obj runtime.Object) (result field.ErrorList) {
	group := obj.(*v4alpha1.Group)

	if group.Spec.DisplayName == "" {
		result = append(result, field.Invalid(field.NewPath("spec", "displayName"),
			group.Spec.DisplayName, "displayName is required"))
	}

	return
}
