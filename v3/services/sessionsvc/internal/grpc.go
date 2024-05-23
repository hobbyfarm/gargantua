package sessionservice

import (
	"context"

	generalpb "github.com/hobbyfarm/gargantua/v3/protos/general"
	sessionpb "github.com/hobbyfarm/gargantua/v3/protos/session"

	"github.com/golang/glog"
	hfv1 "github.com/hobbyfarm/gargantua/v3/pkg/apis/hobbyfarm.io/v1"
	hfClientset "github.com/hobbyfarm/gargantua/v3/pkg/client/clientset/versioned"
	hfClientsetv1 "github.com/hobbyfarm/gargantua/v3/pkg/client/clientset/versioned/typed/hobbyfarm.io/v1"
	hfInformers "github.com/hobbyfarm/gargantua/v3/pkg/client/informers/externalversions"
	listersv1 "github.com/hobbyfarm/gargantua/v3/pkg/client/listers/hobbyfarm.io/v1"
	hferrors "github.com/hobbyfarm/gargantua/v3/pkg/errors"
	"github.com/hobbyfarm/gargantua/v3/pkg/util"
	"google.golang.org/grpc/codes"
	"google.golang.org/protobuf/types/known/emptypb"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/retry"
)

type GrpcSessionServer struct {
	sessionpb.UnimplementedSessionSvcServer
	sessionClient hfClientsetv1.SessionInterface
	sessionLister listersv1.SessionLister
	sessionSynced cache.InformerSynced
}

func NewGrpcSessionServer(hfClientSet hfClientset.Interface, hfInformerFactory hfInformers.SharedInformerFactory) *GrpcSessionServer {
	return &GrpcSessionServer{
		sessionClient: hfClientSet.HobbyfarmV1().Sessions(util.GetReleaseNamespace()),
		sessionLister: hfInformerFactory.Hobbyfarm().V1().Sessions().Lister(),
		sessionSynced: hfInformerFactory.Hobbyfarm().V1().Sessions().Informer().HasSynced,
	}
}

func (s *GrpcSessionServer) CreateSession(ctx context.Context, req *sessionpb.CreateSessionRequest) (*emptypb.Empty, error) {
	scenario := req.GetScenario()
	course := req.GetCourse()
	keepCourseVm := req.GetKeepCourseVm()
	userId := req.GetUser()
	vmClaims := req.GetVmClaim()
	accessCode := req.GetAccessCode()
	labels := req.GetLabels()

	if scenario == "" && course == "" {
		return &emptypb.Empty{}, hferrors.GrpcError(codes.InvalidArgument, "no course/scenario id provided", req)
	}

	requiredStringParams := map[string]string{
		"user":       userId,
		"accessCode": accessCode,
	}
	for param, value := range requiredStringParams {
		if value == "" {
			return &emptypb.Empty{}, hferrors.GrpcNotSpecifiedError(req, param)
		}
	}

	random := util.RandStringRunes(10)
	id := util.GenerateResourceName("ss", random, 10)

	session := &hfv1.Session{
		ObjectMeta: metav1.ObjectMeta{
			Name:   id,
			Labels: labels,
		},
		Spec: hfv1.SessionSpec{
			ScenarioId:   scenario,
			CourseId:     course,
			KeepCourseVM: keepCourseVm,
			UserId:       userId,
			VmClaimSet:   vmClaims,
			AccessCode:   accessCode,
		},
	}

	_, err := s.sessionClient.Create(ctx, session, metav1.CreateOptions{})
	if err != nil {
		return &emptypb.Empty{}, hferrors.GrpcError(
			codes.Internal,
			err.Error(),
			req,
		)
	}
	return &emptypb.Empty{}, nil
}

func (s *GrpcSessionServer) GetSession(ctx context.Context, req *generalpb.GetRequest) (*sessionpb.Session, error) {
	session, err := util.GenericHfGetter(ctx, req, s.sessionClient, s.sessionLister.Sessions(util.GetReleaseNamespace()), "session", s.sessionSynced())
	if err != nil {
		return &sessionpb.Session{}, err
	}

	status := &sessionpb.SessionStatus{
		Paused:         session.Status.Paused,
		PausedTime:     session.Status.PausedTime,
		Active:         session.Status.Active,
		Finished:       session.Status.Finished,
		StartTime:      session.Status.StartTime,
		ExpirationTime: session.Status.ExpirationTime,
	}

	return &sessionpb.Session{
		Id:           session.Name,
		Uid:          string(session.UID),
		Scenario:     session.Spec.ScenarioId,
		Course:       session.Spec.CourseId,
		KeepCourseVm: session.Spec.KeepCourseVM,
		User:         session.Spec.UserId,
		VmClaim:      session.Spec.VmClaimSet,
		AccessCode:   session.Spec.AccessCode,
		Labels:       session.Labels,
		Status:       status,
	}, nil
}

func (s *GrpcSessionServer) UpdateSession(ctx context.Context, req *sessionpb.UpdateSessionRequest) (*emptypb.Empty, error) {
	id := req.GetId()
	if len(id) == 0 {
		return &emptypb.Empty{}, hferrors.GrpcIdNotSpecifiedError(req)
	}
	scenario := req.GetScenario()
	if scenario == "" {
		return &emptypb.Empty{}, hferrors.GrpcNotSpecifiedError(req, "scenario")
	}

	retryErr := retry.RetryOnConflict(retry.DefaultRetry, func() error {
		session, err := s.sessionClient.Get(ctx, id, metav1.GetOptions{})
		if err != nil {
			glog.Error(err)
			return hferrors.GrpcError(
				codes.Internal,
				"error while retrieving session %s",
				req,
				req.GetId(),
			)
		}

		session.Spec.ScenarioId = scenario

		_, updateErr := s.sessionClient.Update(ctx, session, metav1.UpdateOptions{})
		return updateErr
	})

	if retryErr != nil {
		return &emptypb.Empty{}, hferrors.GrpcError(
			codes.Internal,
			"error attempting to update",
			req,
		)
	}

	return &emptypb.Empty{}, nil
}

func (s *GrpcSessionServer) UpdateSessionStatus(ctx context.Context, req *sessionpb.UpdateSessionStatusRequest) (*emptypb.Empty, error) {
	id := req.GetId()
	if len(id) == 0 {
		return &emptypb.Empty{}, hferrors.GrpcIdNotSpecifiedError(req)
	}

	paused := req.GetPaused()
	pausedTime := req.GetPausedTime()
	active := req.GetActive()
	finished := req.GetFinished()
	startTime := req.GetStartTime()
	expirationTime := req.GetExpirationTime()

	retryErr := retry.RetryOnConflict(retry.DefaultRetry, func() error {
		session, err := s.sessionClient.Get(ctx, id, metav1.GetOptions{})
		if err != nil {
			glog.Error(err)
			return hferrors.GrpcError(
				codes.Internal,
				"error while retrieving session %s",
				req,
				req.GetId(),
			)
		}

		if paused != nil {
			session.Status.Paused = paused.GetValue()
		}

		if pausedTime != nil {
			session.Status.PausedTime = pausedTime.GetValue()
		}

		if active != nil {
			session.Status.Active = active.GetValue()
		}

		if finished != nil {
			session.Status.Finished = finished.GetValue()
		}

		if startTime != "" {
			session.Status.StartTime = startTime
		}

		if expirationTime != "" {
			session.Status.ExpirationTime = expirationTime
		}

		_, updateErr := s.sessionClient.UpdateStatus(ctx, session, metav1.UpdateOptions{})
		if updateErr != nil {
			return updateErr
		}
		// @TODO: verify result like in util.go
		glog.V(4).Infof("updated result for session")
		return nil
	})
	if retryErr != nil {
		return &emptypb.Empty{}, hferrors.GrpcError(
			codes.Internal,
			"error attempting to update session status: %v",
			req,
			retryErr,
		)
	}
	return &emptypb.Empty{}, nil
}

func (s *GrpcSessionServer) DeleteSession(ctx context.Context, req *generalpb.ResourceId) (*emptypb.Empty, error) {
	return util.DeleteHfResource(ctx, req, s.sessionClient, "session")
}

func (s *GrpcSessionServer) DeleteCollectionSession(ctx context.Context, listOptions *generalpb.ListOptions) (*emptypb.Empty, error) {
	return util.DeleteHfCollection(ctx, listOptions, s.sessionClient, "session")
}

func (s *GrpcSessionServer) ListSession(ctx context.Context, listOptions *generalpb.ListOptions) (*sessionpb.ListSessionsResponse, error) {
	doLoadFromCache := listOptions.GetLoadFromCache()
	var sessions []hfv1.Session
	var err error
	if !doLoadFromCache {
		var sessionList *hfv1.SessionList
		sessionList, err = util.ListByHfClient(ctx, listOptions, s.sessionClient, "sessions")
		if err == nil {
			sessions = sessionList.Items
		}
	} else {
		sessions, err = util.ListByCache(listOptions, s.sessionLister, "sessions", s.sessionSynced())
	}
	if err != nil {
		glog.Error(err)
		return &sessionpb.ListSessionsResponse{}, err
	}

	preparedSessions := []*sessionpb.Session{}

	for _, session := range sessions {
		status := &sessionpb.SessionStatus{
			Paused:         session.Status.Paused,
			PausedTime:     session.Status.PausedTime,
			Active:         session.Status.Active,
			Finished:       session.Status.Finished,
			StartTime:      session.Status.StartTime,
			ExpirationTime: session.Status.ExpirationTime,
		}

		preparedSessions = append(preparedSessions, &sessionpb.Session{
			Id:           session.Name,
			Uid:          string(session.UID),
			Scenario:     session.Spec.ScenarioId,
			Course:       session.Spec.CourseId,
			KeepCourseVm: session.Spec.KeepCourseVM,
			User:         session.Spec.UserId,
			VmClaim:      session.Spec.VmClaimSet,
			AccessCode:   session.Spec.AccessCode,
			Labels:       session.Labels,
			Status:       status,
		})
	}

	return &sessionpb.ListSessionsResponse{Sessions: preparedSessions}, nil
}
