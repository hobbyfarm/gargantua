package util

import (
	"context"

	hferrors "github.com/hobbyfarm/gargantua/v3/pkg/errors"
	"github.com/hobbyfarm/gargantua/v3/protos/general"
	"google.golang.org/grpc/codes"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
)

type HfClientList[T any] interface {
	List(ctx context.Context, opts metav1.ListOptions) (T, error)
}

type GenericLister[T any] interface {
	List(labels.Selector) ([]*T, error)
}

func ListByHfClient[T any, L HfClientList[T]](ctx context.Context, listOptions *general.ListOptions, lister L, resourcename string) (T, error) {
	labelSelectorString := listOptions.GetLabelSelector()
	objList, err := lister.List(ctx, metav1.ListOptions{
		LabelSelector: labelSelectorString,
	})
	if err != nil {
		return objList, hferrors.GrpcListError(listOptions, resourcename)
	}
	return objList, nil
}

func ListByCache[T any, L GenericLister[T]](listOptions *general.ListOptions, lister L, resourcename string, hasSynced bool) ([]T, error) {
	labelSelectorString := listOptions.GetLabelSelector()
	labelSelector, err := labels.Parse(labelSelectorString)
	if err != nil {
		return []T{}, hferrors.GrpcError(
			codes.Internal,
			"error parsing label selector",
			listOptions,
		)
	}
	vmClaims := []T{}
	if hasSynced {
		vmcList, err := lister.List(labelSelector)
		if err != nil {
			return []T{}, hferrors.GrpcListError(listOptions, resourcename)
		}
		for _, vmc := range vmcList {
			vmClaims = append(vmClaims, *vmc)
		}
		return vmClaims, nil
	} else {
		// our cache is not properly initialized yet ... returning status unavailable
		return []T{}, hferrors.GrpcCacheError(listOptions, resourcename)
	}
}
