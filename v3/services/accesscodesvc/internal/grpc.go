package accesscodeservice

import (
	"context"
	"fmt"

	"github.com/golang/glog"
	hfv1 "github.com/hobbyfarm/gargantua/v3/pkg/apis/hobbyfarm.io/v1"
	hfClientset "github.com/hobbyfarm/gargantua/v3/pkg/client/clientset/versioned"
	"github.com/hobbyfarm/gargantua/v3/pkg/util"
	accessCodeProto "github.com/hobbyfarm/gargantua/v3/protos/accesscode"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	empty "google.golang.org/protobuf/types/known/emptypb"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
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
		Labels:            obj.Labels,
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

func (a *GrpcAccessCodeServer) ValidateExistence(ctx context.Context, gor *accessCodeProto.ResourceId) (*accessCodeProto.ResourceValidation, error) {
	if len(gor.GetId()) == 0 {
		newErr := status.Newf(
			codes.InvalidArgument,
			"no id passed in",
		)
		newErr, wde := newErr.WithDetails(gor)
		if wde != nil {
			return &accessCodeProto.ResourceValidation{Valid: false}, wde
		}
		return &accessCodeProto.ResourceValidation{Valid: false}, newErr.Err()
	}

	_, err := a.hfClientSet.HobbyfarmV1().AccessCodes(util.GetReleaseNamespace()).Get(a.ctx, gor.GetId(), metav1.GetOptions{})
	if err != nil {
		// If AccessCode does not exist check if this might be an OTAC
		_, err := a.hfClientSet.HobbyfarmV1().OneTimeAccessCodes(util.GetReleaseNamespace()).Get(a.ctx, gor.GetId(), metav1.GetOptions{})
		if err != nil {
			return &accessCodeProto.ResourceValidation{Valid: false}, nil
		}
	}

	return &accessCodeProto.ResourceValidation{Valid: true}, nil
}

func (a *GrpcAccessCodeServer) ListOtac(ctx context.Context, listOptions *accessCodeProto.ListOptions) (*accessCodeProto.ListOtacsResponse, error) {
	// LabelSelector: fmt.Sprintf("%s=%s", util2.ScheduledEventLabel, id)
	otacList, err := a.hfClientSet.HobbyfarmV1().OneTimeAccessCodes(util.GetReleaseNamespace()).List(ctx, metav1.ListOptions{
		LabelSelector: listOptions.GetLabelSelector(),
	})

	if err != nil {
		glog.Error(err)
		newErr := status.Newf(
			codes.Internal,
			"error retreiving OTACs",
		)
		return &accessCodeProto.ListOtacsResponse{}, newErr.Err()
	}

	preparedOtacs := []*accessCodeProto.OneTimeAccessCode{} // must be declared this way so as to JSON marshal into [] instead of null
	for _, otac := range otacList.Items {
		preparedOtacs = append(preparedOtacs, &accessCodeProto.OneTimeAccessCode{
			Id:                otac.Name,
			User:              otac.Spec.User,
			RedeemedTimestamp: otac.Spec.RedeemedTimestamp,
			Labels:            otac.Labels,
		})
	}

	glog.V(2).Infof("listed otacs")

	return &accessCodeProto.ListOtacsResponse{Otacs: preparedOtacs}, nil
}

func (a *GrpcAccessCodeServer) CreateOtac(ctx context.Context, cr *accessCodeProto.CreateOtacRequest) (*accessCodeProto.OneTimeAccessCode, error) {
	// Generate an access code that can not be guessed
	genName := ""
	for genParts := 0; genParts < 3; genParts++ {
		genName += util.GenerateResourceName("", util.RandStringRunes(16), 4)
	}
	genName = genName[1:]

	scheduledEventName := cr.GetSeName()
	if scheduledEventName == "" {
		newErr := status.Newf(
			codes.InvalidArgument,
			"error creating otac, se_name field blank",
		)
		newErr, wde := newErr.WithDetails(cr)
		if wde != nil {
			return &accessCodeProto.OneTimeAccessCode{}, wde
		}
		return &accessCodeProto.OneTimeAccessCode{}, newErr.Err()
	}

	scheduledUid := cr.GetSeUid()
	if scheduledUid == "" {
		newErr := status.Newf(
			codes.InvalidArgument,
			"error creating otac, se_uid field blank",
		)
		newErr, wde := newErr.WithDetails(cr)
		if wde != nil {
			return &accessCodeProto.OneTimeAccessCode{}, wde
		}
		return &accessCodeProto.OneTimeAccessCode{}, newErr.Err()
	}

	otac := &hfv1.OneTimeAccessCode{
		ObjectMeta: metav1.ObjectMeta{
			Name: genName,
			OwnerReferences: []metav1.OwnerReference{
				{
					APIVersion: "hobbyfarm.io/v1",
					Kind:       "ScheduledEvent",
					Name:       scheduledEventName,
					UID:        types.UID(scheduledUid),
				},
			},
			Labels: map[string]string{
				util.UserLabel:              "",
				util.ScheduledEventLabel:    scheduledEventName,
				util.OneTimeAccessCodeLabel: genName,
			},
		},
		Spec: hfv1.OneTimeAccessCodeSpec{
			User:              "",
			RedeemedTimestamp: "",
		},
	}
	otac, err := a.hfClientSet.HobbyfarmV1().OneTimeAccessCodes(util.GetReleaseNamespace()).Create(ctx, otac, metav1.CreateOptions{})
	if err != nil {
		glog.Errorf("error creating one time access code %v", err)
		// error handling
	}
	return &accessCodeProto.OneTimeAccessCode{
		Id:                otac.Name,
		User:              otac.Spec.User,
		RedeemedTimestamp: otac.Spec.RedeemedTimestamp,
		Labels:            otac.Labels,
	}, nil
}

func (a *GrpcAccessCodeServer) DeleteOtac(ctx context.Context, dr *accessCodeProto.ResourceId) (*empty.Empty, error) {
	otacId := dr.GetId()
	if otacId == "" {
		newErr := status.Newf(
			codes.InvalidArgument,
			"no ID passed in",
		)
		newErr, wde := newErr.WithDetails(dr)
		if wde != nil {
			return &empty.Empty{}, wde
		}
		return &empty.Empty{}, newErr.Err()
	}

	err := a.hfClientSet.HobbyfarmV1().OneTimeAccessCodes(util.GetReleaseNamespace()).Delete(ctx, otacId, metav1.DeleteOptions{})
	if err != nil {
		newErr := status.Newf(
			codes.Internal,
			"error deleting otac %s",
			otacId,
		)
		newErr, wde := newErr.WithDetails(dr)
		if wde != nil {
			return &empty.Empty{}, wde
		}
		glog.Errorf("error deleting otac %s: %s", otacId, err)
		return &empty.Empty{}, newErr.Err()
	}
	return &empty.Empty{}, nil
}

func (a *GrpcAccessCodeServer) DeleteCollectionOtac(ctx context.Context, listOptions *accessCodeProto.ListOptions) (*empty.Empty, error) {

	// delete the access code for the corresponding ScheduledEvent
	err := a.hfClientSet.HobbyfarmV1().AccessCodes(util.GetReleaseNamespace()).DeleteCollection(ctx, metav1.DeleteOptions{}, metav1.ListOptions{
		LabelSelector: listOptions.GetLabelSelector(),
	})
	if err != nil {
		newErr := status.Newf(
			codes.Internal,
			"error deleting otacs",
		)
		newErr, wde := newErr.WithDetails(listOptions)
		if wde != nil {
			return &empty.Empty{}, wde
		}
		return &empty.Empty{}, newErr.Err()
	}

	return &empty.Empty{}, nil
}
