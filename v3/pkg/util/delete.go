package util

import (
	"context"

	"github.com/golang/glog"
	hferrors "github.com/hobbyfarm/gargantua/v3/pkg/errors"
	"github.com/hobbyfarm/gargantua/v3/protos/general"
	"google.golang.org/grpc/codes"
	empty "google.golang.org/protobuf/types/known/emptypb"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type HfClientDelete interface {
	Delete(ctx context.Context, id string, opts metav1.DeleteOptions) error
	DeleteCollection(ctx context.Context, deleteOptions metav1.DeleteOptions, listOptions metav1.ListOptions) error
}

func DeleteHfResource(
	ctx context.Context,
	req *general.ResourceId,
	clientDelete HfClientDelete,
	resourceName string,
) (*empty.Empty, error) {
	id := req.GetId()
	if len(id) == 0 {
		glog.V(2).Infof("error no id provided for %s", resourceName)
		return &empty.Empty{}, hferrors.GrpcIdNotSpecifiedError(req)
	}

	err := clientDelete.Delete(ctx, id, metav1.DeleteOptions{})
	if err != nil {
		glog.Errorf("error deleting %s %s: %s", resourceName, id, err)
		return &empty.Empty{}, hferrors.GrpcError(
			codes.Internal,
			"error deleting %s %s",
			req,
			resourceName,
			id,
		)
	}
	return &empty.Empty{}, nil
}

func DeleteHfCollection(
	ctx context.Context,
	listOptions *general.ListOptions,
	clientDelete HfClientDelete,
	resourceName string,
) (*empty.Empty, error) {
	err := clientDelete.DeleteCollection(ctx, metav1.DeleteOptions{}, metav1.ListOptions{
		LabelSelector: listOptions.GetLabelSelector(),
	})
	if err != nil {
		return &empty.Empty{}, hferrors.GrpcError(
			codes.Internal,
			"error deleting %s",
			listOptions,
			resourceName,
		)
	}

	return &empty.Empty{}, nil
}
