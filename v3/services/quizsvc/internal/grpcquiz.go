package quizservice

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
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/retry"
)

type GrpcQuizServer struct {
	quizpb.UnimplementedQuizSvcServer
	quizClient hfClientsetv1.QuizInterface
	quizLister listersv1.QuizLister
	quizSynced cache.InformerSynced
}

func NewGrpcQuizServer(hfClientSet hfClientset.Interface, hfInformerFactory hfInformers.SharedInformerFactory) *GrpcQuizServer {
	return &GrpcQuizServer{
		quizClient: hfClientSet.HobbyfarmV1().Quizes(util.GetReleaseNamespace()),
		quizLister: hfInformerFactory.Hobbyfarm().V1().Quizes().Lister(),
		quizSynced: hfInformerFactory.Hobbyfarm().V1().Quizes().Informer().HasSynced,
	}
}

func randomResourceName(prefix string) string {
	random := util.RandStringRunes(16)
	return util.GenerateResourceName(prefix, random, 16)
}

func (gqs *GrpcQuizServer) CreateQuiz(ctx context.Context, req *quizpb.CreateQuizRequest) (*generalpb.ResourceId, error) {
	quizId := randomResourceName("quiz")

	questions := make([]hfv1.QuizQuestion, len(req.GetQuestions()))
	for i, question := range req.GetQuestions() {
		answers := make([]hfv1.QuizAnswer, len(question.GetAnswers()))

		for j, answer := range question.GetAnswers() {
			answers[j] = hfv1.QuizAnswer{
				Title:   answer.GetTitle(),
				Correct: answer.GetCorrect(),
			}
		}

		questions[i] = hfv1.QuizQuestion{
			Title:          question.GetTitle(),
			Description:    question.GetDescription(),
			Type:           question.GetType(),
			Shuffle:        question.GetShuffle(),
			FailureMessage: question.GetFailureMessage(),
			SuccessMessage: question.GetSuccessMessage(),
			ValidationType: question.GetValidationType(),
			Weight:         question.GetWeight(),
			Answers:        answers,
		}
	}

	quiz := &hfv1.Quiz{
		ObjectMeta: metav1.ObjectMeta{
			Name: quizId,
		},
		Spec: hfv1.QuizSpec{
			Title:            req.GetTitle(),
			Type:             req.GetType(),
			Shuffle:          req.GetShuffle(),
			PoolSize:         req.GetPoolSize(),
			MaxAttempts:      req.GetMaxAttempts(),
			SuccessThreshold: req.GetSuccessThreshold(),
			Questions:        questions,
		},
	}

	_, err := gqs.quizClient.Create(ctx, quiz, metav1.CreateOptions{})
	if err != nil {
		return &generalpb.ResourceId{}, hferrors.GrpcError(
			codes.Internal,
			err.Error(),
			req,
		)
	}
	return &generalpb.ResourceId{Id: quizId}, nil
}

func (gqs *GrpcQuizServer) GetQuiz(ctx context.Context, req *generalpb.GetRequest) (*quizpb.Quiz, error) {
	quiz, err := util.GenericHfGetter(ctx, req, gqs.quizClient, gqs.quizLister.Quizes(util.GetReleaseNamespace()), "quiz", gqs.quizSynced())
	if err != nil {
		return &quizpb.Quiz{}, err
	}

	questions := make([]*quizpb.QuizQuestion, len(quiz.Spec.Questions))
	for i, question := range quiz.Spec.Questions {
		answers := make([]*quizpb.QuizAnswer, len(question.Answers))

		for j, answer := range question.Answers {
			answerId := util.GenerateResourceName("answ", answer.Title, 16)
			answers[j] = &quizpb.QuizAnswer{
				Id:      answerId,
				Title:   answer.Title,
				Correct: answer.Correct,
			}
		}

		questionId := util.GenerateResourceName("qsn", question.Title, 16)
		questions[i] = &quizpb.QuizQuestion{
			Id:             questionId,
			Title:          question.Title,
			Description:    question.Description,
			Type:           question.Type,
			Shuffle:        question.Shuffle,
			FailureMessage: question.FailureMessage,
			SuccessMessage: question.SuccessMessage,
			ValidationType: question.ValidationType,
			Weight:         question.Weight,
			Answers:        answers,
		}
	}

	return &quizpb.Quiz{
		Id:               quiz.Name,
		Uid:              string(quiz.UID),
		Title:            quiz.Spec.Title,
		Type:             quiz.Spec.Type,
		Shuffle:          quiz.Spec.Shuffle,
		PoolSize:         quiz.Spec.PoolSize,
		MaxAttempts:      quiz.Spec.MaxAttempts,
		SuccessThreshold: quiz.Spec.SuccessThreshold,
		Questions:        questions,
	}, nil
}

func (gqs *GrpcQuizServer) UpdateQuiz(ctx context.Context, req *quizpb.UpdateQuizRequest) (*emptypb.Empty, error) {
	id := req.GetId()
	if len(id) == 0 {
		return &emptypb.Empty{}, hferrors.GrpcIdNotSpecifiedError(req)
	}

	retryErr := retry.RetryOnConflict(retry.DefaultRetry, func() error {
		quiz, err := gqs.quizClient.Get(ctx, id, metav1.GetOptions{})
		if err != nil {
			glog.Error(err)
			return hferrors.GrpcError(
				codes.Internal,
				"error while retrieving quiz %s",
				req,
				req.GetId(),
			)
		}
		updatedQuiz := gqs.updateQuiz(req, quiz)
		_, updateErr := gqs.quizClient.Update(ctx, updatedQuiz, metav1.UpdateOptions{})
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

func (gqs *GrpcQuizServer) DeleteQuiz(ctx context.Context, req *generalpb.ResourceId) (*emptypb.Empty, error) {
	return util.DeleteHfResource(ctx, req, gqs.quizClient, "quiz")
}

func (gqs *GrpcQuizServer) ListQuiz(ctx context.Context, listOptions *generalpb.ListOptions) (*quizpb.ListQuizzesResponse, error) {
	doLoadFromCache := listOptions.GetLoadFromCache()
	var quizzes []hfv1.Quiz
	var err error
	if !doLoadFromCache {
		var quizList *hfv1.QuizList
		quizList, err = util.ListByHfClient(ctx, listOptions, gqs.quizClient, "quizes")
		if err == nil {
			quizzes = quizList.Items
		}
	} else {
		quizzes, err = util.ListByCache(listOptions, gqs.quizLister, "quizes", gqs.quizSynced())
	}
	if err != nil {
		glog.Error(err)
		return &quizpb.ListQuizzesResponse{}, err
	}

	preparedQuizzes := make([]*quizpb.Quiz, len(quizzes))

	for i, quiz := range quizzes {
		questions := make([]*quizpb.QuizQuestion, len(quiz.Spec.Questions))
		for j, question := range quiz.Spec.Questions {
			answers := make([]*quizpb.QuizAnswer, len(question.Answers))

			for k, answer := range question.Answers {
				answerId := util.GenerateResourceName("answ", answer.Title, 16)
				answers[k] = &quizpb.QuizAnswer{
					Id:      answerId,
					Title:   answer.Title,
					Correct: answer.Correct,
				}
			}

			questionId := util.GenerateResourceName("qsn", question.Title, 16)
			questions[j] = &quizpb.QuizQuestion{
				Id:             questionId,
				Title:          question.Title,
				Description:    question.Description,
				Type:           question.Type,
				Shuffle:        question.Shuffle,
				FailureMessage: question.FailureMessage,
				SuccessMessage: question.SuccessMessage,
				ValidationType: question.ValidationType,
				Weight:         question.Weight,
				Answers:        answers,
			}
		}

		preparedQuizzes[i] = &quizpb.Quiz{
			Id:               quiz.Name,
			Uid:              string(quiz.UID),
			Title:            quiz.Spec.Title,
			Type:             quiz.Spec.Type,
			Shuffle:          quiz.Spec.Shuffle,
			PoolSize:         quiz.Spec.PoolSize,
			MaxAttempts:      quiz.Spec.MaxAttempts,
			SuccessThreshold: quiz.Spec.SuccessThreshold,
			Questions:        questions,
		}
	}

	return &quizpb.ListQuizzesResponse{Quizzes: preparedQuizzes}, nil
}

func (gqs *GrpcQuizServer) updateQuiz(req *quizpb.UpdateQuizRequest, source *hfv1.Quiz) *hfv1.Quiz {
	source.Spec.Title = req.GetTitle()
	source.Spec.Type = req.GetType()
	source.Spec.Shuffle = req.GetShuffle()
	source.Spec.PoolSize = req.GetPoolSize()
	source.Spec.MaxAttempts = req.GetMaxAttempts()
	source.Spec.SuccessThreshold = req.GetSuccessThreshold()

	questions := make([]hfv1.QuizQuestion, len(req.GetQuestions()))

	for i, question := range req.GetQuestions() {
		answers := make([]hfv1.QuizAnswer, len(question.GetAnswers()))

		for j, answer := range question.GetAnswers() {
			answers[j] = hfv1.QuizAnswer{
				Title:   answer.GetTitle(),
				Correct: answer.GetCorrect(),
			}
		}

		questions[i] = hfv1.QuizQuestion{
			Title:          question.GetTitle(),
			Description:    question.GetDescription(),
			Type:           question.GetType(),
			Shuffle:        question.GetShuffle(),
			FailureMessage: question.GetFailureMessage(),
			SuccessMessage: question.GetSuccessMessage(),
			ValidationType: question.GetValidationType(),
			Weight:         question.GetWeight(),
			Answers:        answers,
		}
	}

	source.Spec.Questions = questions

	return source
}
