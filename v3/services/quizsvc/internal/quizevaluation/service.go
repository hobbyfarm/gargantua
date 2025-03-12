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

	preparedQuizEvaluation := NewPreparedQuizEvaluation(quiz.ValidationTypeDetailed, quizEvaluation, true)
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

	existing, err := qes.getQuiz(r.Context(), quizId)
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

	preparedQuizEvaluation := NewPreparedQuizEvaluation(existing.ValidationType, quizEvaluation, false)
	encodedQuiz, err := json.Marshal(preparedQuizEvaluation)
	if err != nil {
		glog.Error(err)
	}
	util.ReturnHTTPContent(w, r, 200, "success", encodedQuiz)

	glog.V(2).Infof("retrieved quiz evaluation %s", quizId)
}

func (qes QuizEvaluationService) StartFunc(w http.ResponseWriter, r *http.Request) {
	user, err := rbac.AuthenticateRequest(r, qes.authnClient)
	if err != nil {
		util.ReturnHTTPMessage(w, r, 401, "unauthorized", "authentication failed")
		return
	}

	impersonatedUserId := user.GetId()
	authrResponse, err := rbac.AuthorizeSimple(r, qes.authrClient, impersonatedUserId, rbac.HobbyfarmPermission(rbac.ResourcePluralQuizEvaluation, rbac.VerbCreate))
	if err != nil || !authrResponse.Success {
		util.ReturnHTTPMessage(w, r, 403, "forbidden", "no access to start quiz evaluation")
		return
	}

	var preparedStartQuizEvaluation PreparedStartQuizEvaluation

	// Decode JSON body
	if err = json.NewDecoder(r.Body).Decode(&preparedStartQuizEvaluation); err != nil {
		util.ReturnHTTPMessage(w, r, 400, "badrequest", "invalid json body")
		return
	}

	req := &quizpb.GetQuizEvaluationForUserRequest{
		Quiz:     preparedStartQuizEvaluation.Quiz,
		User:     impersonatedUserId,
		Scenario: preparedStartQuizEvaluation.Scenario,
	}

	existing, err := qes.getQuiz(r.Context(), req.GetQuiz())
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
			quizEvaluationAttempt := NewPBQuizEvaluationAttemptForStart(1, existing)
			createQuizEvaluation := &quizpb.CreateQuizEvaluationRequest{
				Quiz:     req.GetQuiz(),
				QuizUid:  existing.Uid,
				User:     req.GetUser(),
				Scenario: req.GetScenario(),
				Attempt:  quizEvaluationAttempt,
			}
			quizEvaluationId, err := qes.internalServer.CreateQuizEvaluation(r.Context(), createQuizEvaluation)
			if err != nil {
				statusErr := status.Convert(err)
				if hferrors.IsGrpcParsingError(err) {
					glog.Errorf("error while parsing: %s", statusErr.Message())
					util.ReturnHTTPMessage(w, r, 500, "internalerror", "error parsing")
					return
				}
				glog.Errorf("error starting quiz evaluation %s", hferrors.GetErrorMessage(err))
				util.ReturnHTTPMessage(w, r, 500, "internalerror", "error starting quiz evaluation")
				return
			}

			preparedStartQuizEvaluationResult := NewPreparedStartQuizEvaluationResult(
				quizEvaluationId.GetId(), req.GetQuiz(), req.GetScenario(), quizEvaluationAttempt)
			encodedStartQuizEvaluationResult, err := json.Marshal(preparedStartQuizEvaluationResult)
			if err != nil {
				glog.Errorf("error marshalling prepared quiz evaluation: %v", err)
				util.ReturnHTTPMessage(w, r, 500, "internalerror", "error starting quiz evaluation")
				return
			}

			util.ReturnHTTPContent(w, r, 201, "created", encodedStartQuizEvaluationResult)
			glog.V(4).Infof("Started quiz evaluation %s", quizEvaluationId.GetId())
			return
		}
		errMsg := "error retrieving quiz evaluation while attempting quiz evaluation start"
		util.ReturnHTTPMessage(w, r, http.StatusInternalServerError, "error retrieving quiz evaluation for start", errMsg)
		return
	}

	// update
	attempt := uint32(len(quizEvaluation.GetAttempts())) + 1
	quizEvaluationAttempt := NewPBQuizEvaluationAttemptForStart(attempt, existing)
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
		util.ReturnHTTPMessage(w, r, 500, "internalerror", "error starting quiz evaluation")
		return
	}

	preparedStartQuizEvaluationResult := NewPreparedStartQuizEvaluationResult(
		quizEvaluation.GetId(), req.GetQuiz(), req.GetScenario(), quizEvaluationAttempt)
	encodedStartQuizEvaluationResult, err := json.Marshal(preparedStartQuizEvaluationResult)
	if err != nil {
		glog.Errorf("error marshalling prepared quiz evaluation: %v", err)
		util.ReturnHTTPMessage(w, r, 500, "internalerror", "error starting quiz evaluation")
		return
	}

	util.ReturnHTTPContent(w, r, 201, "created", encodedStartQuizEvaluationResult)
	glog.V(4).Infof("Started quiz evaluation %s", quizEvaluation.GetId())
}

func (qes QuizEvaluationService) RecordFunc(w http.ResponseWriter, r *http.Request) {
	user, err := rbac.AuthenticateRequest(r, qes.authnClient)
	if err != nil {
		util.ReturnHTTPMessage(w, r, 401, "unauthorized", "authentication failed")
		return
	}

	impersonatedUserId := user.GetId()
	authrResponse, err := rbac.AuthorizeSimple(r, qes.authrClient, impersonatedUserId, rbac.HobbyfarmPermission(rbac.ResourcePluralQuizEvaluation, rbac.VerbCreate))
	if err != nil || !authrResponse.Success {
		util.ReturnHTTPMessage(w, r, 403, "forbidden", "no access to record quiz evaluation")
		return
	}

	var preparedRecordQuizEvaluation PreparedRecordQuizEvaluation

	// Decode JSON body
	if err = json.NewDecoder(r.Body).Decode(&preparedRecordQuizEvaluation); err != nil {
		util.ReturnHTTPMessage(w, r, 400, "badrequest", "invalid json body")
		return
	}

	req := &quizpb.GetQuizEvaluationForUserRequest{
		Quiz:     preparedRecordQuizEvaluation.Quiz,
		User:     impersonatedUserId,
		Scenario: preparedRecordQuizEvaluation.Scenario,
	}

	existing, err := qes.getQuiz(r.Context(), req.GetQuiz())
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
		glog.Errorf("error while retrieving quiz evaluation for quiz: %s", hferrors.GetErrorMessage(err))
		if hferrors.IsGrpcNotFound(err) {
			errMsg := fmt.Sprintf("quiz evaluation for quiz %s not found", req.GetQuiz())
			util.ReturnHTTPMessage(w, r, http.StatusNotFound, "not found", errMsg)
			return
		}
		errMsg := fmt.Sprintf("error retrieving quiz evaluation for quiz %s", req.GetQuiz())
		util.ReturnHTTPMessage(w, r, http.StatusInternalServerError, "error", errMsg)
		return
	}

	attempt := uint32(len(quizEvaluation.GetAttempts()))
	quizEvaluationAttempt := NewPBQuizEvaluationAttemptForRecord(attempt, preparedRecordQuizEvaluation, existing)

	attempts := make([]*quizpb.QuizEvaluationAttempt, len(quizEvaluation.GetAttempts()))
	for i := 0; i < len(quizEvaluation.GetAttempts()); i++ {
		currentQuizAttempt := quizEvaluation.GetAttempts()[i]
		if currentQuizAttempt.GetAttempt() == attempt {
			quizEvaluationAttempt.CreationTimestamp = currentQuizAttempt.GetCreationTimestamp()
			attempts[i] = quizEvaluationAttempt
			continue
		}
		attempts[i] = currentQuizAttempt
	}

	updateQuizEvaluation := &quizpb.UpdateQuizEvaluationRequest{
		Id:       quizEvaluation.GetId(),
		Quiz:     req.GetQuiz(),
		User:     req.GetUser(),
		Scenario: req.GetScenario(),
		Attempts: attempts,
	}
	_, err = qes.internalServer.UpdateQuizEvaluation(r.Context(), updateQuizEvaluation)
	if err != nil {
		glog.Error(hferrors.GetErrorMessage(err))
		util.ReturnHTTPMessage(w, r, 500, "internalerror", "error recording quiz evaluation")
		return
	}

	preparedRecordQuizEvaluationResult := PreparedRecordQuizEvaluationResult{
		Id:       quizEvaluation.GetId(),
		Quiz:     req.GetQuiz(),
		Scenario: req.GetScenario(),
		Attempt:  NewPreparedAttempt(existing.ValidationType, quizEvaluationAttempt),
	}
	encodedRecordQuizEvaluationResult, err := json.Marshal(preparedRecordQuizEvaluationResult)
	if err != nil {
		glog.Errorf("error marshalling prepared quiz evaluation: %v", err)
		util.ReturnHTTPMessage(w, r, 500, "internalerror", "error recording quiz evaluation")
		return
	}

	util.ReturnHTTPContent(w, r, 201, "created", encodedRecordQuizEvaluationResult)
	glog.V(4).Infof("Recorded quiz evaluation %s", quizEvaluation.GetId())
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
