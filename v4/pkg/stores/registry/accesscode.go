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

type accessCodeValidator struct {
}

func NewAccessCodeStorage(accessCodeStrategy strategy.CompleteStrategy) (rest.Storage, error) {
	acv := &accessCodeValidator{}
	
	return stores.NewBuilder(accessCodeStrategy.Scheme(), &v4alpha1.AccessCode{}).
		WithCompleteCRUD(accessCodeStrategy).
		WithValidateCreate(acv).
		WithValidateUpdate(acv).Build(), nil
}

func (acv *accessCodeValidator) Validate(ctx context.Context, obj runtime.Object) field.ErrorList {
	return acv.basicValidation(ctx, obj)
}

func (acv *accessCodeValidator) ValidateUpdate(ctx context.Context, obj, old runtime.Object) field.ErrorList {
	return acv.basicValidation(ctx, obj)
}

func (acv *accessCodeValidator) basicValidation(ctx context.Context, obj runtime.Object) field.ErrorList {
	accessCode := obj.(*v4alpha1.AccessCode)

	return ValidateNotBeforeNotAfter(accessCode.Spec.NotBefore, accessCode.Spec.NotAfter)
}
