package accesscode

import (
	"context"
	"github.com/hobbyfarm/gargantua/v4/pkg/apis/hobbyfarm.io/v4alpha1"
	"github.com/hobbyfarm/mink/pkg/stores"
	"github.com/hobbyfarm/mink/pkg/strategy"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/validation/field"
	"k8s.io/apiserver/pkg/registry/rest"
)

type v4alpha1Validator struct {
}

func Newv4alpha1Storage(accessCodeStrategy strategy.CompleteStrategy) (rest.Storage, error) {
	acv := &v4alpha1Validator{}

	return stores.NewBuilder(accessCodeStrategy.Scheme(), &v4alpha1.AccessCode{}).
		WithCompleteCRUD(accessCodeStrategy).
		WithValidateCreate(acv).
		WithValidateUpdate(acv).Build(), nil
}

func (acv *v4alpha1Validator) Validate(ctx context.Context, obj runtime.Object) field.ErrorList {
	return acv.basicValidation(ctx, obj)
}

func (acv *v4alpha1Validator) ValidateUpdate(ctx context.Context, obj, old runtime.Object) field.ErrorList {
	return acv.basicValidation(ctx, obj)
}

func (acv *v4alpha1Validator) basicValidation(ctx context.Context, obj runtime.Object) field.ErrorList {
	accessCode := obj.(*v4alpha1.AccessCode)

	if accessCode.Spec.Code == "" {
		return field.ErrorList{field.Required(field.NewPath("spec").Child("code"), "code is required")}
	}

	if accessCode.Spec.NotAfter.Before(accessCode.Spec.NotBefore) {
		return field.ErrorList{field.Invalid(field.NewPath("spec").Child("notAfter"),
			accessCode.Spec.NotAfter, "notAfter must be after notBefore")}
	}

	if accessCode.Spec.NotBefore.After(accessCode.Spec.NotAfter.Time) {
		return field.ErrorList{field.Invalid(field.NewPath("spec").Child("notBefore"),
			accessCode.Spec.NotBefore, "notBefore must be before notAfter")}
	}

	return nil

}
