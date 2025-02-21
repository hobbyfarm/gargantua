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

type ldapConfigValidator struct{}

func NewLdapConfigStatusStorage(scheme *runtime.Scheme, statusStrategy strategy.StatusUpdater) (rest.Storage, error) {
	return stores.NewStatus(scheme, statusStrategy), nil
}

func NewLdapConfigStorage(ldapConfigStrategy strategy.CompleteStrategy) (rest.Storage, error) {
	var lcv = &ldapConfigValidator{}

	return stores.NewBuilder(ldapConfigStrategy.Scheme(), &v4alpha1.LdapConfig{}).
		WithCompleteCRUD(ldapConfigStrategy).
		WithValidateCreate(lcv).
		WithValidateUpdate(lcv).Build(), nil
}

func (lcv ldapConfigValidator) ValidateUpdate(ctx context.Context, new runtime.Object, old runtime.Object) field.ErrorList {
	return lcv.doValidate(ctx, new)
}

func (lcv ldapConfigValidator) Validate(ctx context.Context, obj runtime.Object) field.ErrorList {
	return lcv.doValidate(ctx, obj)
}

func (lcv ldapConfigValidator) doValidate(ctx context.Context, obj runtime.Object) (result field.ErrorList) {
	config := obj.(*v4alpha1.LdapConfig)

	if config.Spec.BindUsername == "" {
		result = append(result, field.Invalid(field.NewPath("spec", "bindUsername"),
			config.Spec.BindUsername, "bindUsername is required"))
	}

	if config.Spec.BindPasswordSecret == "" {
		result = append(result, field.Invalid(field.NewPath("spec", "bindPasswordSecret"),
			config.Spec.BindPasswordSecret, "bindPasswordSecret is required"))
	}

	if config.Spec.UsernameField == "" {
		result = append(result, field.Invalid(field.NewPath("spec", "usernameField"),
			config.Spec.UsernameField, "usernameField is required"))
	}

	if config.Spec.DisplayNameField == "" {
		result = append(result, field.Invalid(field.NewPath("spec", "displayNameField"),
			config.Spec.DisplayNameField, "displayNameField is required"))
	}

	if len(result) > 0 {
		return result
	}

	return nil
}
