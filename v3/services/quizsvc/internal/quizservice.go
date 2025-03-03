package quizservice

import (
	"encoding/json"
	"fmt"
	"github.com/golang/glog"
	"github.com/gorilla/mux"
	hferrors "github.com/hobbyfarm/gargantua/v3/pkg/errors"
	"github.com/hobbyfarm/gargantua/v3/pkg/rbac"
	"github.com/hobbyfarm/gargantua/v3/pkg/util"
	generalpb "github.com/hobbyfarm/gargantua/v3/protos/general"
	quizpb "github.com/hobbyfarm/gargantua/v3/protos/quiz"
	"google.golang.org/grpc/status"
	"net/http"
)

type PreparedQuiz struct {
	Id               string             `json:"id,omitempty"`
	Title            string             `json:"title"`
	Type             string             `json:"type"`
	Shuffle          bool               `json:"shuffle"`
	PoolSize         uint32             `json:"pool_size"`
	MaxAttempts      uint32             `json:"max_attempts"`
	SuccessThreshold uint32             `json:"success_threshold"`
	Questions        []PreparedQuestion `json:"questions"`
}

func NewPreparedQuiz(quiz *quizpb.Quiz, showCorrect bool) PreparedQuiz {
	questions := make([]PreparedQuestion, len(quiz.GetQuestions()))
	for i, question := range quiz.GetQuestions() {
		questions[i] = NewPreparedQuestion(question, showCorrect)
	}
	return PreparedQuiz{
		Id:               quiz.GetId(),
		Title:            quiz.GetTitle(),
		Type:             quiz.GetType(),
		Shuffle:          quiz.GetShuffle(),
		PoolSize:         quiz.GetPoolSize(),
		MaxAttempts:      quiz.GetMaxAttempts(),
		SuccessThreshold: quiz.GetSuccessThreshold(),
		Questions:        questions,
	}
}

type PreparedQuestion struct {
	Id             string           `json:"id,omitempty"`
	Title          string           `json:"title"`
	Description    string           `json:"description"`
	Type           string           `json:"type"`
	Shuffle        bool             `json:"shuffle"`
	FailureMessage string           `json:"failure_message"`
	SuccessMessage string           `json:"success_message"`
	ValidationType string           `json:"validation_type"`
	Weight         uint32           `json:"weight"`
	Answers        []PreparedAnswer `json:"answers"`
}

func NewPreparedQuestion(question *quizpb.QuizQuestion, showCorrect bool) PreparedQuestion {
	answers := make([]PreparedAnswer, len(question.GetAnswers()))
	for i, answer := range question.GetAnswers() {
		answers[i] = NewPreparedAnswer(answer, showCorrect)
	}
	return PreparedQuestion{
		Id:             question.GetId(),
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

type PreparedAnswer struct {
	Id      string `json:"id,omitempty"`
	Title   string `json:"title"`
	Correct *bool  `json:"correct,omitempty"`
}

func NewPreparedAnswer(answer *quizpb.QuizAnswer, showCorrect bool) PreparedAnswer {
	var correct *bool
	if showCorrect {
		correct = util.Ref[bool](answer.GetCorrect())
	}

	return PreparedAnswer{
		Id:      answer.GetId(),
		Title:   answer.GetTitle(),
		Correct: correct,
	}
}

func (qs QuizServer) GetFunc(w http.ResponseWriter, r *http.Request) {
	user, err := rbac.AuthenticateRequest(r, qs.authnClient)
	if err != nil {
		util.ReturnHTTPMessage(w, r, 401, "unauthorized", "authentication failed")
		return
	}

	impersonatedUserId := user.GetId()
	authrResponse, err := rbac.AuthorizeSimple(r, qs.authrClient, impersonatedUserId, rbac.HobbyfarmPermission(rbac.ResourcePluralQuiz, rbac.VerbGet))
	if err != nil || !authrResponse.Success {
		util.ReturnHTTPMessage(w, r, 403, "forbidden", "no access to get quiz")
		return
	}

	vars := mux.Vars(r)

	quizId := vars["id"]

	if len(quizId) == 0 {
		util.ReturnHTTPMessage(w, r, 400, "bad request", "no quiz id passed in")
		return
	}

	quiz, err := qs.internalQuizServer.GetQuiz(r.Context(), &generalpb.GetRequest{Id: quizId})
	if err != nil {
		glog.Errorf("error while retrieving quiz: %s", hferrors.GetErrorMessage(err))
		if hferrors.IsGrpcNotFound(err) {
			errMsg := fmt.Sprintf("quiz %s not found", quizId)
			util.ReturnHTTPMessage(w, r, http.StatusNotFound, "not found", errMsg)
			return
		}
		errMsg := fmt.Sprintf("error retrieving quiz %s", quizId)
		util.ReturnHTTPMessage(w, r, http.StatusInternalServerError, "error", errMsg)
		return
	}

	preparedQuiz := NewPreparedQuiz(quiz, true)
	encodedQuiz, err := json.Marshal(preparedQuiz)
	if err != nil {
		glog.Error(err)
	}
	util.ReturnHTTPContent(w, r, 200, "success", encodedQuiz)

	glog.V(2).Infof("retrieved quiz %s", quizId)
}

func (qs QuizServer) GetForUserFunc(w http.ResponseWriter, r *http.Request) {
	user, err := rbac.AuthenticateRequest(r, qs.authnClient)
	if err != nil {
		util.ReturnHTTPMessage(w, r, 401, "unauthorized", "authentication failed")
		return
	}

	impersonatedUserId := user.GetId()
	authrResponse, err := rbac.AuthorizeSimple(r, qs.authrClient, impersonatedUserId, rbac.HobbyfarmPermission(rbac.ResourcePluralQuiz, rbac.VerbGet))
	if err != nil || !authrResponse.Success {
		util.ReturnHTTPMessage(w, r, 403, "forbidden", "no access to quiz")
		return
	}

	vars := mux.Vars(r)

	quizId := vars["id"]

	if len(quizId) == 0 {
		util.ReturnHTTPMessage(w, r, 400, "bad request", "no quiz id passed in")
		return
	}

	quiz, err := qs.internalQuizServer.GetQuiz(r.Context(), &generalpb.GetRequest{Id: quizId})
	if err != nil {
		glog.Errorf("error while retrieving quiz: %s", hferrors.GetErrorMessage(err))
		if hferrors.IsGrpcNotFound(err) {
			errMsg := fmt.Sprintf("quiz %s not found", quizId)
			util.ReturnHTTPMessage(w, r, http.StatusNotFound, "not found", errMsg)
			return
		}
		errMsg := fmt.Sprintf("error retrieving quiz %s", quizId)
		util.ReturnHTTPMessage(w, r, http.StatusInternalServerError, "error", errMsg)
		return
	}

	preparedQuiz := NewPreparedQuiz(quiz, false)
	encodedQuiz, err := json.Marshal(preparedQuiz)
	if err != nil {
		glog.Error(err)
	}
	util.ReturnHTTPContent(w, r, 200, "success", encodedQuiz)

	glog.V(2).Infof("retrieved quiz %s", quizId)
}

func (qs QuizServer) ListFunc(w http.ResponseWriter, r *http.Request) {
	user, err := rbac.AuthenticateRequest(r, qs.authnClient)
	if err != nil {
		util.ReturnHTTPMessage(w, r, 401, "unauthorized", "authentication failed")
		return
	}

	impersonatedUserId := user.GetId()
	authrResponse, err := rbac.AuthorizeSimple(r, qs.authrClient, impersonatedUserId, rbac.HobbyfarmPermission(rbac.ResourcePluralQuiz, rbac.VerbList))
	if err != nil || !authrResponse.Success {
		util.ReturnHTTPMessage(w, r, 403, "forbidden", "no access to list quizzes")
		return
	}

	quizList, err := qs.internalQuizServer.ListQuiz(r.Context(), &generalpb.ListOptions{})
	if err != nil {
		glog.Errorf("error while listing all quizzes: %s", hferrors.GetErrorMessage(err))
		util.ReturnHTTPMessage(w, r, 500, "error", "error listing all quizzes")
		return
	}

	preparedQuizzes := make([]PreparedQuiz, len(quizList.GetQuizzes()))
	for i, quiz := range quizList.GetQuizzes() {
		preparedQuizzes[i] = NewPreparedQuiz(quiz, true)
	}

	encodedQuizzes, err := json.Marshal(preparedQuizzes)
	if err != nil {
		glog.Error(err)
	}
	util.ReturnHTTPContent(w, r, 200, "success", encodedQuizzes)

	glog.V(2).Infof("retrieved list of all quizzes")
}

func (qs QuizServer) CreateFunc(w http.ResponseWriter, r *http.Request) {
	user, err := rbac.AuthenticateRequest(r, qs.authnClient)
	if err != nil {
		util.ReturnHTTPMessage(w, r, 401, "unauthorized", "authentication failed")
		return
	}

	impersonatedUserId := user.GetId()
	authrResponse, err := rbac.AuthorizeSimple(r, qs.authrClient, impersonatedUserId, rbac.HobbyfarmPermission(rbac.ResourcePluralQuiz, rbac.VerbCreate))
	if err != nil || !authrResponse.Success {
		util.ReturnHTTPMessage(w, r, 403, "forbidden", "no access to create quiz")
		return
	}

	var preparedQuiz PreparedQuiz

	// Decode JSON body
	if err = json.NewDecoder(r.Body).Decode(&preparedQuiz); err != nil {
		util.ReturnHTTPMessage(w, r, 400, "badrequest", "invalid json body")
		return
	}

	questions := make([]*quizpb.CreateQuizQuestion, len(preparedQuiz.Questions))
	for i, question := range preparedQuiz.Questions {
		answers := make([]*quizpb.CreateQuizAnswer, len(question.Answers))
		for j, answer := range question.Answers {
			answers[j] = &quizpb.CreateQuizAnswer{
				Title:   answer.Title,
				Correct: util.DerefOrDefault[bool](answer.Correct),
			}
		}

		questions[i] = &quizpb.CreateQuizQuestion{
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

	quizId, err := qs.internalQuizServer.CreateQuiz(r.Context(), &quizpb.CreateQuizRequest{
		Title:            preparedQuiz.Title,
		Type:             preparedQuiz.Type,
		Shuffle:          preparedQuiz.Shuffle,
		PoolSize:         preparedQuiz.PoolSize,
		MaxAttempts:      preparedQuiz.MaxAttempts,
		SuccessThreshold: preparedQuiz.SuccessThreshold,
		Questions:        questions,
	})

	if err != nil {
		statusErr := status.Convert(err)
		if hferrors.IsGrpcParsingError(err) {
			glog.Errorf("error while parsing: %s", statusErr.Message())
			util.ReturnHTTPMessage(w, r, 500, "internalerror", "error parsing")
			return
		}
		glog.Errorf("error creating quiz %s", hferrors.GetErrorMessage(err))
		util.ReturnHTTPMessage(w, r, 500, "internalerror", "error creating quiz")
		return
	}

	util.ReturnHTTPMessage(w, r, 201, "created", quizId.GetId())
	glog.V(4).Infof("Created quiz %s", quizId.GetId())
}

func (qs QuizServer) UpdateFunc(w http.ResponseWriter, r *http.Request) {
	user, err := rbac.AuthenticateRequest(r, qs.authnClient)
	if err != nil {
		util.ReturnHTTPMessage(w, r, 401, "unauthorized", "authentication failed")
		return
	}

	impersonatedUserId := user.GetId()
	authrResponse, err := rbac.AuthorizeSimple(r, qs.authrClient, impersonatedUserId, rbac.HobbyfarmPermission(rbac.ResourcePluralQuiz, rbac.VerbUpdate))
	if err != nil || !authrResponse.Success {
		util.ReturnHTTPMessage(w, r, 403, "forbidden", "no access to update quiz")
		return
	}

	vars := mux.Vars(r)
	id := vars["id"]

	if len(id) == 0 {
		util.ReturnHTTPMessage(w, r, 500, "error", "no id passed in")
		return
	}

	var preparedQuiz PreparedQuiz

	// Decode JSON body
	if err = json.NewDecoder(r.Body).Decode(&preparedQuiz); err != nil {
		util.ReturnHTTPMessage(w, r, 400, "badrequest", "invalid json body")
		return
	}

	questions := make([]*quizpb.UpdateQuizQuestion, len(preparedQuiz.Questions))
	for i, question := range preparedQuiz.Questions {
		answers := make([]*quizpb.UpdateQuizAnswer, len(question.Answers))
		for j, answer := range question.Answers {
			answers[j] = &quizpb.UpdateQuizAnswer{
				Title:   answer.Title,
				Correct: util.DerefOrDefault[bool](answer.Correct),
			}
		}

		questions[i] = &quizpb.UpdateQuizQuestion{
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

	_, err = qs.internalQuizServer.UpdateQuiz(r.Context(), &quizpb.UpdateQuizRequest{
		Id:               id,
		Title:            preparedQuiz.Title,
		Type:             preparedQuiz.Type,
		Shuffle:          preparedQuiz.Shuffle,
		PoolSize:         preparedQuiz.PoolSize,
		MaxAttempts:      preparedQuiz.MaxAttempts,
		SuccessThreshold: preparedQuiz.SuccessThreshold,
		Questions:        questions,
	})

	if err != nil {
		glog.Error(hferrors.GetErrorMessage(err))
		util.ReturnHTTPMessage(w, r, 500, "internalerror", "error attempting to update")
		return
	}

	util.ReturnHTTPMessage(w, r, 200, "updated", "")
	glog.V(4).Infof("Updated quiz %s", id)
}

func (qs QuizServer) DeleteFunc(w http.ResponseWriter, r *http.Request) {
	user, err := rbac.AuthenticateRequest(r, qs.authnClient)
	if err != nil {
		util.ReturnHTTPMessage(w, r, 401, "unauthorized", "authentication failed")
		return
	}

	impersonatedUserId := user.GetId()
	authrResponse, err := rbac.AuthorizeSimple(r, qs.authrClient, impersonatedUserId, rbac.HobbyfarmPermission(rbac.ResourcePluralQuiz, rbac.VerbDelete))
	if err != nil || !authrResponse.Success {
		util.ReturnHTTPMessage(w, r, 403, "forbidden", "no access to delete quiz")
		return
	}

	// first, check if the quiz exists
	vars := mux.Vars(r)
	quizId := vars["id"]
	if quizId == "" {
		util.ReturnHTTPMessage(w, r, 400, "badrequest", "no id passed in")
		return
	}

	glog.V(2).Infof("user %s deleting quiz %s", user.GetId(), quizId)

	// first check if the quiz actually exists
	_, err = qs.internalQuizServer.GetQuiz(r.Context(), &generalpb.GetRequest{Id: quizId})
	if err != nil {
		glog.Errorf("error while retrieving quiz: %s", hferrors.GetErrorMessage(err))
		if hferrors.IsGrpcNotFound(err) {
			errMsg := fmt.Sprintf("error retrieving quiz while attempting quiz deletion: quiz %s not found", quizId)
			util.ReturnHTTPMessage(w, r, http.StatusNotFound, "not found", errMsg)
			return
		}
		errMsg := fmt.Sprintf("error retrieving quiz %s while attempting quiz deletion", quizId)
		util.ReturnHTTPMessage(w, r, http.StatusInternalServerError, "error", errMsg)
		return
	}

	// TODO @MARCUS also delete all quiz progresses
	_, err = qs.internalQuizServer.DeleteQuiz(r.Context(), &generalpb.ResourceId{Id: quizId})
	if err != nil {
		glog.Errorf("error deleting quiz: %s", hferrors.GetErrorMessage(err))
		util.ReturnHTTPMessage(w, r, 500, "internalerror", "error deleting quiz")
		return
	}

	util.ReturnHTTPMessage(w, r, 200, "deleted", "quiz deleted")
	glog.V(4).Infof("deleted quiz: %s", quizId)
}
