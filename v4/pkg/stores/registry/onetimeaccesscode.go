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

type otacValidator struct{}

func NewOneTimeAccessCodeStorage(otacStrategy strategy.CompleteStrategy) (rest.Storage, error) {
	var ov = otacValidator{}

	return stores.NewBuilder(otacStrategy.Scheme(), &v4alpha1.OneTimeAccessCode{}).
		WithValidateCreate(ov).
		WithValidateUpdate(ov).
		WithCompleteCRUD(otacStrategy).Build(), nil
}

func (otacValidator) Validate(ctx context.Context, obj runtime.Object) (result field.ErrorList) {
	otac := obj.(*v4alpha1.OneTimeAccessCode)

	return ValidateNotBeforeNotAfter(otac.Spec.NotBefore, otac.Spec.NotAfter)
}

func (otacValidator) ValidateUpdate(ctx context.Context, obj runtime.Object, _ runtime.Object) (result field.ErrorList) {
	otac := obj.(*v4alpha1.OneTimeAccessCode)

	return ValidateNotBeforeNotAfter(otac.Spec.NotBefore, otac.Spec.NotAfter)
}
