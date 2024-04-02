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

type machineValidator struct {
	providerGetter        strategy.Getter
	machineTemplateGetter strategy.Getter
	environmentGetter     strategy.Getter
}

func NewMachineStorage(machineStrategy strategy.CompleteStrategy, providerGetter strategy.Getter,
	machineTemplateGetter strategy.Getter, environmentGetter strategy.Getter) (rest.Storage, error) {
	mv := &machineValidator{
		providerGetter:        providerGetter,
		machineTemplateGetter: machineTemplateGetter,
		environmentGetter:     environmentGetter,
	}

	return stores.NewBuilder(machineStrategy.Scheme(), &v4alpha1.Machine{}).
		WithCompleteCRUD(machineStrategy).
		WithValidateCreate(mv).
		WithValidateUpdate(mv).
		WithValidateDelete(mv).Build(), nil
}

func (mv *machineValidator) Validate(ctx context.Context, obj runtime.Object) (result field.ErrorList) {
	return mv.doBasicValidation(ctx, obj)
}

func (mv *machineValidator) ValidateUpdate(ctx context.Context, obj, _ runtime.Object) (result field.ErrorList) {
	return mv.doBasicValidation(ctx, obj)
}

func (mv *machineValidator) ValidateDelete(_ context.Context, _ runtime.Object) *errors.StatusError {
	return nil
}

func (mv *machineValidator) doBasicValidation(ctx context.Context, obj runtime.Object) (result field.ErrorList) {
	machine := obj.(*v4alpha1.Machine)

	if len(machine.OwnerReferences) > 0 {
		return // don't double-validate if the call is coming from inside the house
	}

	provObj, err := mv.providerGetter.Get(ctx, machine.Namespace, machine.Spec.Provider)
	if err != nil {
		result = append(result, field.Invalid(field.NewPath("spec", "provider"),
			machine.Spec.Provider, err.Error()))
		return
	}

	prov := provObj.(*v4alpha1.Provider)
	errs := ValidateProviderConfigurationMap(machine.Spec.ProviderConfiguration, prov)
	if len(errs) > 0 {
		result = append(result, errs...)
		return
	}

	// machinetemplate
	if _, err = mv.machineTemplateGetter.Get(ctx, machine.Namespace, machine.Spec.MachineTemplate); err != nil {
		result = append(result, field.Invalid(field.NewPath("spec", "machineTemplate"),
			machine.Spec.MachineTemplate, err.Error()))
		return
	}

	//environment
	if _, err = mv.environmentGetter.Get(ctx, machine.Namespace, machine.Spec.Environment); err != nil {
		result = append(result, field.Invalid(field.NewPath("spec", "environment"),
			machine.Spec.Environment, err.Error()))
		return
	}

	return
}
