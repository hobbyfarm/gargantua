package util

import (
	hferrors "github.com/hobbyfarm/gargantua/v3/pkg/errors"
	generalpb "github.com/hobbyfarm/gargantua/v3/protos/general"
	"google.golang.org/grpc/codes"
	"google.golang.org/protobuf/types/known/emptypb"
	"k8s.io/client-go/util/workqueue"
)

func AddToWorkqueue(workqueue workqueue.Interface, req *generalpb.ResourceId) (*emptypb.Empty, error) {
	if workqueue == nil {
		return &emptypb.Empty{}, hferrors.GrpcError(
			codes.Internal,
			"error adding item to workqueue: workqueue is nil",
			req,
		)
	}
	workqueue.Add(req.GetId())
	return &emptypb.Empty{}, nil
}
