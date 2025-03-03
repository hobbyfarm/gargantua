package quizservice

import (
	authnpb "github.com/hobbyfarm/gargantua/v3/protos/authn"
	authrpb "github.com/hobbyfarm/gargantua/v3/protos/authr"

	"github.com/golang/glog"
	"github.com/gorilla/mux"
)

type QuizServer struct {
	authnClient        authnpb.AuthNClient
	authrClient        authrpb.AuthRClient
	internalQuizServer *GrpcQuizServer
}

func NewQuizServer(
	authnClient authnpb.AuthNClient,
	authrClient authrpb.AuthRClient,
	internalQuizServer *GrpcQuizServer,
) QuizServer {
	return QuizServer{
		authnClient:        authnClient,
		authrClient:        authrClient,
		internalQuizServer: internalQuizServer,
	}
}

func (qs QuizServer) SetupRoutes(r *mux.Router) {
	// admin routes
	r.HandleFunc("/a/quiz/list", qs.ListFunc).Methods("GET")
	r.HandleFunc("/a/quiz/{id}", qs.GetFunc).Methods("GET")
	r.HandleFunc("/a/quiz/create", qs.CreateFunc).Methods("POST")
	r.HandleFunc("/a/quiz/{id}/update", qs.UpdateFunc).Methods("PUT")
	r.HandleFunc("/a/quiz/{id}/delete", qs.DeleteFunc).Methods("DELETE")
	// ui routes
	r.HandleFunc("/quiz/{id}", qs.GetForUserFunc).Methods("GET")
	glog.V(2).Infof("set up routes for admin quiz server")
}
