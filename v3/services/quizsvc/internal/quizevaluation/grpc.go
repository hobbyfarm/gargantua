package quizevaluation

import (
	"context"
	"github.com/golang/glog"
	hfv1 "github.com/hobbyfarm/gargantua/v3/pkg/apis/hobbyfarm.io/v1"
	hfClientset "github.com/hobbyfarm/gargantua/v3/pkg/client/clientset/versioned"
	hfClientsetv1 "github.com/hobbyfarm/gargantua/v3/pkg/client/clientset/versioned/typed/hobbyfarm.io/v1"
	hfInformers "github.com/hobbyfarm/gargantua/v3/pkg/client/informers/externalversions"
	listersv1 "github.com/hobbyfarm/gargantua/v3/pkg/client/listers/hobbyfarm.io/v1"
	hferrors "github.com/hobbyfarm/gargantua/v3/pkg/errors"
	"github.com/hobbyfarm/gargantua/v3/pkg/util"
	generalpb "github.com/hobbyfarm/gargantua/v3/protos/general"
	quizpb "github.com/hobbyfarm/gargantua/v3/protos/quiz"
	"google.golang.org/grpc/codes"
	"google.golang.org/protobuf/types/known/emptypb"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/retry"
	"strings"
)

type GrpcQuizEvaluationServer struct {
	quizpb.UnimplementedQuizEvaluationSvcServer
	client hfClientsetv1.QuizEvaluationInterface
	lister listersv1.QuizEvaluationLister
	synced cache.InformerSynced
}

func NewGrpcQuizEvaluationServer(hfClientSet hfClientset.Interface, hfInformerFactory hfInformers.SharedInformerFactory) *GrpcQuizEvaluationServer {
	return &GrpcQuizEvaluationServer{
		client: hfClientSet.HobbyfarmV1().QuizEvaluations(util.GetReleaseNamespace()),
		lister: hfInformerFactory.Hobbyfarm().V1().QuizEvaluations().Lister(),
		synced: hfInformerFactory.Hobbyfarm().V1().QuizEvaluations().Informer().HasSynced,
	}
}

func resourceName(quiz, user, scenario string) string {
	fields := strings.Join([]string{quiz, user, scenario}, "")
	return util.GenerateResourceName("quizeval", fields, 16)
}

func (gqes GrpcQuizEvaluationServer) CreateQuizEvaluation(ctx context.Context, req *quizpb.CreateQuizEvaluationRequest) (*generalpb.ResourceId, error) {
	quizEvaluationId := resourceName(req.Quiz, req.User, req.Scenario)

	corrects := make(map[string][]string)
	for questionId, answerIds := range req.GetAttempt().GetCorrects() {
		corrects[questionId] = answerIds.GetValues()
	}

	selects := make(map[string][]string)
	for questionId, answerIds := range req.GetAttempt().GetSelects() {
		selects[questionId] = answerIds.GetValues()
	}

	attempts := []hfv1.QuizEvaluationAttempt{
		{
			CreationTimestamp: req.GetAttempt().GetCreationTimestamp(),
			Timestamp:         req.GetAttempt().GetTimestamp(),
			Attempt:           req.GetAttempt().GetAttempt(),
			Score:             req.GetAttempt().GetScore(),
			Pass:              req.GetAttempt().GetPass(),
			Corrects:          corrects,
			Selects:           selects,
		},
	}

	quizEvaluation := &hfv1.QuizEvaluation{
		ObjectMeta: metav1.ObjectMeta{
			Name: quizEvaluationId,
			OwnerReferences: []metav1.OwnerReference{
				{
					APIVersion: "hobbyfarm.io/v1",
					Kind:       "Quiz",
					Name:       req.GetQuiz(),
					UID:        types.UID(req.GetQuizUid()),
				},
			},
		},
		Spec: hfv1.QuizEvaluationSpec{
			Quiz:     req.GetQuiz(),
			User:     req.GetUser(),
			Scenario: req.GetScenario(),
			Attempts: attempts,
		},
	}

	_, err := gqes.client.Create(ctx, quizEvaluation, metav1.CreateOptions{})
	if err != nil {
		return &generalpb.ResourceId{}, hferrors.GrpcError(
			codes.Internal,
			err.Error(),
			req,
		)
	}
	return &generalpb.ResourceId{Id: quizEvaluationId}, nil
}

func (gqes GrpcQuizEvaluationServer) GetQuizEvaluation(ctx context.Context, req *generalpb.GetRequest) (*quizpb.QuizEvaluation, error) {
	quizEvaluationId, err := util.GenericHfGetter(ctx, req, gqes.client, gqes.lister.QuizEvaluations(util.GetReleaseNamespace()), "quizevaluation", gqes.synced())
	if err != nil {
		return &quizpb.QuizEvaluation{}, err
	}

	return NewPBQuizEvaluation(quizEvaluationId), nil
}

func (gqes GrpcQuizEvaluationServer) GetQuizEvaluationForUser(ctx context.Context, req *quizpb.GetQuizEvaluationForUserRequest) (*quizpb.QuizEvaluation, error) {
	evaluationId := resourceName(req.GetQuiz(), req.GetUser(), req.GetScenario())
	return gqes.GetQuizEvaluation(ctx, &generalpb.GetRequest{Id: evaluationId})
}

func (gqes GrpcQuizEvaluationServer) UpdateQuizEvaluation(ctx context.Context, req *quizpb.UpdateQuizEvaluationRequest) (*emptypb.Empty, error) {
	id := req.GetId()
	if len(id) == 0 {
		return &emptypb.Empty{}, hferrors.GrpcIdNotSpecifiedError(req)
	}

	retryErr := retry.RetryOnConflict(retry.DefaultRetry, func() error {
		quizEvaluation, err := gqes.client.Get(ctx, id, metav1.GetOptions{})
		if err != nil {
			glog.Error(err)
			return hferrors.GrpcError(
				codes.Internal,
				"error while retrieving quiz %s",
				req,
				req.GetId(),
			)
		}

		quizEvaluation.Spec.Quiz = req.GetQuiz()
		quizEvaluation.Spec.User = req.GetUser()
		quizEvaluation.Spec.Scenario = req.GetScenario()

		attempts := make([]hfv1.QuizEvaluationAttempt, len(req.GetAttempts()))
		for i, attempt := range req.GetAttempts() {
			corrects := make(map[string][]string)
			for questionId, answerIds := range attempt.GetCorrects() {
				corrects[questionId] = answerIds.GetValues()
			}

			selects := make(map[string][]string)
			for questionId, answerIds := range attempt.GetSelects() {
				selects[questionId] = answerIds.GetValues()
			}

			attempts[i] = hfv1.QuizEvaluationAttempt{
				CreationTimestamp: attempt.GetCreationTimestamp(),
				Timestamp:         attempt.GetTimestamp(),
				Attempt:           attempt.GetAttempt(),
				Score:             attempt.GetScore(),
				Pass:              attempt.GetPass(),
				Corrects:          corrects,
				Selects:           selects,
			}
		}
		quizEvaluation.Spec.Attempts = attempts

		_, updateErr := gqes.client.Update(ctx, quizEvaluation, metav1.UpdateOptions{})
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

func (gqes GrpcQuizEvaluationServer) DeleteQuizEvaluation(ctx context.Context, req *generalpb.ResourceId) (*emptypb.Empty, error) {
	return util.DeleteHfResource(ctx, req, gqes.client, "quizevaluation")
}

func (gqes GrpcQuizEvaluationServer) ListQuizEvaluation(ctx context.Context, listOptions *generalpb.ListOptions) (*quizpb.ListQuizEvaluationsResponse, error) {
	doLoadFromCache := listOptions.GetLoadFromCache()
	var quizEvaluations []hfv1.QuizEvaluation
	var err error
	if !doLoadFromCache {
		var quizEvaluationList *hfv1.QuizEvaluationList
		quizEvaluationList, err = util.ListByHfClient(ctx, listOptions, gqes.client, "quizevaluations")
		if err == nil {
			quizEvaluations = quizEvaluationList.Items
		}
	} else {
		quizEvaluations, err = util.ListByCache(listOptions, gqes.lister, "quizevaluations", gqes.synced())
	}
	if err != nil {
		glog.Error(err)
		return &quizpb.ListQuizEvaluationsResponse{}, err
	}

	preparedQuizEvaluations := NewPBQuizEvaluationList(quizEvaluations)
	return &quizpb.ListQuizEvaluationsResponse{QuizEvaluations: preparedQuizEvaluations}, nil
}
