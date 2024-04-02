package registry

import (
	"context"
	"fmt"
	"github.com/hobbyfarm/gargantua/v4/pkg/apis/hobbyfarm.io/v4alpha1"
	"github.com/hobbyfarm/mink/pkg/strategy"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/util/validation/field"
	"k8s.io/apiserver/pkg/storage"
	"net/http"
)

func HandleConflictList(ctx context.Context, namespace string, lister strategy.Lister, labelSelector map[string]string, objName string) *errors.StatusError {
	list, err := lister.List(ctx, namespace, storage.ListOptions{
		Predicate: storage.SelectionPredicate{
			Label: labels.SelectorFromSet(labelSelector),
		},
	})
	if err != nil {
		return errors.NewInternalError(err)
	}

	if poml, ok := list.(*metav1.PartialObjectMetadataList); ok {
		if len(poml.Items) == 0 {
			return nil
		}

		var out = make([]string, len(poml.Items))
		for i, v := range poml.Items {
			out[i] = v.Name
		}

		return &errors.StatusError{
			ErrStatus: metav1.Status{
				Status: metav1.StatusFailure,
				Code:   http.StatusConflict,
				Message: fmt.Sprintf("cannot delete referenced object. reference list %s contains items %v",
					list.GetObjectKind().GroupVersionKind().String(), out),
			},
		}
	} else {
		return errors.NewInternalError(fmt.Errorf("error converting %s to PartialObjectMetadataList for object %s",
			list.GetObjectKind().GroupVersionKind().String(), objName))
	}
}

func ValidateProviderConfigurationMap(confMap map[string]string, provider *v4alpha1.Provider) (result field.ErrorList) {
	for k, v := range confMap {
		// lookup config item in provider
		item, ok := provider.Spec.ProviderConfiguration[k]
		if !ok {
			result = append(result, field.Invalid(field.NewPath(k), v,
				fmt.Sprintf("configuration item with key %s not found in provider %s", k, provider.Name)))
			continue // don't validate something that we can't locate
		}

		if err := item.Validate(v); err != nil {
			result = append(result, field.Invalid(field.NewPath(k), v, fmt.Sprintf("invalid value for configuration item %s: %s",
				k, err.Error())))
		}
	}

	return
}
