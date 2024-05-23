package util

import (
	"context"

	"github.com/golang/glog"
	hferrors "github.com/hobbyfarm/gargantua/v3/pkg/errors"
	generalpb "github.com/hobbyfarm/gargantua/v3/protos/general"
	"google.golang.org/protobuf/types/known/wrapperspb"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type HfClientGet[T metav1.Object] interface {
	Get(ctx context.Context, id string, opts metav1.GetOptions) (T, error)
}

type GenericCacheRetriever[T metav1.Object] interface {
	Get(id string) (T, error)
}

func GenericHfGetter[T metav1.Object, G HfClientGet[T], C GenericCacheRetriever[T]](
	ctx context.Context,
	req *generalpb.GetRequest,
	getter G,
	cacheGetter C,
	resourceName string,
	hasSynced bool,
) (T, error) {
	var result T
	id := req.GetId()
	doLoadFromCache := req.GetLoadFromCache()
	if len(id) == 0 {
		glog.V(2).Infof("error no id provided for %s", resourceName)
		return result, hferrors.GrpcIdNotSpecifiedError(req)
	}
	var obj T
	var err error
	if !doLoadFromCache {
		obj, err = getter.Get(ctx, id, metav1.GetOptions{})
	} else if hasSynced {
		obj, err = cacheGetter.Get(id)
	} else {
		glog.V(2).Infof("error while retrieving %s by id: cache is not properly synced yet", resourceName)
		// our cache is not properly initialized yet ... returning status unavailable
		return result, hferrors.GrpcCacheError(req, resourceName)
	}
	if errors.IsNotFound(err) {
		return result, hferrors.GrpcNotFoundError(req, resourceName)
	} else if err != nil {
		glog.V(2).Infof("error while retrieving %s: %v", resourceName, err)
		return result, hferrors.GrpcGetError(req, resourceName, err)
	}

	return obj, nil
}

func GetOwnerReferences[T metav1.Object, G HfClientGet[T], C GenericCacheRetriever[T]](
	ctx context.Context,
	req *generalpb.GetRequest,
	getter G,
	cacheGetter C,
	resourceName string,
	hasSynced bool,
) (*generalpb.OwnerReferences, error) {
	obj, err := GenericHfGetter(ctx, req, getter, cacheGetter, resourceName, hasSynced)
	if err != nil {
		return &generalpb.OwnerReferences{}, err
	}

	preparedOwnerRefs := []*generalpb.OwnerReference{}

	for _, ownerRef := range obj.GetOwnerReferences() {
		hasManagingController := ownerRef.Controller
		hasBlockOwnerDeletion := ownerRef.BlockOwnerDeletion
		tempOwnerRef := &generalpb.OwnerReference{
			ApiVersion: ownerRef.APIVersion,
			Kind:       ownerRef.Kind,
			Name:       ownerRef.Name,
			Uid:        string(ownerRef.UID),
		}
		if hasManagingController != nil {
			tempOwnerRef.Controller = wrapperspb.Bool(*hasManagingController)
		}
		if hasBlockOwnerDeletion != nil {
			tempOwnerRef.BlockOwnerDeletion = wrapperspb.Bool(*hasBlockOwnerDeletion)
		}
		preparedOwnerRefs = append(preparedOwnerRefs, tempOwnerRef)
	}

	return &generalpb.OwnerReferences{OwnerReferences: preparedOwnerRefs}, nil
}
