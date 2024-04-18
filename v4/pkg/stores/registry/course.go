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

type courseValidator struct {
}

func NewCourseStorage(courseStrategy strategy.CompleteStrategy) (rest.Storage, error) {
	var cv = courseValidator{}

	return stores.NewBuilder(courseStrategy.Scheme(), &v4alpha1.Course{}).
		WithValidateCreate(cv).
		WithValidateUpdate(cv).
		WithCompleteCRUD(courseStrategy).Build(), nil
}

func (cv courseValidator) Validate(ctx context.Context, obj runtime.Object) field.ErrorList {
	return cv.doValidation(ctx, obj)
}

func (cv courseValidator) ValidateUpdate(ctx context.Context, obj runtime.Object, _ runtime.Object) field.ErrorList {
	return cv.doValidation(ctx, obj)
}

func (cv courseValidator) doValidation(ctx context.Context, obj runtime.Object) (result field.ErrorList) {
	course := obj.(*v4alpha1.Course)

	errs := ValidatePauseKeepaliveDurations(course.Spec.PauseDuration, course.Spec.KeepaliveDuration)

	if len(errs) > 0 {
		result = append(result, errs...)
	}

	return
}
