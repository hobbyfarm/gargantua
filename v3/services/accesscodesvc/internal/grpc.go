package accesscodeservice

import (
	"context"
	"time"

	"github.com/golang/glog"
	hfv1 "github.com/hobbyfarm/gargantua/v3/pkg/apis/hobbyfarm.io/v1"
	hfClientset "github.com/hobbyfarm/gargantua/v3/pkg/client/clientset/versioned"
	hfClientsetv1 "github.com/hobbyfarm/gargantua/v3/pkg/client/clientset/versioned/typed/hobbyfarm.io/v1"
	hfInformers "github.com/hobbyfarm/gargantua/v3/pkg/client/informers/externalversions"
	listersv1 "github.com/hobbyfarm/gargantua/v3/pkg/client/listers/hobbyfarm.io/v1"
	hferrors "github.com/hobbyfarm/gargantua/v3/pkg/errors"
	hflabels "github.com/hobbyfarm/gargantua/v3/pkg/labels"
	"github.com/hobbyfarm/gargantua/v3/pkg/util"
	accesscodepb "github.com/hobbyfarm/gargantua/v3/protos/accesscode"
	generalpb "github.com/hobbyfarm/gargantua/v3/protos/general"
	scheduledeventpb "github.com/hobbyfarm/gargantua/v3/protos/scheduledevent"
	"google.golang.org/grpc/codes"
	"google.golang.org/protobuf/types/known/emptypb"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/selection"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/retry"
)

type GrpcAccessCodeServer struct {
	accesscodepb.UnimplementedAccessCodeSvcServer
	acClient    hfClientsetv1.AccessCodeInterface
	acLister    listersv1.AccessCodeLister
	acSynced    cache.InformerSynced
	otacClient  hfClientsetv1.OneTimeAccessCodeInterface
	otacLister  listersv1.OneTimeAccessCodeLister
	otacSynced  cache.InformerSynced
	eventClient scheduledeventpb.ScheduledEventSvcClient
}

func NewGrpcAccessCodeServer(hfClientSet hfClientset.Interface, hfInformerFactory hfInformers.SharedInformerFactory, eventClient scheduledeventpb.ScheduledEventSvcClient) *GrpcAccessCodeServer {
	return &GrpcAccessCodeServer{
		acClient:    hfClientSet.HobbyfarmV1().AccessCodes(util.GetReleaseNamespace()),
		acLister:    hfInformerFactory.Hobbyfarm().V1().AccessCodes().Lister(),
		acSynced:    hfInformerFactory.Hobbyfarm().V1().AccessCodes().Informer().HasSynced,
		otacClient:  hfClientSet.HobbyfarmV1().OneTimeAccessCodes(util.GetReleaseNamespace()),
		otacLister:  hfInformerFactory.Hobbyfarm().V1().OneTimeAccessCodes().Lister(),
		otacSynced:  hfInformerFactory.Hobbyfarm().V1().OneTimeAccessCodes().Informer().HasSynced,
		eventClient: eventClient,
	}
}

/**************************************************************************************************************
 * Resource oriented RPCs for AccessCodes
 *
 * The following functions implement the resource oriented RPCs for AccessCodes
 **************************************************************************************************************/

func (a *GrpcAccessCodeServer) CreateAc(ctx context.Context, cr *accesscodepb.CreateAcRequest) (*emptypb.Empty, error) {

	if err := a.checkInputParamsForCreateAc(cr); err != nil {
		return &emptypb.Empty{}, err
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
				hflabels.ScheduledEventLabel: seName,
				hflabels.AccessCodeLabel:     acName,
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

	_, err := a.acClient.Create(ctx, ac, metav1.CreateOptions{})
	if err != nil {
		return &emptypb.Empty{}, err
	}

	return &emptypb.Empty{}, nil
}

func (a *GrpcAccessCodeServer) GetAc(ctx context.Context, req *generalpb.GetRequest) (*accesscodepb.AccessCode, error) {
	ac, err := util.GenericHfGetter(ctx, req, a.acClient, a.acLister.AccessCodes(util.GetReleaseNamespace()), "access code", a.acSynced())
	if err != nil {
		return &accesscodepb.AccessCode{}, err
	}

	return &accesscodepb.AccessCode{
		Id:                  ac.Name,
		Uid:                 string(ac.UID),
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

func (a *GrpcAccessCodeServer) UpdateAc(ctx context.Context, acRequest *accesscodepb.UpdateAccessCodeRequest) (*emptypb.Empty, error) {
	id := acRequest.GetId()
	if id == "" {
		return &emptypb.Empty{}, hferrors.GrpcError(
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
		ac, err := a.acClient.Get(ctx, id, metav1.GetOptions{})
		if err != nil {
			glog.Error(err)
			return hferrors.GrpcError(
				codes.Internal,
				"error while retrieving access code %s",
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
			ac.Spec.RestrictedBindValue = ac.ObjectMeta.Labels[hflabels.ScheduledEventLabel]
		}
		if printable != nil {
			ac.Spec.Printable = printable.Value
		}

		_, updateErr := a.acClient.Update(ctx, ac, metav1.UpdateOptions{})
		return updateErr
	})

	if retryErr != nil {
		return &emptypb.Empty{}, hferrors.GrpcError(
			codes.Internal,
			"error attempting to update",
			acRequest,
		)
	}

	return &emptypb.Empty{}, nil
}

func (a *GrpcAccessCodeServer) DeleteAc(ctx context.Context, req *generalpb.ResourceId) (*emptypb.Empty, error) {
	return util.DeleteHfResource(ctx, req, a.acClient, "access code")
}

func (a *GrpcAccessCodeServer) DeleteCollectionAc(ctx context.Context, listOptions *generalpb.ListOptions) (*emptypb.Empty, error) {
	return util.DeleteHfCollection(ctx, listOptions, a.acClient, "access codes")
}

func (a *GrpcAccessCodeServer) ListAc(ctx context.Context, listOptions *generalpb.ListOptions) (*accesscodepb.ListAcsResponse, error) {
	doLoadFromCache := listOptions.GetLoadFromCache()
	var accessCodes []hfv1.AccessCode
	var err error
	if !doLoadFromCache {
		var acList *hfv1.AccessCodeList
		acList, err = util.ListByHfClient(ctx, listOptions, a.acClient, "access codes")
		if err == nil {
			accessCodes = acList.Items
		}
	} else {
		accessCodes, err = util.ListByCache(listOptions, a.acLister, "access codes", a.acSynced())
	}
	if err != nil {
		glog.Error(err)
		return &accesscodepb.ListAcsResponse{}, err
	}

	preparedAcs := []*accesscodepb.AccessCode{}

	for _, accessCode := range accessCodes {

		if accessCode.Spec.Expiration != "" {
			expiration, err := time.Parse(time.UnixDate, accessCode.Spec.Expiration)

			if err != nil {
				return &accesscodepb.ListAcsResponse{}, hferrors.GrpcError(
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

		preparedAcs = append(preparedAcs, &accesscodepb.AccessCode{
			Id:                  accessCode.Name,
			Uid:                 string(accessCode.UID),
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

	return &accesscodepb.ListAcsResponse{AccessCodes: preparedAcs}, nil
}

/**************************************************************************************************************
 * Resource oriented RPCs for OneTimeAccessCodes
 *
 * The following functions implement the resource oriented RPCs for OneTimeAccessCodes
 **************************************************************************************************************/

func (a *GrpcAccessCodeServer) CreateOtac(ctx context.Context, cr *accesscodepb.CreateOtacRequest) (*accesscodepb.OneTimeAccessCode, error) {
	// Generate an access code that can not be guessed
	genName := ""
	for genParts := 0; genParts < 3; genParts++ {
		genName += util.GenerateResourceName("", util.RandStringRunes(16), 4)
	}
	genName = genName[1:]

	scheduledEventName := cr.GetSeName()
	if scheduledEventName == "" {
		return &accesscodepb.OneTimeAccessCode{}, hferrors.GrpcError(
			codes.InvalidArgument,
			"error creating otac, se_name field blank",
			cr,
		)
	}

	scheduledUid := cr.GetSeUid()
	if scheduledUid == "" {
		return &accesscodepb.OneTimeAccessCode{}, hferrors.GrpcError(
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
				hflabels.UserLabel:              "",
				hflabels.ScheduledEventLabel:    scheduledEventName,
				hflabels.OneTimeAccessCodeLabel: genName,
			},
		},
		Spec: hfv1.OneTimeAccessCodeSpec{
			User:              "",
			RedeemedTimestamp: "",
		},
	}
	otac, err := a.otacClient.Create(ctx, otac, metav1.CreateOptions{})
	if err != nil {
		glog.Errorf("error creating one time access code %v", err)
		// error handling
	}
	return &accesscodepb.OneTimeAccessCode{
		Id:                otac.Name,
		User:              otac.Spec.User,
		RedeemedTimestamp: otac.Spec.RedeemedTimestamp,
		MaxDuration:       otac.Spec.MaxDuration,
		Labels:            otac.Labels,
	}, nil
}

func (a *GrpcAccessCodeServer) GetOtac(ctx context.Context, req *generalpb.GetRequest) (*accesscodepb.OneTimeAccessCode, error) {
	otac, err := util.GenericHfGetter(ctx, req, a.otacClient, a.otacLister.OneTimeAccessCodes(util.GetReleaseNamespace()), "OTAC", a.otacSynced())
	if err != nil {
		return &accesscodepb.OneTimeAccessCode{}, err
	}

	glog.V(2).Infof("retrieved OTAC %s", otac.Name)

	return &accesscodepb.OneTimeAccessCode{
		Id:                otac.Name,
		Uid:               string(otac.UID),
		User:              otac.Spec.User,
		RedeemedTimestamp: otac.Spec.RedeemedTimestamp,
		MaxDuration:       otac.Spec.MaxDuration,
		Labels:            otac.Labels,
	}, nil
}

func (a *GrpcAccessCodeServer) UpdateOtac(ctx context.Context, otacRequest *accesscodepb.OneTimeAccessCode) (*emptypb.Empty, error) {
	id := otacRequest.GetId()
	if id == "" {
		return &emptypb.Empty{}, hferrors.GrpcError(
			codes.InvalidArgument,
			"no ID passed in",
			otacRequest,
		)
	}

	retryErr := retry.RetryOnConflict(retry.DefaultRetry, func() error {
		otac, err := a.otacClient.Get(ctx, id, metav1.GetOptions{})
		if err != nil {
			glog.Error(err)
			return hferrors.GrpcError(
				codes.Internal,
				"error while retrieving OTAC %s",
				otacRequest,
				otacRequest.GetId(),
			)
		}

		otac.Spec.User = otacRequest.GetUser()
		otac.Spec.RedeemedTimestamp = otacRequest.GetRedeemedTimestamp()
		otac.Spec.MaxDuration = otacRequest.GetMaxDuration()
		otac.Labels[hflabels.UserLabel] = otacRequest.GetUser()

		_, updateErr := a.otacClient.Update(ctx, otac, metav1.UpdateOptions{})
		return updateErr
	})

	if retryErr != nil {
		return &emptypb.Empty{}, hferrors.GrpcError(
			codes.Internal,
			"error attempting to update",
			otacRequest,
		)
	}

	return &emptypb.Empty{}, nil
}

func (a *GrpcAccessCodeServer) DeleteOtac(ctx context.Context, req *generalpb.ResourceId) (*emptypb.Empty, error) {
	return util.DeleteHfResource(ctx, req, a.otacClient, "OTAC")
}

func (a *GrpcAccessCodeServer) DeleteCollectionOtac(ctx context.Context, listOptions *generalpb.ListOptions) (*emptypb.Empty, error) {
	return util.DeleteHfCollection(ctx, listOptions, a.otacClient, "OTACs")
}

func (a *GrpcAccessCodeServer) ListOtac(ctx context.Context, listOptions *generalpb.ListOptions) (*accesscodepb.ListOtacsResponse, error) {
	doLoadFromCache := listOptions.GetLoadFromCache()
	var otacs []hfv1.OneTimeAccessCode
	var err error
	if !doLoadFromCache {
		var otacList *hfv1.OneTimeAccessCodeList
		otacList, err = util.ListByHfClient(ctx, listOptions, a.otacClient, "OTACs")
		if err == nil {
			otacs = otacList.Items
		}
	} else {
		otacs, err = util.ListByCache(listOptions, a.otacLister, "OTACs", a.otacSynced())
	}
	if err != nil {
		glog.Error(err)
		return &accesscodepb.ListOtacsResponse{}, err
	}

	preparedOtacs := []*accesscodepb.OneTimeAccessCode{} // must be declared this way so as to JSON marshal into [] instead of null
	for _, otac := range otacs {
		preparedOtacs = append(preparedOtacs, &accesscodepb.OneTimeAccessCode{
			Id:                otac.Name,
			Uid:               string(otac.UID),
			User:              otac.Spec.User,
			RedeemedTimestamp: otac.Spec.RedeemedTimestamp,
			MaxDuration:       otac.Spec.MaxDuration,
			Labels:            otac.Labels,
		})
	}

	glog.V(2).Infof("listed otacs")

	return &accesscodepb.ListOtacsResponse{Otacs: preparedOtacs}, nil
}

/**************************************************************************************************************
 * Helper RPCs
 *
 * This section includes Helper RPCs exposed by the internal gRPC server.
 * These RPCs provide advanced functionalities beyond basic resource-related operations.
 **************************************************************************************************************/

func (a *GrpcAccessCodeServer) ValidateExistence(ctx context.Context, gor *generalpb.ResourceId) (*accesscodepb.ResourceValidation, error) {
	if len(gor.GetId()) == 0 {
		return &accesscodepb.ResourceValidation{Valid: false}, hferrors.GrpcIdNotSpecifiedError(gor)
	}

	_, err := a.acClient.Get(ctx, gor.GetId(), metav1.GetOptions{})
	if err != nil {
		// If AccessCode does not exist check if this might be an OTAC
		_, err := a.otacClient.Get(ctx, gor.GetId(), metav1.GetOptions{})
		if err != nil {
			return &accesscodepb.ResourceValidation{Valid: false}, nil
		}
	}

	return &accesscodepb.ResourceValidation{Valid: true}, nil
}

func (a *GrpcAccessCodeServer) GetAccessCodesWithOTACs(ctx context.Context, codeIds *accesscodepb.ResourceIds) (*accesscodepb.ListAcsResponse, error) {
	ids := codeIds.GetIds()
	otacReq, err := labels.NewRequirement(hflabels.OneTimeAccessCodeLabel, selection.In, ids)
	if err != nil {
		return &accesscodepb.ListAcsResponse{}, hferrors.GrpcError(
			codes.Internal,
			"Unable to create label selector from access code ids",
			codeIds,
		)
	}
	selector := labels.NewSelector()
	selector = selector.Add(*otacReq)
	selectorString := selector.String()

	// First get the oneTimeAccessCodes
	otacList, err := a.ListOtac(ctx, &generalpb.ListOptions{LabelSelector: selectorString})

	if err != nil {
		return nil, err
	}

	//Append the value of onetime access codes to the list
	for _, otac := range otacList.Otacs {
		se, err := a.eventClient.GetScheduledEvent(ctx, &generalpb.GetRequest{Id: otac.Labels[hflabels.ScheduledEventLabel]})
		if err != nil {
			glog.Error(err)
			return &accesscodepb.ListAcsResponse{}, hferrors.GrpcError(
				codes.Internal,
				"error retreiving scheduled event from OTAC: %v",
				codeIds,
				err,
			)
		}
		ids = append(ids, se.AccessCode)
	}

	// Update the label selector
	acReq, err := labels.NewRequirement(hflabels.AccessCodeLabel, selection.In, ids)
	if err != nil {
		return &accesscodepb.ListAcsResponse{}, hferrors.GrpcError(
			codes.Internal,
			"Unable to create label selector from access code ids",
			codeIds,
		)
	}
	selector = labels.NewSelector()
	selector = selector.Add(*acReq)
	selectorString = selector.String()

	accessCodes, err := a.ListAc(ctx, &generalpb.ListOptions{LabelSelector: selectorString})
	return accessCodes, err
}

func (a *GrpcAccessCodeServer) GetAccessCodeWithOTACs(ctx context.Context, codeId *generalpb.ResourceId) (*accesscodepb.AccessCode, error) {
	accessCodeId := codeId.GetId()
	if len(accessCodeId) == 0 {
		return &accesscodepb.AccessCode{}, hferrors.GrpcIdNotSpecifiedError(codeId)
	}

	accessCodeList, err := a.GetAccessCodesWithOTACs(ctx, &accesscodepb.ResourceIds{Ids: []string{accessCodeId}})

	if err != nil {
		return &accesscodepb.AccessCode{}, hferrors.GrpcError(
			codes.NotFound,
			"access code (%s) not found: %v",
			codeId,
			accessCodeId,
			err,
		)
	}

	accessCodes := accessCodeList.GetAccessCodes()

	if len(accessCodes) != 1 {
		return &accesscodepb.AccessCode{}, hferrors.GrpcError(
			codes.Internal,
			"insane result found",
			codeId,
		)
	}

	return accessCodes[0], nil
}

func (a *GrpcAccessCodeServer) GetAcOwnerReferences(ctx context.Context, req *generalpb.GetRequest) (*generalpb.OwnerReferences, error) {
	return util.GetOwnerReferences(ctx, req, a.acClient, a.acLister.AccessCodes(util.GetReleaseNamespace()), "access code", a.acSynced())
}

/**************************************************************************************************************
 * Internal helper functions
 *
 * Internal helper functions which are only used within this file
 **************************************************************************************************************/

func (a *GrpcAccessCodeServer) checkInputParamsForCreateAc(cr *accesscodepb.CreateAcRequest) error {
	if cr.GetAcName() == "" ||
		cr.GetDescription() == "" ||
		cr.GetExpiration() == "" ||
		cr.GetSeName() == "" ||
		cr.GetSeUid() == "" ||
		(cr.GetRestrictedBind() && cr.GetRestrictedBindValue() == "") {

		return hferrors.GrpcError(
			codes.InvalidArgument,
			"error creating access code, required input field is blank",
			cr,
		)
	}
	return nil
}
