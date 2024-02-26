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
	hfClientsetv1 "github.com/hobbyfarm/gargantua/v3/pkg/client/clientset/versioned/typed/hobbyfarm.io/v1"
	hfInformers "github.com/hobbyfarm/gargantua/v3/pkg/client/informers/externalversions"
	listersv1 "github.com/hobbyfarm/gargantua/v3/pkg/client/listers/hobbyfarm.io/v1"
	hferrors "github.com/hobbyfarm/gargantua/v3/pkg/errors"
	"github.com/hobbyfarm/gargantua/v3/pkg/util"
	"google.golang.org/grpc/codes"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/retry"
)

type GrpcProgressServer struct {
	progressProto.UnimplementedProgressSvcServer
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

	_, err := s.progressClient.Create(ctx, progress, v1.CreateOptions{})
	if err != nil {
		return &empty.Empty{}, hferrors.GrpcError(
			codes.Internal,
			err.Error(),
			req,
		)
	}
	return &empty.Empty{}, nil
}

func (s *GrpcProgressServer) GetProgress(ctx context.Context, req *general.GetRequest) (*progressProto.Progress, error) {
	progress, err := util.GenericHfGetter(ctx, req, s.progressClient, s.progressLister.Progresses(util.GetReleaseNamespace()), "progress", s.progressSynced())
	if err != nil {
		return &progressProto.Progress{}, err
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
		return &empty.Empty{}, hferrors.GrpcIdNotSpecifiedError(req)
	}

	currentStep := req.GetCurrentStep()
	maxStep := req.GetMaxStep()
	totalStep := req.GetTotalStep()
	lastUpdate := req.GetLastUpdate()
	finished := req.GetFinished()
	steps := req.GetSteps()

	retryErr := retry.RetryOnConflict(retry.DefaultRetry, func() error {
		progress, err := s.progressClient.Get(ctx, id, v1.GetOptions{})
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

		_, updateErr := s.progressClient.Update(ctx, progress, v1.UpdateOptions{})
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

	err := s.progressClient.Delete(ctx, id, v1.DeleteOptions{})

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
	err := s.progressClient.DeleteCollection(ctx, v1.DeleteOptions{}, v1.ListOptions{
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
		return &progressProto.ListProgressesResponse{}, err
	}

	preparedProgress := []*progressProto.Progress{}

	for _, progress := range progresses {
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
