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

type userValidator struct {
}

func NewUserStatusStorage(scheme *runtime.Scheme, storage strategy.StatusUpdater) rest.Storage {
	return stores.NewStatus(scheme, storage)
}

func NewUserStorage(userStrategy strategy.CompleteStrategy) (rest.Storage, error) {
	var uv = userValidator{}

	return stores.NewBuilder(userStrategy.Scheme(), &v4alpha1.User{}).
		WithValidateUpdate(uv).
		WithCompleteCRUD(userStrategy).Build(), nil
}

func (uv userValidator) ValidateUpdate(ctx context.Context, obj runtime.Object, _ runtime.Object) (result field.ErrorList) {
	u := obj.(*v4alpha1.User)

	if len(u.Spec.Principals) == 0 {
		result = append(result, field.Invalid(field.NewPath("spec", "principals"),
			u.Spec.Principals, "must have at least one principal listed"))
		return
	}

	return
}
