package progressservice

import (
	"context"
	"time"

	"github.com/hobbyfarm/gargantua/v3/protos/general"
	progressProto "github.com/hobbyfarm/gargantua/v3/protos/progress"

	"github.com/golang/glog"
	"github.com/golang/protobuf/ptypes/empty"
	hfv1 "github.com/hobbyfarm/gargantua/v3/pkg/apis/hobbyfarm.io/v1"
	hfClientset "github.com/hobbyfarm/gargantua/v3/pkg/client/clientset/versioned"
	hferrors "github.com/hobbyfarm/gargantua/v3/pkg/errors"
	"github.com/hobbyfarm/gargantua/v3/pkg/util"
	"google.golang.org/grpc/codes"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/util/retry"
)

type GrpcProgressServer struct {
	progressProto.UnimplementedProgressSvcServer
	hfClientSet hfClientset.Interface
}

func NewGrpcProgressServer(hfClientSet hfClientset.Interface) *GrpcProgressServer {
	return &GrpcProgressServer{
		hfClientSet: hfClientSet,
	}
}

func (s *GrpcProgressServer) CreateProgress(ctx context.Context, req *progressProto.CreateProgressRequest) (*empty.Empty, error) {
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

	_, err := s.hfClientSet.HobbyfarmV1().Progresses(util.GetReleaseNamespace()).Create(ctx, progress, v1.CreateOptions{})
	if err != nil {
		return &empty.Empty{}, hferrors.GrpcError(
			codes.Internal,
			err.Error(),
			req,
		)
	}
	return &empty.Empty{}, nil
}

func (s *GrpcProgressServer) GetProgress(ctx context.Context, id *general.ResourceId) (*progressProto.Progress, error) {
	if len(id.GetId()) == 0 {
		return &progressProto.Progress{}, hferrors.GrpcError(
			codes.InvalidArgument,
			"no id passed in",
			id,
		)
	}
	progress, err := s.hfClientSet.HobbyfarmV1().Progresses(util.GetReleaseNamespace()).Get(ctx, id.GetId(), v1.GetOptions{})
	if errors.IsNotFound(err) {
		return &progressProto.Progress{}, hferrors.GrpcNotFoundError(id, "progress")
	} else if err != nil {
		glog.V(2).Infof("error while retrieving progress: %v", err)
		return &progressProto.Progress{}, hferrors.GrpcError(
			codes.Internal,
			"error while retrieving progress by id: %s with error: %v",
			id,
			id.GetId(),
			err,
		)
	}

	progressSteps := []*progressProto.ProgressStep{}

	for _, step := range progress.Spec.Steps {
		progressStep := &progressProto.ProgressStep{
			Step:      uint32(step.Step),
			Timestamp: step.Timestamp,
		}
		progressSteps = append(progressSteps, progressStep)
	}

	return &progressProto.Progress{
		Id:          progress.Name,
		CurrentStep: uint32(progress.Spec.CurrentStep),
		MaxStep:     uint32(progress.Spec.MaxStep),
		TotalStep:   uint32(progress.Spec.TotalStep),
		Scenario:    progress.Spec.Scenario,
		Course:      progress.Spec.Course,
		User:        progress.Spec.UserId,
		Started:     progress.Spec.Started,
		LastUpdate:  progress.Spec.LastUpdate,
		Finished:    progress.Spec.Finished,
		Steps:       progressSteps,
		Labels:      progress.Labels,
	}, nil
}

func (s *GrpcProgressServer) UpdateProgress(ctx context.Context, req *progressProto.UpdateProgressRequest) (*empty.Empty, error) {
	id := req.GetId()
	if len(id) == 0 {
		return &empty.Empty{}, hferrors.GrpcError(
			codes.InvalidArgument,
			"no id passed in",
			req,
		)
	}

	currentStep := req.GetCurrentStep()
	maxStep := req.GetMaxStep()
	totalStep := req.GetTotalStep()
	lastUpdate := req.GetLastUpdate()
	finished := req.GetFinished()
	steps := req.GetSteps()

	retryErr := retry.RetryOnConflict(retry.DefaultRetry, func() error {
		progress, err := s.hfClientSet.HobbyfarmV1().Progresses(util.GetReleaseNamespace()).Get(ctx, id, v1.GetOptions{})
		if err != nil {
			glog.Error(err)
			return hferrors.GrpcError(
				codes.Internal,
				"error while retrieving progress %s",
				req,
				req.GetId(),
			)
		}

		if currentStep != nil {
			progress.Spec.CurrentStep = int(currentStep.Value)
		}

		if maxStep != nil {
			progress.Spec.MaxStep = int(maxStep.Value)
		}

		if totalStep != nil {
			progress.Spec.TotalStep = int(totalStep.Value)
		}

		if lastUpdate != "" {
			progress.Spec.LastUpdate = lastUpdate
		}

		if finished != "" {
			progress.Spec.Finished = finished
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

		_, updateErr := s.hfClientSet.HobbyfarmV1().Progresses(util.GetReleaseNamespace()).Update(ctx, progress, v1.UpdateOptions{})
		return updateErr
	})

	if retryErr != nil {
		return &empty.Empty{}, hferrors.GrpcError(
			codes.Internal,
			"error attempting to update",
			req,
		)
	}

	return &empty.Empty{}, nil
}

func (s *GrpcProgressServer) DeleteProgress(ctx context.Context, req *general.ResourceId) (*empty.Empty, error) {
	id := req.GetId()
	if len(id) == 0 {
		return &empty.Empty{}, hferrors.GrpcError(
			codes.InvalidArgument,
			"no ID passed in",
			req,
		)
	}

	err := s.hfClientSet.HobbyfarmV1().Progresses(util.GetReleaseNamespace()).Delete(ctx, id, v1.DeleteOptions{})

	if err != nil {
		glog.Errorf("error deleting progress %s: %v", id, err)
		return &empty.Empty{}, hferrors.GrpcError(
			codes.Internal,
			"error deleting progress %s",
			req,
			id,
		)
	}

	return &empty.Empty{}, nil
}

func (s *GrpcProgressServer) DeleteCollectionProgress(ctx context.Context, listOptions *general.ListOptions) (*empty.Empty, error) {
	err := s.hfClientSet.HobbyfarmV1().Progresses(util.GetReleaseNamespace()).DeleteCollection(ctx, v1.DeleteOptions{}, v1.ListOptions{
		LabelSelector: listOptions.GetLabelSelector(),
	})
	if err != nil {
		return &empty.Empty{}, hferrors.GrpcError(
			codes.Internal,
			"error deleting progresses",
			listOptions,
		)
	}
	return &empty.Empty{}, nil
}

func (s *GrpcProgressServer) ListProgress(ctx context.Context, listOptions *general.ListOptions) (*progressProto.ListProgressesResponse, error) {
	progressList, err := s.hfClientSet.HobbyfarmV1().Progresses(util.GetReleaseNamespace()).List(ctx, v1.ListOptions{
		LabelSelector: listOptions.GetLabelSelector(),
	})
	if err != nil {
		glog.Error(err)
		return &progressProto.ListProgressesResponse{}, hferrors.GrpcError(
			codes.Internal,
			"error retreiving progresses",
			listOptions,
		)
	}

	preparedProgress := []*progressProto.Progress{}

	for _, progress := range progressList.Items {
		progressSteps := []*progressProto.ProgressStep{}
		for _, step := range progress.Spec.Steps {
			progressStep := &progressProto.ProgressStep{
				Step:      uint32(step.Step),
				Timestamp: step.Timestamp,
			}
			progressSteps = append(progressSteps, progressStep)
		}

		preparedProgress = append(preparedProgress, &progressProto.Progress{
			Id:          progress.Name,
			CurrentStep: uint32(progress.Spec.CurrentStep),
			MaxStep:     uint32(progress.Spec.MaxStep),
			TotalStep:   uint32(progress.Spec.TotalStep),
			Scenario:    progress.Spec.Scenario,
			Course:      progress.Spec.Course,
			User:        progress.Spec.UserId,
			Started:     progress.Spec.Started,
			LastUpdate:  progress.Spec.LastUpdate,
			Finished:    progress.Spec.Finished,
			Steps:       progressSteps,
			Labels:      progress.Labels,
		})
	}

	return &progressProto.ListProgressesResponse{Progresses: preparedProgress}, nil
}
