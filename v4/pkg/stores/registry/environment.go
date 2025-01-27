package registry

import (
	"context"
	"github.com/hobbyfarm/gargantua/v4/pkg/apis/hobbyfarm.io/v4alpha1"
	labels2 "github.com/hobbyfarm/gargantua/v4/pkg/labels"
	"github.com/hobbyfarm/mink/pkg/stores"
	"github.com/hobbyfarm/mink/pkg/strategy"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/validation/field"
	"k8s.io/apiserver/pkg/registry/rest"
)

type environmentValidator struct {
	providerGetter       strategy.Getter
	machineSetLister     strategy.Lister
	machineLister        strategy.Lister
	scheduledEventLister strategy.Lister
}

func NewEnvironmentStatusStorage(scheme *runtime.Scheme, updater strategy.StatusUpdater) rest.Storage {
	return stores.NewStatus(scheme, updater)
}

func NewEnvironmentStorage(
	environmentStrategy strategy.CompleteStrategy,
	providerGetter strategy.Getter,
	machineSetLister strategy.Lister,
	machineLister strategy.Lister,
	scheduledEventLister strategy.Lister,
) (rest.Storage, error) {
	ev := &environmentValidator{
		providerGetter:       providerGetter,
		machineSetLister:     machineSetLister,
		machineLister:        machineLister,
		scheduledEventLister: scheduledEventLister,
	}

	return stores.NewBuilder(environmentStrategy.Scheme(), &v4alpha1.Environment{}).
		WithCompleteCRUD(environmentStrategy).
		WithValidateCreate(ev).
		WithValidateUpdate(ev).
		WithValidateDelete(ev).Build(), nil
}

func (ev *environmentValidator) ValidateDelete(ctx context.Context, obj runtime.Object) *errors.StatusError {
	// can only delete environment if not being referenced in other objects
	// namely, machines, machinesets, and upcoming scheduledevents
	env := obj.(*v4alpha1.Environment)

	environmentSelector := map[string]string{
		labels2.EnvironmentLabel: env.Name,
	}

	if err := HandleConflictList(ctx, env.Namespace, ev.machineSetLister, environmentSelector, env.Name); err != nil {
		return err
	}

	if err := HandleConflictList(ctx, env.Namespace, ev.machineLister, environmentSelector, env.Name); err != nil {
		return err
	}

	// TODO: Make sure that we add logic to the controllers to set the complete label to true
	environmentSelector[labels2.ScheduledEventCompleteLabel] = "True"
	if err := HandleConflictList(ctx, env.Namespace, ev.scheduledEventLister, environmentSelector, env.Name); err != nil {
		return err
	}

	return nil
}

func (ev *environmentValidator) ValidateUpdate(ctx context.Context, obj runtime.Object, _ runtime.Object) (result field.ErrorList) {
	return ev.doBasicValidations(ctx, obj)
}

func (ev *environmentValidator) Validate(ctx context.Context, obj runtime.Object) (result field.ErrorList) {
	return ev.doBasicValidations(ctx, obj)
}

func (ev *environmentValidator) doBasicValidations(ctx context.Context, obj runtime.Object) (result field.ErrorList) {
	env := obj.(*v4alpha1.Environment)

	// Ensure the provider exists
	prov, err := ev.getProvider(ctx, env)
	if err != nil {
		return err
	}

	// validate configuration
	if err := ValidateProviderConfigurationMap(env.Spec.ProviderConfiguration, prov); err != nil {
		return err
	}

	// for each machinetemplate, validate any configuration items
	// map[string]map[string]string can be seen as:
	// template_name:
	//   key: value
	for _, v := range env.Spec.TemplateConfiguration {
		if err := ValidateProviderConfigurationMap(v, prov); err != nil {
			result = append(result, err...)
		}
	}

	return
}

func (ev *environmentValidator) getProvider(ctx context.Context, env *v4alpha1.Environment) (*v4alpha1.Provider, field.ErrorList) {
	// Ensure the provider exists
	provObj, err := ev.providerGetter.Get(ctx, env.Namespace, env.Spec.Provider)
	if err != nil {
		return nil, field.ErrorList{
			field.Invalid(field.NewPath("spec", "provider"), env.Spec.Provider, err.Error()),
		}
	}

	prov := provObj.(*v4alpha1.Provider)

	return prov, nil
}
