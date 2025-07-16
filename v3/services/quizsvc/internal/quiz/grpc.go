package quiz

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
	client hfClientsetv1.QuizInterface
	lister listersv1.QuizLister
	synced cache.InformerSynced
}

func NewGrpcQuizServer(hfClientSet hfClientset.Interface, hfInformerFactory hfInformers.SharedInformerFactory) *GrpcQuizServer {
	return &GrpcQuizServer{
		client: hfClientSet.HobbyfarmV1().Quizes(util.GetReleaseNamespace()),
		lister: hfInformerFactory.Hobbyfarm().V1().Quizes().Lister(),
		synced: hfInformerFactory.Hobbyfarm().V1().Quizes().Informer().HasSynced,
	}
}

func randomResourceName(prefix string) string {
	random := util.RandStringRunes(16)
	return util.GenerateResourceName(prefix, random, 16)
}

func (gqs *GrpcQuizServer) CreateQuiz(ctx context.Context, req *quizpb.CreateQuizRequest) (*generalpb.ResourceId, error) {
	quizId := randomResourceName("quiz")
	quiz := NewQuizFromCreate(quizId, req)

	_, err := gqs.client.Create(ctx, quiz, metav1.CreateOptions{})
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
	quiz, err := util.GenericHfGetter(ctx, req, gqs.client, gqs.lister.Quizes(util.GetReleaseNamespace()), "quiz", gqs.synced())
	if err != nil {
		return &quizpb.Quiz{}, err
	}

	return NewPBQuiz(quiz), nil
}

func (gqs *GrpcQuizServer) UpdateQuiz(ctx context.Context, req *quizpb.UpdateQuizRequest) (*emptypb.Empty, error) {
	id := req.GetId()
	if len(id) == 0 {
		return &emptypb.Empty{}, hferrors.GrpcIdNotSpecifiedError(req)
	}

	retryErr := retry.RetryOnConflict(retry.DefaultRetry, func() error {
		quiz, err := gqs.client.Get(ctx, id, metav1.GetOptions{})
		if err != nil {
			glog.Error(err)
			return hferrors.GrpcError(
				codes.Internal,
				"error while retrieving quiz %s",
				req,
				req.GetId(),
			)
		}
		updatedQuiz := NewQuizFromUpdate(req, quiz)
		_, updateErr := gqs.client.Update(ctx, updatedQuiz, metav1.UpdateOptions{})
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
	return util.DeleteHfResource(ctx, req, gqs.client, "quiz")
}

func (gqs *GrpcQuizServer) ListQuiz(ctx context.Context, listOptions *generalpb.ListOptions) (*quizpb.ListQuizzesResponse, error) {
	doLoadFromCache := listOptions.GetLoadFromCache()
	var quizzes []hfv1.Quiz
	var err error
	if !doLoadFromCache {
		var quizList *hfv1.QuizList
		quizList, err = util.ListByHfClient(ctx, listOptions, gqs.client, "quizes")
		if err == nil {
			quizzes = quizList.Items
		}
	} else {
		quizzes, err = util.ListByCache(listOptions, gqs.lister, "quizes", gqs.synced())
	}
	if err != nil {
		glog.Error(err)
		return &quizpb.ListQuizzesResponse{}, err
	}

	preparedQuizzes := NewPBQuizList(quizzes)
	return &quizpb.ListQuizzesResponse{Quizzes: preparedQuizzes}, nil
}
