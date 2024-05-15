package util

import (
	"github.com/golang/protobuf/ptypes/empty"
	hferrors "github.com/hobbyfarm/gargantua/v3/pkg/errors"
	"github.com/hobbyfarm/gargantua/v3/protos/general"
	"google.golang.org/grpc/codes"
	"k8s.io/client-go/util/workqueue"
)

func AddToWorkqueue(workqueue workqueue.Interface, req *general.ResourceId) (*empty.Empty, error) {
	if workqueue == nil {
		return &empty.Empty{}, hferrors.GrpcError(
			codes.Internal,
			"error adding item to workqueue: workqueue is nil",
			req,
		)
	}
	workqueue.Add(req.GetId())
	return &empty.Empty{}, nil
}
