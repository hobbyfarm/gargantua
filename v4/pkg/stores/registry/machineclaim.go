package registry

import (
	"context"
	"github.com/hobbyfarm/gargantua/v4/pkg/apis/hobbyfarm.io/v4alpha1"
	"github.com/hobbyfarm/mink/pkg/stores"
	"github.com/hobbyfarm/mink/pkg/strategy"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/validation/field"
	"k8s.io/apiserver/pkg/registry/rest"
)

type machineClaimValidator struct {
	machineTemplateGetter strategy.Getter
	userGetter            strategy.Getter
	accessCodeLister      strategy.Getter
	scheduledEventGetter  strategy.Getter
}

func NewMachineClaimStorage(machineClaimStrategy strategy.CompleteStrategy) (rest.Storage, error) {
	mcv := &machineClaimValidator{}

	return stores.NewBuilder(machineClaimStrategy.Scheme(), &v4alpha1.MachineClaim{}).
		WithCompleteCRUD(machineClaimStrategy).
		WithValidateCreate(mcv).
		WithValidateUpdate(mcv).
		WithValidateDelete(mcv).Build(), nil
}

func (mcv *machineClaimValidator) Validate(ctx context.Context, obj runtime.Object) field.ErrorList {

}

func (mcv *machineClaimValidator) ValidateUpdate(ctx context.Context, obj runtime.Object, _ runtime.Object) field.ErrorList {

}

func (mcv *machineClaimValidator) ValidateDelete(ctx context.Context, obj runtime.Object) *errors.StatusError {

}

func (mcv *machineClaimValidator) doBasicValidation(ctx context.Context, obj runtime.Object) (result field.ErrorList) {
	machineClaim := obj.(*v4alpha1.MachineClaim)

	if machineClaim.Spec.BindStrategy == v4alpha1.BindStrategyRequireMachineSets && machineClaim.Spec.PreferRequireMachineSets == nil {
		result = append(result, field.Required(field.NewPath("spec").Child("machineSet"),
			"machineSet is required when bindStrategy is RequireMachineSets"))
	}

	if machineClaim.Spec.BindStrategy == v4alpha1.BindStrategyPreferMachineSets && machineClaim.Spec.PreferRequireMachineSets == nil {
		result = append(result, field.Required(field.NewPath("spec").Child("machineSet"),
			"machineSet is required when bindStrategy is PreferMachineSets"))
	}

	return
}
