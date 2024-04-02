package registry

import (
	"context"
	"github.com/hobbyfarm/gargantua/v4/pkg/apis/hobbyfarm.io/v4alpha1"
	hfLabels "github.com/hobbyfarm/gargantua/v4/pkg/labels"
	"github.com/hobbyfarm/mink/pkg/stores"
	"github.com/hobbyfarm/mink/pkg/strategy"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/validation/field"
	"k8s.io/apiserver/pkg/registry/rest"
)

var (
	RequiredMachineInformationItems = []string{
		"primary_address",
	}

	RequiredProviderConfigurationItems = []string{}
)

type providerValidator struct {
	machineSetLister  strategy.Lister
	machineLister     strategy.Lister
	environmentLister strategy.Lister
}

func NewProviderStorage(
	providerStrategy strategy.CompleteStrategy,
	machineSetLister strategy.Lister,
	machineLister strategy.Lister,
	environmentLister strategy.Lister,
) (rest.Storage, error) {
	pv := &providerValidator{
		machineLister:     machineLister,
		machineSetLister:  machineSetLister,
		environmentLister: environmentLister,
	}

	return stores.NewBuilder(providerStrategy.Scheme(), &v4alpha1.Provider{}).
		WithCompleteCRUD(providerStrategy).
		WithValidateCreate(pv).
		WithValidateUpdate(pv).
		WithValidateDelete(pv).Build(), nil
}

func (pv *providerValidator) ValidateDelete(ctx context.Context, obj runtime.Object) *errors.StatusError {
	p := obj.(*v4alpha1.Provider)

	// before we delete the provider, need to ensure nobody is using it
	// this means no environments, machineset, etc. can be using it
	var providerSelector = map[string]string{
		hfLabels.ProviderLabel: p.Name,
	}

	if err := HandleConflictList(ctx, p.Namespace, pv.environmentLister, providerSelector, p.Name); err != nil {
		return err
	}

	if err := HandleConflictList(ctx, p.Namespace, pv.machineSetLister, providerSelector, p.Name); err != nil {
		return err
	}

	if err := HandleConflictList(ctx, p.Namespace, pv.machineLister, providerSelector, p.Name); err != nil {
		return err
	}

	return nil
}

func (pv *providerValidator) ValidateUpdate(_ context.Context, obj runtime.Object, _ runtime.Object) (result field.ErrorList) {
	p := obj.(*v4alpha1.Provider)

	return pv.validateRequiredProperties(p)
}

func (pv *providerValidator) Validate(_ context.Context, obj runtime.Object) (result field.ErrorList) {
	p := obj.(*v4alpha1.Provider)

	return pv.validateRequiredProperties(p)
}

func (pv *providerValidator) validateRequiredProperties(p *v4alpha1.Provider) (result field.ErrorList) {
	result = append(result, pv.validateConfigurationItems(p)...)
	result = append(result, pv.validateMachineInformation(p)...)

	return
}

func (pv *providerValidator) validateConfigurationItems(p *v4alpha1.Provider) (result field.ErrorList) {
	// ensure that the provider has all the required configuration items
	for _, item := range RequiredProviderConfigurationItems {
		if _, ok := p.Spec.ProviderConfiguration[item]; !ok {
			result = append(result, field.NotFound(field.NewPath("spec", "providerConfiguration"), item))
		}
	}

	return
}

func (pv *providerValidator) validateMachineInformation(p *v4alpha1.Provider) (result field.ErrorList) {
	// ensure that the provider has all the required machine information items
	for _, item := range RequiredMachineInformationItems {
		if _, ok := p.Spec.MachineInformation[item]; !ok {
			result = append(result, field.NotFound(field.NewPath("spec", "machineInformation"), item))
		}
	}

	return
}
