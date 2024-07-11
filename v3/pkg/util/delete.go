package util

import (
	"context"

	"github.com/golang/glog"
	hferrors "github.com/hobbyfarm/gargantua/v3/pkg/errors"
	generalpb "github.com/hobbyfarm/gargantua/v3/protos/general"
	"google.golang.org/grpc/codes"
	"google.golang.org/protobuf/types/known/emptypb"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type HfClientDelete interface {
	Delete(ctx context.Context, id string, opts metav1.DeleteOptions) error
	DeleteCollection(ctx context.Context, deleteOptions metav1.DeleteOptions, listOptions metav1.ListOptions) error
}

func DeleteHfResource(
	ctx context.Context,
	req *generalpb.ResourceId,
	clientDelete HfClientDelete,
	resourceName string,
) (*emptypb.Empty, error) {
	id := req.GetId()
	if len(id) == 0 {
		glog.V(2).Infof("error no id provided for %s", resourceName)
		return &emptypb.Empty{}, hferrors.GrpcIdNotSpecifiedError(req)
	}

	err := clientDelete.Delete(ctx, id, metav1.DeleteOptions{})
	if errors.IsNotFound(err) {
		return &emptypb.Empty{}, hferrors.GrpcNotFoundError(req, resourceName)
	} else if err != nil {
		glog.Errorf("error deleting %s %s: %s", resourceName, id, err)
		return &emptypb.Empty{}, hferrors.GrpcError(
			codes.Internal,
			"error deleting %s %s",
			req,
			resourceName,
			id,
		)
	}
	return &emptypb.Empty{}, nil
}

func DeleteHfCollection(
	ctx context.Context,
	listOptions *generalpb.ListOptions,
	clientDelete HfClientDelete,
	resourceName string,
) (*emptypb.Empty, error) {
	err := clientDelete.DeleteCollection(ctx, metav1.DeleteOptions{}, metav1.ListOptions{
		LabelSelector: listOptions.GetLabelSelector(),
	})
	if err != nil {
		return &emptypb.Empty{}, hferrors.GrpcError(
			codes.Internal,
			"error deleting %s",
			listOptions,
			resourceName,
		)
	}

	return &emptypb.Empty{}, nil
}
