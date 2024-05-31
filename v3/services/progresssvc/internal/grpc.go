package progressservice

import (
	"context"
	"time"

	generalpb "github.com/hobbyfarm/gargantua/v3/protos/general"
	progresspb "github.com/hobbyfarm/gargantua/v3/protos/progress"

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
	"google.golang.org/protobuf/types/known/timestamppb"
	"google.golang.org/protobuf/types/known/wrapperspb"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/retry"
)

type GrpcProgressServer struct {
	progresspb.UnimplementedProgressSvcServer
	progressClient hfClientsetv1.ProgressInterface
	progressLister listersv1.ProgressLister
	progressSynced cache.InformerSynced
}

func NewGrpcProgressServer(hfClientSet hfClientset.Interface, hfInformerFactory hfInformers.SharedInformerFactory) *GrpcProgressServer {
	return &GrpcProgressServer{
		progressClient: hfClientSet.HobbyfarmV1().Progresses(util.GetReleaseNamespace()),
		progressLister: hfInformerFactory.Hobbyfarm().V1().Progresses().Lister(),
		progressSynced: hfInformerFactory.Hobbyfarm().V1().Progresses().Informer().HasSynced,
	}
}

func (s *GrpcProgressServer) CreateProgress(ctx context.Context, req *progresspb.CreateProgressRequest) (*generalpb.ResourceId, error) {
	random := util.RandStringRunes(16)
	now := time.Now()
	progressId := util.GenerateResourceName("progress", random, 16)

	currentStep := req.GetCurrentStep()
	maxStep := req.GetMaxStep()
	totalStep := req.GetTotalStep()
	scenario := req.GetScenario()
	course := req.GetCourse()
	user := req.GetUser()
	labels := req.GetLabels()

	progress := &hfv1.Progress{
		ObjectMeta: metav1.ObjectMeta{
			Name:   progressId,
			Labels: labels,
		},
		Spec: hfv1.ProgressSpec{
			CurrentStep: int(currentStep),
			MaxStep:     int(maxStep),
			TotalStep:   int(totalStep),
			Course:      course,
			Scenario:    scenario,
			UserId:      user,
			Started:     now.Format(time.UnixDate),
			LastUpdate:  now.Format(time.UnixDate),
			Finished:    "false",
		},
	}

	steps := []hfv1.ProgressStep{}
	step := hfv1.ProgressStep{Step: 0, Timestamp: now.Format(time.UnixDate)}
	steps = append(steps, step)
	progress.Spec.Steps = steps

	_, err := s.progressClient.Create(ctx, progress, metav1.CreateOptions{})
	if err != nil {
		return &generalpb.ResourceId{}, hferrors.GrpcError(
			codes.Internal,
			err.Error(),
			req,
		)
	}
	return &generalpb.ResourceId{Id: progressId}, nil
}

func (s *GrpcProgressServer) GetProgress(ctx context.Context, req *generalpb.GetRequest) (*progresspb.Progress, error) {
	progress, err := util.GenericHfGetter(ctx, req, s.progressClient, s.progressLister.Progresses(util.GetReleaseNamespace()), "progress", s.progressSynced())
	if err != nil {
		return &progresspb.Progress{}, err
	}

	progressSteps := []*progresspb.ProgressStep{}

	for _, step := range progress.Spec.Steps {
		progressStep := &progresspb.ProgressStep{
			Step:      uint32(step.Step),
			Timestamp: step.Timestamp,
		}
		progressSteps = append(progressSteps, progressStep)
	}

	var creationTimeStamp *timestamppb.Timestamp
	if !progress.CreationTimestamp.IsZero() {
		creationTimeStamp = timestamppb.New(progress.CreationTimestamp.Time)
	}

	return &progresspb.Progress{
		Id:                progress.Name,
		Uid:               string(progress.UID),
		CurrentStep:       uint32(progress.Spec.CurrentStep),
		MaxStep:           uint32(progress.Spec.MaxStep),
		TotalStep:         uint32(progress.Spec.TotalStep),
		Scenario:          progress.Spec.Scenario,
		Course:            progress.Spec.Course,
		User:              progress.Spec.UserId,
		Started:           progress.Spec.Started,
		LastUpdate:        progress.Spec.LastUpdate,
		Finished:          progress.Spec.Finished,
		Steps:             progressSteps,
		Labels:            progress.Labels,
		CreationTimestamp: creationTimeStamp,
	}, nil
}

func (s *GrpcProgressServer) UpdateProgress(ctx context.Context, req *progresspb.UpdateProgressRequest) (*emptypb.Empty, error) {
	id := req.GetId()
	if len(id) == 0 {
		return &emptypb.Empty{}, hferrors.GrpcIdNotSpecifiedError(req)
	}

	currentStep := req.GetCurrentStep()
	maxStep := req.GetMaxStep()
	totalStep := req.GetTotalStep()
	lastUpdate := req.GetLastUpdate()
	finished := req.GetFinished()
	steps := req.GetSteps()

	retryErr := retry.RetryOnConflict(retry.DefaultRetry, func() error {
		progress, err := s.progressClient.Get(ctx, id, metav1.GetOptions{})
		if err != nil {
			glog.Error(err)
			return hferrors.GrpcError(
				codes.Internal,
				"error while retrieving progress %s",
				req,
				req.GetId(),
			)
		}
		return s.updateProgress(ctx, progress, currentStep, maxStep, totalStep, lastUpdate, finished, steps)
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

func (s *GrpcProgressServer) UpdateCollectionProgress(ctx context.Context, req *progresspb.UpdateCollectionProgressRequest) (*emptypb.Empty, error) {
	progressList, err := util.ListByHfClient(ctx, &generalpb.ListOptions{LabelSelector: req.GetLabelselector()}, s.progressClient, "progresses")
	if err != nil {
		return &emptypb.Empty{}, hferrors.GrpcError(
			codes.Internal,
			"error while listing progress",
			req,
		)
	}
	progresses := progressList.Items
	// If a client tries to update on an empty list, we throw a notFound error... this is an invalid operation.
	if len(progresses) == 0 {
		return &emptypb.Empty{}, hferrors.GrpcError(
			codes.NotFound,
			"error no progress found",
			req,
		)
	}

	currentStep := req.GetCurrentStep()
	maxStep := req.GetMaxStep()
	totalStep := req.GetTotalStep()
	lastUpdate := req.GetLastUpdate()
	finished := req.GetFinished()
	steps := req.GetSteps()

	var retryErr error
	for _, progress := range progresses {
		retryErr = retry.RetryOnConflict(retry.DefaultRetry, func() error {
			return s.updateProgress(ctx, &progress, currentStep, maxStep, totalStep, lastUpdate, finished, steps)
		})
	}

	if retryErr != nil {
		return &emptypb.Empty{}, hferrors.GrpcError(
			codes.Internal,
			"error attempting to update",
			req,
		)
	}

	return &emptypb.Empty{}, nil
}

func (s *GrpcProgressServer) DeleteProgress(ctx context.Context, req *generalpb.ResourceId) (*emptypb.Empty, error) {
	return util.DeleteHfResource(ctx, req, s.progressClient, "progress")
}

func (s *GrpcProgressServer) DeleteCollectionProgress(ctx context.Context, listOptions *generalpb.ListOptions) (*emptypb.Empty, error) {
	return util.DeleteHfCollection(ctx, listOptions, s.progressClient, "progresses")
}

func (s *GrpcProgressServer) ListProgress(ctx context.Context, listOptions *generalpb.ListOptions) (*progresspb.ListProgressesResponse, error) {
	doLoadFromCache := listOptions.GetLoadFromCache()
	var progresses []hfv1.Progress
	var err error
	if !doLoadFromCache {
		var progressList *hfv1.ProgressList
		progressList, err = util.ListByHfClient(ctx, listOptions, s.progressClient, "progresses")
		if err == nil {
			progresses = progressList.Items
		}
	} else {
		progresses, err = util.ListByCache(listOptions, s.progressLister, "progresses", s.progressSynced())
	}
	if err != nil {
		glog.Error(err)
		return &progresspb.ListProgressesResponse{}, err
	}

	preparedProgress := []*progresspb.Progress{}

	for _, progress := range progresses {
		progressSteps := []*progresspb.ProgressStep{}
		for _, step := range progress.Spec.Steps {
			progressStep := &progresspb.ProgressStep{
				Step:      uint32(step.Step),
				Timestamp: step.Timestamp,
			}
			progressSteps = append(progressSteps, progressStep)
		}

		var creationTimeStamp *timestamppb.Timestamp
		if !progress.CreationTimestamp.IsZero() {
			creationTimeStamp = timestamppb.New(progress.CreationTimestamp.Time)
		}

		preparedProgress = append(preparedProgress, &progresspb.Progress{
			Id:                progress.Name,
			Uid:               string(progress.UID),
			CurrentStep:       uint32(progress.Spec.CurrentStep),
			MaxStep:           uint32(progress.Spec.MaxStep),
			TotalStep:         uint32(progress.Spec.TotalStep),
			Scenario:          progress.Spec.Scenario,
			Course:            progress.Spec.Course,
			User:              progress.Spec.UserId,
			Started:           progress.Spec.Started,
			LastUpdate:        progress.Spec.LastUpdate,
			Finished:          progress.Spec.Finished,
			Steps:             progressSteps,
			Labels:            progress.Labels,
			CreationTimestamp: creationTimeStamp,
		})
	}

	return &progresspb.ListProgressesResponse{Progresses: preparedProgress}, nil
}

func (s *GrpcProgressServer) updateProgress(
	ctx context.Context,
	progress *hfv1.Progress,
	currStep *wrapperspb.UInt32Value,
	maxStep *wrapperspb.UInt32Value,
	totalStep *wrapperspb.UInt32Value,
	lastUpdate string,
	finished string,
	steps []*progresspb.ProgressStep,
) error {
	if currStep != nil {
		progress.Spec.CurrentStep = int(currStep.Value)
	}

	if maxStep != nil {
		progress.Spec.MaxStep = int(maxStep.Value)
	}

	// ensure the max visited step is always >= the currently visited step after an update
	if progress.Spec.CurrentStep > progress.Spec.MaxStep {
		progress.Spec.MaxStep = progress.Spec.CurrentStep
	}

	if totalStep != nil {
		progress.Spec.TotalStep = int(totalStep.Value)
	}

	if lastUpdate != "" {
		progress.Spec.LastUpdate = lastUpdate
	}

	if finished != "" {
		progress.Spec.Finished = finished
		progress.Labels["finished"] = finished
	}

	if len(steps) > 0 {
		progressSteps := []hfv1.ProgressStep{}
		for _, step := range steps {
			progressStep := hfv1.ProgressStep{
				Step:      int(step.GetStep()),
				Timestamp: step.GetTimestamp(),
			}
			progressSteps = append(progressSteps, progressStep)
		}
		progress.Spec.Steps = progressSteps
	}

	_, updateErr := s.progressClient.Update(ctx, progress, metav1.UpdateOptions{})
	return updateErr
}
