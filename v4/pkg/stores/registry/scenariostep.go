package registry

import (
	"context"
	"github.com/hobbyfarm/gargantua/v4/pkg/apis/hobbyfarm.io/v4alpha1"
	"github.com/hobbyfarm/mink/pkg/stores"
	"github.com/hobbyfarm/mink/pkg/strategy"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apiserver/pkg/registry/rest"
	"net/http"
)

type scenarioStepValidator struct{}

func NewScenarioStepStatusStorage(scheme *runtime.Scheme, storage strategy.StatusUpdater) rest.Storage {
	return stores.NewStatus(scheme, storage)
}

func NewScenarioStepStorage(scenarioStepStrategy strategy.CompleteStrategy) (rest.Storage, error) {
	var ssv = scenarioStepValidator{}

	return stores.NewBuilder(scenarioStepStrategy.Scheme(), &v4alpha1.ScenarioStep{}).
		WithValidateDelete(ssv).
		WithCompleteCRUD(scenarioStepStrategy).Build(), nil
}

func (ssv scenarioStepValidator) ValidateDelete(ctx context.Context, obj runtime.Object) *errors.StatusError {
	ss := obj.(*v4alpha1.ScenarioStep)

	if len(ss.Status.ReferringScenarios) > 0 {
		return &errors.StatusError{
			ErrStatus: metav1.Status{
				Status:  metav1.StatusFailure,
				Message: "cannot delete ScenarioStep still referenced by scenarios (status.referringScenarios > 0)",
				Code:    http.StatusConflict,
			},
		}
	}

	return nil
}
