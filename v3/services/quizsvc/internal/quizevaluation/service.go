package quizevaluation

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/golang/glog"
	"github.com/gorilla/mux"
	"github.com/hobbyfarm/gargantua/services/quizsvc/v3/internal/quiz"
	hferrors "github.com/hobbyfarm/gargantua/v3/pkg/errors"
	"github.com/hobbyfarm/gargantua/v3/pkg/rbac"
	"github.com/hobbyfarm/gargantua/v3/pkg/util"
	authnpb "github.com/hobbyfarm/gargantua/v3/protos/authn"
	authrpb "github.com/hobbyfarm/gargantua/v3/protos/authr"
	generalpb "github.com/hobbyfarm/gargantua/v3/protos/general"
	quizpb "github.com/hobbyfarm/gargantua/v3/protos/quiz"
	"google.golang.org/grpc/status"
	"net/http"
)

type QuizEvaluationService struct {
	authnClient        authnpb.AuthNClient
	authrClient        authrpb.AuthRClient
	internalServer     *GrpcQuizEvaluationServer
	internalQuizServer *quiz.GrpcQuizServer
}

func NewQuizEvaluationService(
	authnClient authnpb.AuthNClient,
	authrClient authrpb.AuthRClient,
	internalQuizEvaluationServer *GrpcQuizEvaluationServer,
	internalQuizServer *quiz.GrpcQuizServer,
) *QuizEvaluationService {
	return &QuizEvaluationService{
		authnClient:        authnClient,
		authrClient:        authrClient,
		internalServer:     internalQuizEvaluationServer,
		internalQuizServer: internalQuizServer,
	}
}

func (qes QuizEvaluationService) GetFunc(w http.ResponseWriter, r *http.Request) {
	user, err := rbac.AuthenticateRequest(r, qes.authnClient)
	if err != nil {
		util.ReturnHTTPMessage(w, r, 401, "unauthorized", "authentication failed")
		return
	}

	impersonatedUserId := user.GetId()
	authrResponse, err := rbac.AuthorizeSimple(r, qes.authrClient, impersonatedUserId, rbac.HobbyfarmPermission(rbac.ResourcePluralQuizEvaluation, rbac.VerbGet))
	if err != nil || !authrResponse.Success {
		util.ReturnHTTPMessage(w, r, 403, "forbidden", "no access to get quiz evaluation")
		return
	}

	vars := mux.Vars(r)

	quizEvaluationId := vars["id"]
	if len(quizEvaluationId) == 0 {
		util.ReturnHTTPMessage(w, r, 400, "bad request", "no quiz evaluation id passed in")
		return
	}

	quizEvaluation, err := qes.internalServer.GetQuizEvaluation(r.Context(), &generalpb.GetRequest{Id: quizEvaluationId})

	if err != nil {
		glog.Errorf("error while retrieving quiz evaluation: %s", hferrors.GetErrorMessage(err))
		if hferrors.IsGrpcNotFound(err) {
			errMsg := fmt.Sprintf("quiz evaluation %s not found", quizEvaluationId)
			util.ReturnHTTPMessage(w, r, http.StatusNotFound, "not found", errMsg)
			return
		}
		errMsg := fmt.Sprintf("error retrieving quiz evaluation %s", quizEvaluationId)
		util.ReturnHTTPMessage(w, r, http.StatusInternalServerError, "error", errMsg)
		return
	}

	preparedQuizEvaluation := NewPreparedQuizEvaluation(quizEvaluation)
	encodedQuiz, err := json.Marshal(preparedQuizEvaluation)
	if err != nil {
		glog.Error(err)
	}
	util.ReturnHTTPContent(w, r, 200, "success", encodedQuiz)

	glog.V(2).Infof("retrieved quiz evaluation %s", quizEvaluationId)
}

func (qes QuizEvaluationService) GetForUserFunc(w http.ResponseWriter, r *http.Request) {
	user, err := rbac.AuthenticateRequest(r, qes.authnClient)
	if err != nil {
		util.ReturnHTTPMessage(w, r, 401, "unauthorized", "authentication failed")
		return
	}

	impersonatedUserId := user.GetId()
	authrResponse, err := rbac.AuthorizeSimple(r, qes.authrClient, impersonatedUserId, rbac.HobbyfarmPermission(rbac.ResourcePluralQuizEvaluation, rbac.VerbGet))
	if err != nil || !authrResponse.Success {
		util.ReturnHTTPMessage(w, r, 403, "forbidden", "no access to get quiz evaluation")
		return
	}

	vars := mux.Vars(r)

	quizId := vars["quiz_id"]
	if len(quizId) == 0 {
		util.ReturnHTTPMessage(w, r, 400, "bad request", "no quiz id passed in")
		return
	}

	scenarioId := vars["scenario_id"]
	if len(scenarioId) == 0 {
		util.ReturnHTTPMessage(w, r, 400, "bad request", "no scenario id passed in")
		return
	}

	quiz, err := qes.getQuiz(r.Context(), quizId)
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

	req := &quizpb.GetQuizEvaluationForUserRequest{
		Quiz:     quizId,
		User:     impersonatedUserId,
		Scenario: scenarioId,
	}
	quizEvaluation, err := qes.internalServer.GetQuizEvaluationForUser(r.Context(), req)
	if err != nil {
		glog.Errorf("error while retrieving quiz evaluation: %s", hferrors.GetErrorMessage(err))
		if hferrors.IsGrpcNotFound(err) {
			errMsg := fmt.Sprintf("quiz evaluation %s not found", quizId)
			util.ReturnHTTPMessage(w, r, http.StatusNotFound, "not found", errMsg)
			return
		}
		errMsg := fmt.Sprintf("error retrieving quiz evaluation %s", quizId)
		util.ReturnHTTPMessage(w, r, http.StatusInternalServerError, "error", errMsg)
		return
	}

	attempts := make([]PreparedAttempt, len(quizEvaluation.GetAttempts()))
	for i, attempt := range quizEvaluation.GetAttempts() {
		attempts[i] = NewPreparedAttempt(quiz.ValidationType, attempt)
	}
	preparedQuizEvaluation := PreparedQuizEvaluation{
		Id:       quizEvaluation.GetId(),
		Quiz:     quizEvaluation.GetQuiz(),
		User:     quizEvaluation.GetUser(),
		Scenario: quizEvaluation.GetScenario(),
		Attempts: attempts,
	}
	encodedQuiz, err := json.Marshal(preparedQuizEvaluation)
	if err != nil {
		glog.Error(err)
	}
	util.ReturnHTTPContent(w, r, 200, "success", encodedQuiz)

	glog.V(2).Infof("retrieved quiz evaluation %s", quizId)
}

func (qes QuizEvaluationService) CreateFunc(w http.ResponseWriter, r *http.Request) {
	user, err := rbac.AuthenticateRequest(r, qes.authnClient)
	if err != nil {
		util.ReturnHTTPMessage(w, r, 401, "unauthorized", "authentication failed")
		return
	}

	impersonatedUserId := user.GetId()
	authrResponse, err := rbac.AuthorizeSimple(r, qes.authrClient, impersonatedUserId, rbac.HobbyfarmPermission(rbac.ResourcePluralQuizEvaluation, rbac.VerbCreate))
	if err != nil || !authrResponse.Success {
		util.ReturnHTTPMessage(w, r, 403, "forbidden", "no access to create quiz evaluation")
		return
	}

	var preparedCreateQuizEvaluation PreparedCreateQuizEvaluation

	// Decode JSON body
	if err = json.NewDecoder(r.Body).Decode(&preparedCreateQuizEvaluation); err != nil {
		util.ReturnHTTPMessage(w, r, 400, "badrequest", "invalid json body")
		return
	}

	req := &quizpb.GetQuizEvaluationForUserRequest{
		Quiz:     preparedCreateQuizEvaluation.Quiz,
		User:     impersonatedUserId,
		Scenario: preparedCreateQuizEvaluation.Scenario,
	}

	quiz, err := qes.getQuiz(r.Context(), req.GetQuiz())
	if err != nil {
		glog.Errorf("error while retrieving quiz: %s", hferrors.GetErrorMessage(err))
		if hferrors.IsGrpcNotFound(err) {
			errMsg := fmt.Sprintf("quiz %s not found", req.GetQuiz())
			util.ReturnHTTPMessage(w, r, http.StatusNotFound, "not found", errMsg)
			return
		}
		errMsg := fmt.Sprintf("error retrieving quiz %s", req.GetQuiz())
		util.ReturnHTTPMessage(w, r, http.StatusInternalServerError, "error", errMsg)
		return
	}

	quizEvaluation, err := qes.internalServer.GetQuizEvaluationForUser(r.Context(), req)
	if err != nil {
		if hferrors.IsGrpcNotFound(err) {
			// create
			quizEvaluationAttempt := NewPBQuizEvaluationAttempt(1, preparedCreateQuizEvaluation, quiz)
			createQuizEvaluation := &quizpb.CreateQuizEvaluationRequest{
				Quiz:     req.GetQuiz(),
				QuizUid:  quiz.Uid,
				User:     req.GetUser(),
				Scenario: req.GetScenario(),
				Attempts: []*quizpb.QuizEvaluationAttempt{quizEvaluationAttempt},
			}
			quizEvaluationId, err := qes.internalServer.CreateQuizEvaluation(r.Context(), createQuizEvaluation)
			if err != nil {
				statusErr := status.Convert(err)
				if hferrors.IsGrpcParsingError(err) {
					glog.Errorf("error while parsing: %s", statusErr.Message())
					util.ReturnHTTPMessage(w, r, 500, "internalerror", "error parsing")
					return
				}
				glog.Errorf("error creating quiz evaluation %s", hferrors.GetErrorMessage(err))
				util.ReturnHTTPMessage(w, r, 500, "internalerror", "error creating quiz evaluation")
				return
			}

			preparedQuizEvaluationAttempt := NewPreparedAttempt(quiz.ValidationType, quizEvaluationAttempt)
			encodedQuizEvaluationAttempt, err := json.Marshal(preparedQuizEvaluationAttempt)
			if err != nil {
				glog.Errorf("error marshalling prepared quiz evaluation: %v", err)
				util.ReturnHTTPMessage(w, r, 500, "internalerror", "error creating quiz evaluation")
				return
			}

			util.ReturnHTTPContent(w, r, 201, "created", encodedQuizEvaluationAttempt)
			glog.V(4).Infof("Created quiz evaluation %s", quizEvaluationId.GetId())
			return
		}
		errMsg := "error retrieving quiz evaluation while attempting quiz evaluation creation"
		util.ReturnHTTPMessage(w, r, http.StatusInternalServerError, "error retrieving quiz evaluation for create", errMsg)
		return
	}

	// update
	attempt := uint32(len(quizEvaluation.GetAttempts())) + 1
	quizEvaluationAttempt := NewPBQuizEvaluationAttempt(attempt, preparedCreateQuizEvaluation, quiz)
	updateQuizEvaluation := &quizpb.UpdateQuizEvaluationRequest{
		Id:       quizEvaluation.GetId(),
		Quiz:     req.GetQuiz(),
		User:     req.GetUser(),
		Scenario: req.GetScenario(),
		Attempts: append(quizEvaluation.GetAttempts(), quizEvaluationAttempt),
	}
	_, err = qes.internalServer.UpdateQuizEvaluation(r.Context(), updateQuizEvaluation)
	if err != nil {
		glog.Error(hferrors.GetErrorMessage(err))
		util.ReturnHTTPMessage(w, r, 500, "internalerror", "error creating quiz evaluation")
		return
	}

	preparedQuizEvaluationAttempt := NewPreparedAttempt(quiz.ValidationType, quizEvaluationAttempt)
	encodedQuizEvaluationAttempt, err := json.Marshal(preparedQuizEvaluationAttempt)
	if err != nil {
		glog.Errorf("error marshalling prepared quiz evaluation: %v", err)
		util.ReturnHTTPMessage(w, r, 500, "internalerror", "error creating quiz evaluation")
		return
	}

	util.ReturnHTTPContent(w, r, 201, "created", encodedQuizEvaluationAttempt)
	glog.V(4).Infof("Created quiz evaluation %s", quizEvaluation.GetId())
}

func (qes QuizEvaluationService) getQuiz(ctx context.Context, quizId string) (*quizpb.Quiz, error) {
	return qes.internalQuizServer.GetQuiz(ctx, &generalpb.GetRequest{Id: quizId})
}

func (qes QuizEvaluationService) DeleteFunc(w http.ResponseWriter, r *http.Request) {
	user, err := rbac.AuthenticateRequest(r, qes.authnClient)
	if err != nil {
		util.ReturnHTTPMessage(w, r, 401, "unauthorized", "authentication failed")
		return
	}

	impersonatedUserId := user.GetId()
	authrResponse, err := rbac.AuthorizeSimple(r, qes.authrClient, impersonatedUserId, rbac.HobbyfarmPermission(rbac.ResourcePluralQuizEvaluation, rbac.VerbDelete))
	if err != nil || !authrResponse.Success {
		util.ReturnHTTPMessage(w, r, 403, "forbidden", "no access to delete quiz evaluation")
		return
	}

	vars := mux.Vars(r)
	quizEvaluationId := vars["id"]
	if quizEvaluationId == "" {
		util.ReturnHTTPMessage(w, r, 400, "badrequest", "no quiz evaluation id passed in")
		return
	}

	glog.V(2).Infof("user %s deleting quiz evaluation %s", user.GetId(), quizEvaluationId)

	// first check if the quiz evaluation actually exists
	_, err = qes.internalServer.GetQuizEvaluation(r.Context(), &generalpb.GetRequest{Id: quizEvaluationId})
	if err != nil {
		glog.Errorf("error while retrieving quiz evaluation: %s", hferrors.GetErrorMessage(err))
		if hferrors.IsGrpcNotFound(err) {
			errMsg := fmt.Sprintf("error retrieving quiz evaluation while attempting quiz evaluation deletion: quiz evaluation %s not found", quizEvaluationId)
			util.ReturnHTTPMessage(w, r, http.StatusNotFound, "not found", errMsg)
			return
		}
		errMsg := fmt.Sprintf("error retrieving quiz evaluation %s while attempting quiz evaluation deletion", quizEvaluationId)
		util.ReturnHTTPMessage(w, r, http.StatusInternalServerError, "error", errMsg)
		return
	}

	_, err = qes.internalServer.DeleteQuizEvaluation(r.Context(), &generalpb.ResourceId{Id: quizEvaluationId})
	if err != nil {
		glog.Errorf("error deleting quiz evaluation: %s", hferrors.GetErrorMessage(err))
		util.ReturnHTTPMessage(w, r, 500, "internalerror", "error deleting quiz evaluation")
		return
	}

	util.ReturnHTTPMessage(w, r, 200, "deleted", "quiz evaluation deleted")
	glog.V(4).Infof("deleted quiz evaluation: %s", quizEvaluationId)
}
