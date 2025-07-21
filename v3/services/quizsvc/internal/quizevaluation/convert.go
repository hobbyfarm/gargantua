package quizevaluation

import (
	"github.com/hobbyfarm/gargantua/services/quizsvc/v3/internal/quiz"
	hfv1 "github.com/hobbyfarm/gargantua/v3/pkg/apis/hobbyfarm.io/v1"
	"github.com/hobbyfarm/gargantua/v3/pkg/util"
	generalpb "github.com/hobbyfarm/gargantua/v3/protos/general"
	quizpb "github.com/hobbyfarm/gargantua/v3/protos/quiz"
	"math/rand"
	"reflect"
	"time"
)

func NewPreparedQuizEvaluation(validationType quiz.ValidationType, quizEvaluation *quizpb.QuizEvaluation, showUser bool) PreparedQuizEvaluation {
	attempts := make([]PreparedAttempt, len(quizEvaluation.GetAttempts()))
	for i, attempt := range quizEvaluation.GetAttempts() {
		corrects := make(map[string][]string)
		for questionId, answerIds := range attempt.GetCorrects() {
			corrects[questionId] = answerIds.GetValues()
		}

		selects := make(map[string][]string)
		for questionId, answerIds := range attempt.GetSelects() {
			selects[questionId] = answerIds.GetValues()
		}

		attempts[i] = NewPreparedAttempt(validationType, attempt)
	}

	var user string
	if showUser {
		user = quizEvaluation.GetUser()
	}

	return PreparedQuizEvaluation{
		Id:       quizEvaluation.GetId(),
		Quiz:     quizEvaluation.GetQuiz(),
		User:     user,
		Scenario: quizEvaluation.GetScenario(),
		Attempts: attempts,
	}
}

func NewPBQuizEvaluationAttemptForStart(attempt uint32, quiz *quizpb.Quiz) *quizpb.QuizEvaluationAttempt {
	var selects map[string]*generalpb.StringArray
	if quiz.GetShuffle() {
		selects = randomSelects(quiz)
	} else {
		selects = orderedSelects(quiz)
	}

	return &quizpb.QuizEvaluationAttempt{
		CreationTimestamp: time.Now().Format(time.UnixDate),
		Attempt:           attempt,
		Selects:           selects,
	}
}

func randomSelects(quiz *quizpb.Quiz) map[string]*generalpb.StringArray {
	selects := make(map[string]*generalpb.StringArray)

	rng := rand.New(rand.NewSource(time.Now().UnixNano()))
	for {
		randomIndex := rng.Intn(len(quiz.GetQuestions()))
		questionId := quiz.GetQuestions()[randomIndex].GetId()

		if _, exists := selects[questionId]; exists {
			continue
		}
		selects[questionId] = &generalpb.StringArray{Values: []string{}}

		if len(selects) == int(quiz.PoolSize) {
			break
		}
	}
	return selects
}

func orderedSelects(quiz *quizpb.Quiz) map[string]*generalpb.StringArray {
	selects := make(map[string]*generalpb.StringArray)

	poolSize := int(quiz.PoolSize)
	for i := 0; i < poolSize; i++ {
		questionId := quiz.GetQuestions()[i].GetId()
		selects[questionId] = &generalpb.StringArray{Values: []string{}}
	}

	return selects
}

func NewPBQuizEvaluationAttemptForRecord(attempt uint32, evaluation PreparedRecordQuizEvaluation, quiz *quizpb.Quiz) *quizpb.QuizEvaluationAttempt {
	var achievable uint32
	var actual uint32
	selects := make(map[string]*generalpb.StringArray)
	corrects := make(map[string]*generalpb.StringArray)

	for _, question := range quiz.GetQuestions() {
		if selectedAnswers, exists := evaluation.Answers[question.GetId()]; exists {
			achievable += question.GetWeight()

			selects[question.GetId()] = &generalpb.StringArray{Values: selectedAnswers}
			selectedAnswerIds := make(map[string]struct{})
			for _, selectedAnswerId := range selectedAnswers {
				selectedAnswerIds[selectedAnswerId] = struct{}{}
			}

			correctAnswers := make([]string, 0)
			for _, answer := range question.GetAnswers() {
				if answer.GetCorrect() {
					correctAnswers = append(correctAnswers, answer.GetId())
				}
			}
			corrects[question.GetId()] = &generalpb.StringArray{Values: correctAnswers}

			correctAnswerIds := make(map[string]struct{})
			for _, correctAnswerId := range correctAnswers {
				correctAnswerIds[correctAnswerId] = struct{}{}
			}

			if reflect.DeepEqual(selectedAnswerIds, correctAnswerIds) {
				actual += question.GetWeight()
			}
		}
	}

	// (actual / achievable) * 100
	achievedPercent := (float32(actual) / float32(achievable)) * float32(100)
	passed := achievedPercent >= float32(quiz.SuccessThreshold)

	return &quizpb.QuizEvaluationAttempt{
		Timestamp: time.Now().Format(time.UnixDate),
		Attempt:   attempt,
		Score:     uint32(achievedPercent), // always round down
		Pass:      passed,
		Corrects:  corrects,
		Selects:   selects,
	}
}

func NewPreparedStartQuizEvaluationResult(id, quiz, scenario string, attempt *quizpb.QuizEvaluationAttempt) PreparedStartQuizEvaluationResult {
	questions := make([]string, 0, len(attempt.GetSelects()))
	for questionId, _ := range attempt.GetSelects() {
		questions = append(questions, questionId)
	}
	return PreparedStartQuizEvaluationResult{
		Id:                id,
		Quiz:              quiz,
		Scenario:          scenario,
		CreationTimestamp: attempt.GetCreationTimestamp(),
		Attempt:           attempt.GetAttempt(),
		Questions:         questions,
	}
}

func NewPreparedAttempt(validationType quiz.ValidationType, attempt *quizpb.QuizEvaluationAttempt) PreparedAttempt {
	selects := make(map[string][]string)
	for questionId, answerId := range attempt.GetSelects() {
		selects[questionId] = answerId.GetValues()
	}

	corrects := make(map[string][]string)
	switch validationType {
	case quiz.ValidationTypeStandard:
		// only show answers which have been correctly answered
		allCorrects := make(map[string][]string)
		for questionId, answerId := range attempt.GetCorrects() {
			allCorrects[questionId] = answerId.GetValues()
		}

		for questionId, selectedAnswerIDs := range selects {
			if _, exists := allCorrects[questionId]; exists {
				if util.SliceSortEqual(selectedAnswerIDs, allCorrects[questionId]) {
					corrects[questionId] = selectedAnswerIDs
				}
			}
		}
	case quiz.ValidationTypeDetailed:
		// show all correct answer even if they have been falsely answered
		for questionId, answerId := range attempt.GetCorrects() {
			corrects[questionId] = answerId.GetValues()
		}
	}

	return PreparedAttempt{
		CreationTimestamp: attempt.GetCreationTimestamp(),
		Timestamp:         attempt.GetTimestamp(),
		Attempt:           attempt.GetAttempt(),
		Score:             attempt.GetScore(),
		Pass:              attempt.GetPass(),
		Corrects:          corrects,
		Selects:           selects,
	}
}

func NewPBQuizEvaluationList(quizEvaluations []hfv1.QuizEvaluation) []*quizpb.QuizEvaluation {
	preparedQuizEvaluations := make([]*quizpb.QuizEvaluation, len(quizEvaluations))
	for i, eval := range quizEvaluations {
		preparedQuizEvaluations[i] = NewPBQuizEvaluation(&eval)
	}
	return preparedQuizEvaluations
}

func NewPBQuizEvaluation(quizEvaluations *hfv1.QuizEvaluation) *quizpb.QuizEvaluation {
	attempts := make([]*quizpb.QuizEvaluationAttempt, len(quizEvaluations.Spec.Attempts))

	for i, attempt := range quizEvaluations.Spec.Attempts {
		corrects := make(map[string]*generalpb.StringArray)
		for questionId, answerIds := range attempt.Corrects {
			corrects[questionId] = &generalpb.StringArray{Values: answerIds}
		}

		selects := make(map[string]*generalpb.StringArray)
		for questionId, answerIds := range attempt.Selects {
			selects[questionId] = &generalpb.StringArray{Values: answerIds}
		}

		attempts[i] = &quizpb.QuizEvaluationAttempt{
			CreationTimestamp: attempt.CreationTimestamp,
			Timestamp:         attempt.Timestamp,
			Attempt:           attempt.Attempt,
			Score:             attempt.Score,
			Pass:              attempt.Pass,
			Corrects:          corrects,
			Selects:           selects,
		}
	}

	return &quizpb.QuizEvaluation{
		Id:       quizEvaluations.Name,
		Uid:      string(quizEvaluations.UID),
		Quiz:     quizEvaluations.Spec.Quiz,
		User:     quizEvaluations.Spec.User,
		Scenario: quizEvaluations.Spec.Scenario,
		Attempts: attempts,
	}
}
