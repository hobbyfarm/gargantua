package quiz

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/golang/glog"
	"github.com/gorilla/mux"
	hferrors "github.com/hobbyfarm/gargantua/v3/pkg/errors"
	"github.com/hobbyfarm/gargantua/v3/pkg/rbac"
	"github.com/hobbyfarm/gargantua/v3/pkg/util"
	authnpb "github.com/hobbyfarm/gargantua/v3/protos/authn"
	authrpb "github.com/hobbyfarm/gargantua/v3/protos/authr"
	generalpb "github.com/hobbyfarm/gargantua/v3/protos/general"
	"google.golang.org/grpc/status"
	"net/http"
)

type QuizService struct {
	authnClient    authnpb.AuthNClient
	authrClient    authrpb.AuthRClient
	internalServer *GrpcQuizServer
}

func NewQuizService(
	authnClient authnpb.AuthNClient,
	authrClient authrpb.AuthRClient,
	internalQuizServer *GrpcQuizServer,
) *QuizService {
	return &QuizService{
		authnClient:    authnClient,
		authrClient:    authrClient,
		internalServer: internalQuizServer,
	}
}

func (qs QuizService) GetFunc(w http.ResponseWriter, r *http.Request) {
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

	quiz, err := qs.internalServer.GetQuiz(r.Context(), &generalpb.GetRequest{Id: quizId})
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

func (qs QuizService) GetForUserFunc(w http.ResponseWriter, r *http.Request) {
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

	quiz, err := qs.internalServer.GetQuiz(r.Context(), &generalpb.GetRequest{Id: quizId})
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

func (qs QuizService) ListFunc(w http.ResponseWriter, r *http.Request) {
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

	quizList, err := qs.internalServer.ListQuiz(r.Context(), &generalpb.ListOptions{})
	if err != nil {
		glog.Errorf("error while listing all quizzes: %s", hferrors.GetErrorMessage(err))
		util.ReturnHTTPMessage(w, r, 500, "error", "error listing all quizzes")
		return
	}

	preparedQuizzes := NewPreparedQuizList(quizList.Quizzes)

	encodedQuizzes, err := json.Marshal(preparedQuizzes)
	if err != nil {
		glog.Error(err)
	}
	util.ReturnHTTPContent(w, r, 200, "success", encodedQuizzes)

	glog.V(2).Infof("retrieved list of all quizzes")
}

func (qs QuizService) CreateFunc(w http.ResponseWriter, r *http.Request) {
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

	if err = validatePreparedQuiz(preparedQuiz); err != nil {
		util.ReturnHTTPMessage(w, r, 400, "badrequest", err.Error())
		return
	}

	quiz := NewPBCreateQuiz(preparedQuiz)
	quizId, err := qs.internalServer.CreateQuiz(r.Context(), quiz)

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

func validatePreparedQuiz(preparedQuiz PreparedQuiz) error {
	if preparedQuiz.PoolSize == 0 {
		return errors.New("pool size needs to be greater than 0")
	}

	if int(preparedQuiz.PoolSize) > len(preparedQuiz.Questions) {
		return errors.New("pool size can not be greater than amount of questions")
	}

	if preparedQuiz.MaxAttempts == 0 {
		return errors.New("max attempts needs to be greater than 0")
	}

	if preparedQuiz.SuccessThreshold > 100 {
		return errors.New("success threshold needs to be between 0 and 100")
	}

	for _, question := range preparedQuiz.Questions {
		if question.Weight == 0 {
			return errors.New("question weight needs to be greater than 0")
		}
		var hasCorrect bool
		for _, answer := range question.Answers {
			if util.DerefOrDefault(answer.Correct) {
				hasCorrect = true
			}
		}
		if !hasCorrect {
			return errors.New("each question needs at least one correct answer")
		}
	}
	return nil
}

func (qs QuizService) UpdateFunc(w http.ResponseWriter, r *http.Request) {
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

	if err = validatePreparedQuiz(preparedQuiz); err != nil {
		util.ReturnHTTPMessage(w, r, 400, "badrequest", err.Error())
		return
	}

	quiz := NewPBUpdateQuiz(id, preparedQuiz)
	_, err = qs.internalServer.UpdateQuiz(r.Context(), quiz)

	if err != nil {
		glog.Error(hferrors.GetErrorMessage(err))
		util.ReturnHTTPMessage(w, r, 500, "internalerror", "error attempting to update")
		return
	}

	util.ReturnHTTPMessage(w, r, 200, "updated", "")
	glog.V(4).Infof("Updated quiz %s", id)
}

func (qs QuizService) DeleteFunc(w http.ResponseWriter, r *http.Request) {
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
		util.ReturnHTTPMessage(w, r, 400, "badrequest", "no quiz id passed in")
		return
	}

	glog.V(2).Infof("user %s deleting quiz %s", user.GetId(), quizId)

	// first check if the quiz actually exists
	_, err = qs.internalServer.GetQuiz(r.Context(), &generalpb.GetRequest{Id: quizId})
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

	_, err = qs.internalServer.DeleteQuiz(r.Context(), &generalpb.ResourceId{Id: quizId})
	if err != nil {
		glog.Errorf("error deleting quiz: %s", hferrors.GetErrorMessage(err))
		util.ReturnHTTPMessage(w, r, 500, "internalerror", "error deleting quiz")
		return
	}

	util.ReturnHTTPMessage(w, r, 200, "deleted", "quiz deleted")
	glog.V(4).Infof("deleted quiz: %s", quizId)
}
