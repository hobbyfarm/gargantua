package quizservice

import (
	"github.com/hobbyfarm/gargantua/services/quizsvc/v3/internal/quiz"
	"github.com/hobbyfarm/gargantua/services/quizsvc/v3/internal/quizevaluation"
	authnpb "github.com/hobbyfarm/gargantua/v3/protos/authn"
	authrpb "github.com/hobbyfarm/gargantua/v3/protos/authr"

	"github.com/golang/glog"
	"github.com/gorilla/mux"
)

type QuizServer struct {
	internalQuizService           *quiz.QuizService
	internalQuizEvaluationService *quizevaluation.QuizEvaluationService
}

func NewQuizServer(
	authnClient authnpb.AuthNClient,
	authrClient authrpb.AuthRClient,
	internalQuizServer *quiz.GrpcQuizServer,
	internalQuizEvaluationServer *quizevaluation.GrpcQuizEvaluationServer,
) QuizServer {
	return QuizServer{
		internalQuizService:           quiz.NewQuizService(authnClient, authrClient, internalQuizServer),
		internalQuizEvaluationService: quizevaluation.NewQuizEvaluationService(authnClient, authrClient, internalQuizEvaluationServer, internalQuizServer),
	}
}

func (qs QuizServer) SetupRoutes(r *mux.Router) {
	// quiz
	r.HandleFunc("/a/quiz/list", qs.internalQuizService.ListFunc).Methods("GET")
	r.HandleFunc("/a/quiz/{id}", qs.internalQuizService.GetFunc).Methods("GET")
	r.HandleFunc("/a/quiz/create", qs.internalQuizService.CreateFunc).Methods("POST")
	r.HandleFunc("/a/quiz/{id}/update", qs.internalQuizService.UpdateFunc).Methods("PUT")
	r.HandleFunc("/a/quiz/{id}/delete", qs.internalQuizService.DeleteFunc).Methods("DELETE")
	r.HandleFunc("/quiz/{id}", qs.internalQuizService.GetForUserFunc).Methods("GET")
	// quiz score
	r.HandleFunc("/a/quiz/evaluation/{id}", qs.internalQuizEvaluationService.GetFunc).Methods("GET")
	r.HandleFunc("/a/quiz/evaluation/{id}/delete", qs.internalQuizEvaluationService.DeleteFunc).Methods("DELETE")
	r.HandleFunc("/quiz/evaluation/create", qs.internalQuizEvaluationService.CreateFunc).Methods("POST")
	r.HandleFunc("/quiz/evaluation/{quiz_id}/{scenario_id}", qs.internalQuizEvaluationService.GetForUserFunc).Methods("GET")
	glog.V(2).Infof("set up routes for quiz server")
}
