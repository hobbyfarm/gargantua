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

type machineSetValidator struct {
	providerGetter        strategy.Getter
	machineTemplateGetter strategy.Getter
	environmentGetter     strategy.Getter
}

func NewMachineSetStorage(machineSetStrategy strategy.CompleteStrategy,
	providerGetter strategy.Getter,
	machineTemplateGetter strategy.Getter,
	environmentGetter strategy.Getter) (rest.Storage, error) {
	msv := &machineSetValidator{
		providerGetter:        providerGetter,
		machineTemplateGetter: machineTemplateGetter,
		environmentGetter:     environmentGetter,
	}
	return stores.NewBuilder(machineSetStrategy.Scheme(), &v4alpha1.MachineSet{}).
		WithCompleteCRUD(machineSetStrategy).
		WithValidateCreate(msv).
		WithValidateUpdate(msv).Build(), nil
}

func (msv *machineSetValidator) ValidateUpdate(ctx context.Context, obj runtime.Object, _ runtime.Object) (result field.ErrorList) {
	return msv.doBasicValidation(ctx, obj)
}

func (msv *machineSetValidator) Validate(ctx context.Context, obj runtime.Object) (result field.ErrorList) {
	return msv.doBasicValidation(ctx, obj)
}

func (msv *machineSetValidator) doBasicValidation(ctx context.Context, obj runtime.Object) (result field.ErrorList) {
	machineSet := obj.(*v4alpha1.MachineSet)

	// we only want to do this validation if the creator of this object is external
	// we trust that internally-created objects (e.g. machineset from scheduledevent)
	// have already had the upstream sources of this configuration validated
	// and so it does not make sense to double-validate
	if len(machineSet.OwnerReferences) > 0 {
		return nil
	}

	// ensure the provider exists, err if not
	provObj, err := msv.providerGetter.Get(ctx, machineSet.Namespace, machineSet.Spec.Provider)
	if err != nil {
		result = append(result, field.Invalid(field.NewPath("spec", "provider"),
			machineSet.Spec.ProviderConfiguration, err.Error()))
		return
	}

	// convert the provOvj (types.Object from Mink) to a v4alpha1 Provider
	// then validate the providerconfiguration map on the machineset using that provider
	prov := provObj.(*v4alpha1.Provider)
	if err := ValidateProviderConfigurationMap(machineSet.Spec.ProviderConfiguration, prov); err != nil {
		return err
	}

	// ensure the machinetemplate exists
	_, err = msv.machineTemplateGetter.Get(ctx, machineSet.Namespace, machineSet.Spec.MachineTemplate)
	if err != nil {
		result = append(result, field.Invalid(field.NewPath("spec", "machineTemplate"),
			machineSet.Spec.MachineTemplate, err.Error()))
		return
	}

	// ensure the environment exists
	_, err = msv.environmentGetter.Get(ctx, machineSet.Namespace, machineSet.Spec.Environment)
	if err != nil {
		result = append(result, field.Invalid(field.NewPath("spec", "environment"),
			machineSet.Spec.Environment, err.Error()))
		return
	}

	return
}
