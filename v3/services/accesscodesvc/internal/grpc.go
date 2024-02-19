package accesscodeservice

import (
	"context"
	"fmt"
	"time"

	"github.com/golang/glog"
	hfv1 "github.com/hobbyfarm/gargantua/v3/pkg/apis/hobbyfarm.io/v1"
	hfClientset "github.com/hobbyfarm/gargantua/v3/pkg/client/clientset/versioned"
	"github.com/hobbyfarm/gargantua/v3/pkg/errors"
	"github.com/hobbyfarm/gargantua/v3/pkg/util"
	accessCodeProto "github.com/hobbyfarm/gargantua/v3/protos/accesscode"
	"github.com/hobbyfarm/gargantua/v3/protos/general"
	"github.com/hobbyfarm/gargantua/v3/protos/user"
	"google.golang.org/grpc/codes"
	empty "google.golang.org/protobuf/types/known/emptypb"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/selection"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/util/retry"
)

type GrpcAccessCodeServer struct {
	accessCodeProto.UnimplementedAccessCodeSvcServer
	hfClientSet hfClientset.Interface
	userClient  user.UserSvcClient
}

func NewGrpcAccessCodeServer(hfClientSet hfClientset.Interface, userClient user.UserSvcClient) *GrpcAccessCodeServer {
	return &GrpcAccessCodeServer{
		hfClientSet: hfClientSet,
		userClient:  userClient,
	}
}

/**************************************************************************************************************
 * Resource oriented RPCs for AccessCodes
 *
 * The following functions implement the resource oriented RPCs for AccessCodes
 **************************************************************************************************************/

func (a *GrpcAccessCodeServer) CreateAc(ctx context.Context, cr *accessCodeProto.CreateAcRequest) (*empty.Empty, error) {

	if err := a.checkInputParamsForCreateAc(cr); err != nil {
		return &empty.Empty{}, err
	}
	acName := cr.GetAcName()
	seName := cr.GetSeName()
	seUid := types.UID(cr.GetSeUid())
	description := cr.GetDescription()
	scenarios := cr.GetScenarios()
	courses := cr.GetCourses()
	expiration := cr.GetExpiration()
	restrictedBind := cr.GetRestrictedBind()
	restrictedBindValue := cr.GetRestrictedBindValue()
	printable := cr.GetPrintable()

	ac := &hfv1.AccessCode{
		ObjectMeta: metav1.ObjectMeta{
			Name: acName,
			OwnerReferences: []metav1.OwnerReference{
				{
					APIVersion: "hobbyfarm.io/v1",
					Kind:       "ScheduledEvent",
					Name:       seName,
					UID:        seUid,
				},
			},
			Labels: map[string]string{
				util.ScheduledEventLabel: seName,
				util.AccessCodeLabel:     acName,
			},
		},
		Spec: hfv1.AccessCodeSpec{
			Code:           acName,
			Description:    description,
			Scenarios:      scenarios,
			Courses:        courses,
			Expiration:     expiration,
			RestrictedBind: restrictedBind,
			Printable:      printable,
		},
	}

	if restrictedBind {
		ac.Spec.RestrictedBindValue = restrictedBindValue
	}

	_, err := a.hfClientSet.HobbyfarmV1().AccessCodes(util.GetReleaseNamespace()).Create(ctx, ac, metav1.CreateOptions{})
	if err != nil {
		return &empty.Empty{}, err
	}

	return &empty.Empty{}, nil
}

func (a *GrpcAccessCodeServer) GetAc(ctx context.Context, id *general.ResourceId) (*accessCodeProto.AccessCode, error) {
	if len(id.GetId()) == 0 {
		return &accessCodeProto.AccessCode{}, errors.GrpcError(
			codes.InvalidArgument,
			"no id passed in",
			id,
		)
	}

	ac, err := a.hfClientSet.HobbyfarmV1().AccessCodes(util.GetReleaseNamespace()).Get(ctx, id.GetId(), metav1.GetOptions{})

	if err != nil {
		glog.V(2).Infof("error while retrieving accesscode: %v", err)
		return &accessCodeProto.AccessCode{}, errors.GrpcError(
			codes.Internal,
			"error while retrieving accesscode by id: %s with error: %v",
			id,
			id.GetId(),
			err,
		)
	}

	return &accessCodeProto.AccessCode{
		Id:                  ac.Name,
		Description:         ac.Spec.Description,
		Scenarios:           ac.Spec.Scenarios,
		Courses:             ac.Spec.Courses,
		Expiration:          ac.Spec.Expiration,
		RestrictedBind:      ac.Spec.RestrictedBind,
		RestrictedBindValue: ac.Spec.RestrictedBindValue,
		Printable:           ac.Spec.Printable,
		Labels:              ac.Labels,
	}, nil
}

func (a *GrpcAccessCodeServer) UpdateAc(ctx context.Context, acRequest *accessCodeProto.UpdateAccessCodeRequest) (*empty.Empty, error) {
	id := acRequest.GetId()
	if id == "" {
		return &empty.Empty{}, errors.GrpcError(
			codes.InvalidArgument,
			"no ID passed in",
			acRequest,
		)
	}

	description := acRequest.GetDescription()
	scenarios := acRequest.GetScenarios()
	courses := acRequest.GetCourses()
	expiration := acRequest.GetExpiration()
	restrictedBind := acRequest.GetRestrictedBind()
	printable := acRequest.GetPrintable()

	retryErr := retry.RetryOnConflict(retry.DefaultRetry, func() error {
		ac, err := a.hfClientSet.HobbyfarmV1().AccessCodes(util.GetReleaseNamespace()).Get(ctx, id, metav1.GetOptions{})
		if err != nil {
			glog.Error(err)
			return errors.GrpcError(
				codes.Internal,
				"error while retrieving accesscode %s",
				acRequest,
				acRequest.GetId(),
			)
		}

		// In the current implementation the code from  the access code spec equals the object's kubernetes name/id.
		// This ensures that access codes are unique.
		// Hence the .Spec.Code is immutable and should not be updated.
		// To update an access codes code name it has to be deleted and then recreated.
		// ac.Spec.Code = acRequest.GetId()

		// Only update values if they're input value is not empty/blank
		if description != "" {
			ac.Spec.Description = description
		}
		// To update scenarios and/or courses, at least one of these arrays needs to contain values
		if len(scenarios) > 0 || len(courses) > 0 {
			ac.Spec.Scenarios = scenarios
			ac.Spec.Courses = courses
		}
		if expiration != "" {
			ac.Spec.Expiration = expiration
		}
		if restrictedBind != nil {
			ac.Spec.RestrictedBind = restrictedBind.Value
		}
		// if restricted bind is disabled, make sure that restricted bind value is also empty...
		// else update restricted bind value if specified
		if !ac.Spec.RestrictedBind {
			ac.Spec.RestrictedBindValue = ""
		} else if ac.Spec.RestrictedBindValue == "" {
			ac.Spec.RestrictedBindValue = ac.ObjectMeta.Labels[util.ScheduledEventLabel]
		}
		if printable != nil {
			ac.Spec.Printable = printable.Value
		}

		_, updateErr := a.hfClientSet.HobbyfarmV1().AccessCodes(util.GetReleaseNamespace()).Update(ctx, ac, metav1.UpdateOptions{})
		return updateErr
	})

	if retryErr != nil {
		return &empty.Empty{}, errors.GrpcError(
			codes.Internal,
			"error attempting to update",
			acRequest,
		)
	}

	return &empty.Empty{}, nil
}

func (a *GrpcAccessCodeServer) DeleteAc(ctx context.Context, dr *general.ResourceId) (*empty.Empty, error) {
	acId := dr.GetId()
	if acId == "" {
		return &empty.Empty{}, errors.GrpcError(
			codes.InvalidArgument,
			"no ID passed in",
			dr,
		)
	}

	err := a.hfClientSet.HobbyfarmV1().AccessCodes(util.GetReleaseNamespace()).Delete(ctx, acId, metav1.DeleteOptions{})
	if err != nil {
		glog.Errorf("error deleting accesscode %s: %s", acId, err)
		return &empty.Empty{}, errors.GrpcError(
			codes.Internal,
			"error deleting accesscode %s",
			dr,
			acId,
		)
	}
	return &empty.Empty{}, nil
}

func (a *GrpcAccessCodeServer) DeleteCollectionAc(ctx context.Context, listOptions *general.ListOptions) (*empty.Empty, error) {

	// delete the access code for the corresponding ScheduledEvent
	err := a.hfClientSet.HobbyfarmV1().AccessCodes(util.GetReleaseNamespace()).DeleteCollection(ctx, metav1.DeleteOptions{}, metav1.ListOptions{
		LabelSelector: listOptions.GetLabelSelector(),
	})
	if err != nil {
		return &empty.Empty{}, errors.GrpcError(
			codes.Internal,
			"error deleting access codes",
			listOptions,
		)
	}

	return &empty.Empty{}, nil
}

func (a *GrpcAccessCodeServer) ListAc(ctx context.Context, listOptions *general.ListOptions) (*accessCodeProto.ListAcsResponse, error) {

	accessCodeList, err := a.hfClientSet.HobbyfarmV1().AccessCodes(util.GetReleaseNamespace()).List(ctx, metav1.ListOptions{
		LabelSelector: listOptions.GetLabelSelector(),
	})

	if err != nil {
		glog.Error(err)
		return &accessCodeProto.ListAcsResponse{}, errors.GrpcError(
			codes.Internal,
			"error retreiving access codes",
			listOptions,
		)
	}
	preparedAcs := []*accessCodeProto.AccessCode{}

	for _, accessCode := range accessCodeList.Items {

		if accessCode.Spec.Expiration != "" {
			expiration, err := time.Parse(time.UnixDate, accessCode.Spec.Expiration)

			if err != nil {
				return &accessCodeProto.ListAcsResponse{}, errors.GrpcError(
					codes.Internal,
					"error while parsing expiration time for access code %s %v",
					listOptions,
					accessCode.Name,
					err,
				)
			}

			if time.Now().After(expiration) { // if the access code is expired don't return any scenarios
				glog.V(4).Infof("access code %s was expired at %s", accessCode.Name, accessCode.Spec.Expiration)
				continue
			}
		}

		preparedAcs = append(preparedAcs, &accessCodeProto.AccessCode{
			Id:                  accessCode.Name,
			Description:         accessCode.Spec.Description,
			Scenarios:           accessCode.Spec.Scenarios,
			Courses:             accessCode.Spec.Courses,
			Expiration:          accessCode.Spec.Expiration,
			RestrictedBind:      accessCode.Spec.RestrictedBind,
			RestrictedBindValue: accessCode.Spec.RestrictedBindValue,
			Printable:           accessCode.Spec.Printable,
			Labels:              accessCode.Labels,
		})
	}

	glog.V(2).Infof("listed access codes")

	return &accessCodeProto.ListAcsResponse{AccessCodes: preparedAcs}, nil
}

/**************************************************************************************************************
 * Resource oriented RPCs for OneTimeAccessCodes
 *
 * The following functions implement the resource oriented RPCs for OneTimeAccessCodes
 **************************************************************************************************************/

func (a *GrpcAccessCodeServer) CreateOtac(ctx context.Context, cr *accessCodeProto.CreateOtacRequest) (*accessCodeProto.OneTimeAccessCode, error) {
	// Generate an access code that can not be guessed
	genName := ""
	for genParts := 0; genParts < 3; genParts++ {
		genName += util.GenerateResourceName("", util.RandStringRunes(16), 4)
	}
	genName = genName[1:]

	scheduledEventName := cr.GetSeName()
	if scheduledEventName == "" {
		return &accessCodeProto.OneTimeAccessCode{}, errors.GrpcError(
			codes.InvalidArgument,
			"error creating otac, se_name field blank",
			cr,
		)
	}

	scheduledUid := cr.GetSeUid()
	if scheduledUid == "" {
		return &accessCodeProto.OneTimeAccessCode{}, errors.GrpcError(
			codes.InvalidArgument,
			"error creating otac, se_uid field blank",
			cr,
		)
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
		MaxDuration:       otac.Spec.MaxDuration,
		Labels:            otac.Labels,
	}, nil
}

func (a *GrpcAccessCodeServer) GetOtac(ctx context.Context, id *general.ResourceId) (*accessCodeProto.OneTimeAccessCode, error) {
	if len(id.GetId()) == 0 {
		return &accessCodeProto.OneTimeAccessCode{}, errors.GrpcError(
			codes.InvalidArgument,
			"no id passed in",
			id,
		)
	}

	otac, err := a.getOtac(ctx, id.GetId())

	if err != nil {
		glog.V(2).Infof("%v is not an OTAC, returning status NotFound", err)
		return &accessCodeProto.OneTimeAccessCode{}, errors.GrpcError(
			codes.NotFound,
			"no OTAC %s found",
			id,
			id.GetId(),
		)
	}
	glog.V(2).Infof("retrieved OTAC %s", id.GetId())
	return otac, nil
}

func (a *GrpcAccessCodeServer) UpdateOtac(ctx context.Context, otacRequest *accessCodeProto.OneTimeAccessCode) (*empty.Empty, error) {
	id := otacRequest.GetId()
	if id == "" {
		return &empty.Empty{}, errors.GrpcError(
			codes.InvalidArgument,
			"no ID passed in",
			otacRequest,
		)
	}

	retryErr := retry.RetryOnConflict(retry.DefaultRetry, func() error {
		otac, err := a.hfClientSet.HobbyfarmV1().OneTimeAccessCodes(util.GetReleaseNamespace()).Get(ctx, id, metav1.GetOptions{})
		if err != nil {
			glog.Error(err)
			return errors.GrpcError(
				codes.Internal,
				"error while retrieving OTAC %s",
				otacRequest,
				otacRequest.GetId(),
			)
		}

		otac.Spec.User = otacRequest.GetUser()
		otac.Spec.RedeemedTimestamp = otacRequest.GetRedeemedTimestamp()
		otac.Spec.MaxDuration = otacRequest.GetMaxDuration()
		otac.Labels[util.UserLabel] = otacRequest.GetUser()

		_, updateErr := a.hfClientSet.HobbyfarmV1().OneTimeAccessCodes(util.GetReleaseNamespace()).Update(ctx, otac, metav1.UpdateOptions{})
		return updateErr
	})

	if retryErr != nil {
		return &empty.Empty{}, errors.GrpcError(
			codes.Internal,
			"error attempting to update",
			otacRequest,
		)
	}

	return &empty.Empty{}, nil
}

func (a *GrpcAccessCodeServer) DeleteOtac(ctx context.Context, dr *general.ResourceId) (*empty.Empty, error) {
	otacId := dr.GetId()
	if otacId == "" {
		return &empty.Empty{}, errors.GrpcError(
			codes.InvalidArgument,
			"no ID passed in",
			dr,
		)
	}

	err := a.hfClientSet.HobbyfarmV1().OneTimeAccessCodes(util.GetReleaseNamespace()).Delete(ctx, otacId, metav1.DeleteOptions{})
	if err != nil {
		glog.Errorf("error deleting otac %s: %s", otacId, err)
		return &empty.Empty{}, errors.GrpcError(
			codes.Internal,
			"error deleting otac %s",
			dr,
			otacId,
		)
	}
	return &empty.Empty{}, nil
}

func (a *GrpcAccessCodeServer) DeleteCollectionOtac(ctx context.Context, listOptions *general.ListOptions) (*empty.Empty, error) {

	// delete the access code for the corresponding ScheduledEvent
	err := a.hfClientSet.HobbyfarmV1().OneTimeAccessCodes(util.GetReleaseNamespace()).DeleteCollection(ctx, metav1.DeleteOptions{}, metav1.ListOptions{
		LabelSelector: listOptions.GetLabelSelector(),
	})
	if err != nil {
		return &empty.Empty{}, errors.GrpcError(
			codes.Internal,
			"error deleting otacs",
			listOptions,
		)
	}

	return &empty.Empty{}, nil
}

func (a *GrpcAccessCodeServer) ListOtac(ctx context.Context, listOptions *general.ListOptions) (*accessCodeProto.ListOtacsResponse, error) {
	// LabelSelector: fmt.Sprintf("%s=%s", util2.ScheduledEventLabel, id)
	otacList, err := a.hfClientSet.HobbyfarmV1().OneTimeAccessCodes(util.GetReleaseNamespace()).List(ctx, metav1.ListOptions{
		LabelSelector: listOptions.GetLabelSelector(),
	})

	if err != nil {
		glog.Error(err)
		return &accessCodeProto.ListOtacsResponse{}, errors.GrpcError(
			codes.Internal,
			"error retreiving OTACs",
			listOptions,
		)
	}

	preparedOtacs := []*accessCodeProto.OneTimeAccessCode{} // must be declared this way so as to JSON marshal into [] instead of null
	for _, otac := range otacList.Items {
		preparedOtacs = append(preparedOtacs, &accessCodeProto.OneTimeAccessCode{
			Id:                otac.Name,
			User:              otac.Spec.User,
			RedeemedTimestamp: otac.Spec.RedeemedTimestamp,
			MaxDuration:       otac.Spec.MaxDuration,
			Labels:            otac.Labels,
		})
	}

	glog.V(2).Infof("listed otacs")

	return &accessCodeProto.ListOtacsResponse{Otacs: preparedOtacs}, nil
}

/**************************************************************************************************************
 * Helper RPCs
 *
 * This section includes Helper RPCs exposed by the internal gRPC server.
 * These RPCs provide advanced functionalities beyond basic resource-related operations.
 **************************************************************************************************************/

func (a *GrpcAccessCodeServer) ValidateExistence(ctx context.Context, gor *general.ResourceId) (*accessCodeProto.ResourceValidation, error) {
	if len(gor.GetId()) == 0 {
		return &accessCodeProto.ResourceValidation{Valid: false}, errors.GrpcError(
			codes.InvalidArgument,
			"no id passed in",
			gor,
		)
	}

	_, err := a.hfClientSet.HobbyfarmV1().AccessCodes(util.GetReleaseNamespace()).Get(ctx, gor.GetId(), metav1.GetOptions{})
	if err != nil {
		// If AccessCode does not exist check if this might be an OTAC
		_, err := a.hfClientSet.HobbyfarmV1().OneTimeAccessCodes(util.GetReleaseNamespace()).Get(ctx, gor.GetId(), metav1.GetOptions{})
		if err != nil {
			return &accessCodeProto.ResourceValidation{Valid: false}, nil
		}
	}

	return &accessCodeProto.ResourceValidation{Valid: true}, nil
}

func (a *GrpcAccessCodeServer) GetAccessCodesWithOTACs(ctx context.Context, codeIds *accessCodeProto.ResourceIds) (*accessCodeProto.ListAcsResponse, error) {
	ids := codeIds.GetIds()
	otacReq, err := labels.NewRequirement(util.OneTimeAccessCodeLabel, selection.In, ids)
	if err != nil {
		return &accessCodeProto.ListAcsResponse{}, errors.GrpcError(
			codes.Internal,
			"Unable to create label selector from access code ids",
			codeIds,
		)
	}
	selector := labels.NewSelector()
	selector = selector.Add(*otacReq)
	selectorString := selector.String()

	// First get the oneTimeAccessCodes
	otacList, err := a.ListOtac(ctx, &general.ListOptions{LabelSelector: selectorString})

	if err != nil {
		return nil, err
	}

	//Append the value of onetime access codes to the list
	for _, otac := range otacList.Otacs {
		// @TODO: Query internal ScheduledEvent Service here!
		se, err := a.hfClientSet.HobbyfarmV1().ScheduledEvents(util.GetReleaseNamespace()).Get(ctx, otac.Labels[util.ScheduledEventLabel], metav1.GetOptions{})
		if err != nil {
			glog.Error(err)
			return &accessCodeProto.ListAcsResponse{}, errors.GrpcError(
				codes.Internal,
				"error retreiving scheduled event from OTAC: %v",
				codeIds,
				err,
			)
		}
		ids = append(ids, se.Spec.AccessCode)
	}

	// Update the label selector
	otacReq, err = labels.NewRequirement(util.OneTimeAccessCodeLabel, selection.In, ids)
	if err != nil {
		return &accessCodeProto.ListAcsResponse{}, errors.GrpcError(
			codes.Internal,
			"Unable to create label selector from access code ids",
			codeIds,
		)
	}
	selector = labels.NewSelector()
	selector = selector.Add(*otacReq)
	selectorString = selector.String()

	accessCodes, err := a.ListAc(ctx, &general.ListOptions{LabelSelector: selectorString})
	return accessCodes, err
}

func (a *GrpcAccessCodeServer) GetAccessCodeWithOTACs(ctx context.Context, codeId *general.ResourceId) (*accessCodeProto.AccessCode, error) {
	accessCodeId := codeId.GetId()
	if len(accessCodeId) == 0 {
		return &accessCodeProto.AccessCode{}, errors.GrpcError(
			codes.InvalidArgument,
			"no id passed in",
			codeId,
		)
	}

	accessCodeList, err := a.GetAccessCodesWithOTACs(ctx, &accessCodeProto.ResourceIds{Ids: []string{accessCodeId}})

	if err != nil {
		return &accessCodeProto.AccessCode{}, errors.GrpcError(
			codes.NotFound,
			"access code (%s) not found: %v",
			codeId,
			accessCodeId,
			err,
		)
	}

	accessCodes := accessCodeList.GetAccessCodes()

	if len(accessCodes) != 1 {
		return &accessCodeProto.AccessCode{}, errors.GrpcError(
			codes.Internal,
			"insane result found",
			codeId,
		)
	}

	return accessCodes[0], nil
}

/**************************************************************************************************************
 * Internal helper functions
 *
 * Internal helper functions which are only used within this file
 **************************************************************************************************************/

func (a *GrpcAccessCodeServer) getOtac(ctx context.Context, id string) (*accessCodeProto.OneTimeAccessCode, error) {
	if len(id) == 0 {
		return &accessCodeProto.OneTimeAccessCode{}, fmt.Errorf("OTAC id passed in was empty")
	}
	obj, err := a.hfClientSet.HobbyfarmV1().OneTimeAccessCodes(util.GetReleaseNamespace()).Get(ctx, id, metav1.GetOptions{})
	if err != nil {
		return &accessCodeProto.OneTimeAccessCode{}, fmt.Errorf("error while retrieving OTAC by id: %s with error: %v", id, err)
	}

	return &accessCodeProto.OneTimeAccessCode{
		Id:                obj.Name,
		User:              obj.Spec.User,
		RedeemedTimestamp: obj.Spec.RedeemedTimestamp,
		MaxDuration:       obj.Spec.MaxDuration,
		Labels:            obj.Labels,
	}, nil
}

func (a *GrpcAccessCodeServer) checkInputParamsForCreateAc(cr *accessCodeProto.CreateAcRequest) error {
	if cr.GetAcName() == "" ||
		cr.GetDescription() == "" ||
		cr.GetExpiration() == "" ||
		cr.GetSeName() == "" ||
		cr.GetSeUid() == "" ||
		(cr.GetRestrictedBind() && cr.GetRestrictedBindValue() == "") {

		return errors.GrpcError(
			codes.InvalidArgument,
			"error creating accesscode, required input field is blank",
			cr,
		)
	}
	return nil
}
