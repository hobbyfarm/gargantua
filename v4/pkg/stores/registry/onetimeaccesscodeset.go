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

type otacSetValidator struct{}

func NewOneTimeAccessCodeSetStatusStorage(scheme *runtime.Scheme, storage strategy.StatusUpdater) (rest.Storage, error) {
	return stores.NewStatus(scheme, storage), nil
}

func NewOneTimeAccessCodeSetStorage(strat strategy.CompleteStrategy) (rest.Storage, error) {
	var osv = &otacSetValidator{}

	return stores.NewBuilder(strat.Scheme(), &v4alpha1.OneTimeAccessCodeSet{}).
		WithValidateCreate(osv).
		WithValidateUpdate(osv).
		WithCompleteCRUD(strat).Build(), nil
}

func (osv otacSetValidator) Validate(ctx context.Context, obj runtime.Object) (result field.ErrorList) {
	return osv.doValidate(ctx, obj)
}

func (osv otacSetValidator) ValidateUpdate(ctx context.Context, oldObj, newObj runtime.Object) (result field.ErrorList) {
	return osv.doValidate(ctx, newObj)
}

func (osv otacSetValidator) doValidate(ctx context.Context, obj runtime.Object) (result field.ErrorList) {
	otacSet := obj.(*v4alpha1.OneTimeAccessCodeSet)

	return ValidateNotBeforeNotAfter(otacSet.Spec.Template.NotBefore, otacSet.Spec.Template.NotAfter)
}
