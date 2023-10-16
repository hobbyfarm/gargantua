package accesscodeservice

import (
	"context"
	"fmt"

	"github.com/golang/glog"
	hfClientset "github.com/hobbyfarm/gargantua/v3/pkg/client/clientset/versioned"
	"github.com/hobbyfarm/gargantua/v3/pkg/util"
	accessCodeProto "github.com/hobbyfarm/gargantua/v3/protos/accesscode"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	empty "google.golang.org/protobuf/types/known/emptypb"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/util/retry"
)

type GrpcAccessCodeServer struct {
	accessCodeProto.UnimplementedAccessCodeSvcServer
	hfClientSet hfClientset.Interface
	ctx         context.Context
}

func NewGrpcAccessCodeServer(hfClientSet hfClientset.Interface, ctx context.Context) *GrpcAccessCodeServer {
	return &GrpcAccessCodeServer{
		hfClientSet: hfClientSet,
		ctx:         ctx,
	}
}

func (a *GrpcAccessCodeServer) getOtac(id string) (*accessCodeProto.OneTimeAccessCode, error) {
	if len(id) == 0 {
		return &accessCodeProto.OneTimeAccessCode{}, fmt.Errorf("OTAC id passed in was empty")
	}
	obj, err := a.hfClientSet.HobbyfarmV1().OneTimeAccessCodes(util.GetReleaseNamespace()).Get(a.ctx, id, metav1.GetOptions{})
	if err != nil {
		return &accessCodeProto.OneTimeAccessCode{}, fmt.Errorf("error while retrieving OTAC by id: %s with error: %v", id, err)
	}

	return &accessCodeProto.OneTimeAccessCode{
		Id:                obj.Name,
		User:              obj.Spec.User,
		RedeemedTimestamp: obj.Spec.RedeemedTimestamp,
	}, nil
}

func (a *GrpcAccessCodeServer) GetOtac(ctx context.Context, gor *accessCodeProto.ResourceId) (*accessCodeProto.OneTimeAccessCode, error) {
	if len(gor.GetId()) == 0 {
		newErr := status.Newf(
			codes.InvalidArgument,
			"no id passed in",
		)
		newErr, wde := newErr.WithDetails(gor)
		if wde != nil {
			return &accessCodeProto.OneTimeAccessCode{}, wde
		}
		return &accessCodeProto.OneTimeAccessCode{}, newErr.Err()
	}

	otac, err := a.getOtac(gor.GetId())

	if err != nil {
		glog.V(2).Infof("%v is not an OTAC, returning status NotFound", err)
		newErr := status.Newf(
			codes.NotFound,
			"no OTAC %s found",
			gor.GetId(),
		)
		newErr, wde := newErr.WithDetails(gor)
		if wde != nil {
			return &accessCodeProto.OneTimeAccessCode{}, wde
		}
		return &accessCodeProto.OneTimeAccessCode{}, newErr.Err()
	}
	glog.V(2).Infof("retrieved OTAC %s", gor.GetId())
	return otac, nil
}

func (a *GrpcAccessCodeServer) UpdateOtac(ctx context.Context, otacRequest *accessCodeProto.OneTimeAccessCode) (*empty.Empty, error) {
	id := otacRequest.GetId()
	if id == "" {
		newErr := status.Newf(
			codes.InvalidArgument,
			"no ID passed in",
		)
		newErr, wde := newErr.WithDetails(otacRequest)
		if wde != nil {
			return &empty.Empty{}, wde
		}
		return &empty.Empty{}, newErr.Err()
	}

	retryErr := retry.RetryOnConflict(retry.DefaultRetry, func() error {
		otac, err := a.hfClientSet.HobbyfarmV1().OneTimeAccessCodes(util.GetReleaseNamespace()).Get(a.ctx, id, metav1.GetOptions{})
		if err != nil {
			newErr := status.Newf(
				codes.Internal,
				"error while retrieving OTAC %s",
				otacRequest.GetId(),
			)
			newErr, wde := newErr.WithDetails(otacRequest)
			if wde != nil {
				return wde
			}
			glog.Error(err)
			return newErr.Err()
		}

		otac.Spec.User = otacRequest.GetUser()
		otac.Spec.RedeemedTimestamp = otacRequest.GetRedeemedTimestamp()
		otac.Labels[util.UserLabel] = otacRequest.GetUser()

		_, updateErr := a.hfClientSet.HobbyfarmV1().OneTimeAccessCodes(util.GetReleaseNamespace()).Update(a.ctx, otac, metav1.UpdateOptions{})
		return updateErr
	})

	if retryErr != nil {
		newErr := status.Newf(
			codes.Internal,
			"error attempting to update",
		)
		newErr, wde := newErr.WithDetails(otacRequest)
		if wde != nil {
			return &empty.Empty{}, wde
		}
		return &empty.Empty{}, newErr.Err()
	}

	return &empty.Empty{}, nil
}
