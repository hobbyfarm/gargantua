package quizevaluation

import (
	"fmt"
	"github.com/hobbyfarm/gargantua/services/quizsvc/v3/internal/quiz"
	generalpb "github.com/hobbyfarm/gargantua/v3/protos/general"
	quizpb "github.com/hobbyfarm/gargantua/v3/protos/quiz"
	"github.com/stretchr/testify/assert"
	"reflect"
	"testing"
)

func TestNewPBQuizEvaluationAttemptForRecord(t *testing.T) {
	quiz := &quizpb.Quiz{
		Id:               "quiz-id",
		SuccessThreshold: 50,
		ValidationType:   quiz.ValidationTypeDetailed,
		Questions: []*quizpb.QuizQuestion{
			{
				Id:     "question-1",
				Weight: 1,
				Answers: []*quizpb.QuizAnswer{
					{Id: "question-1-answer-1", Correct: true},
					{Id: "question-1-answer-2", Correct: false},
					{Id: "question-1-answer-3", Correct: true},
					{Id: "question-1-answer-4", Correct: false},
				},
			},
			{
				Id:     "question-2",
				Weight: 2,
				Answers: []*quizpb.QuizAnswer{
					{Id: "question-2-answer-1", Correct: false},
					{Id: "question-2-answer-2", Correct: true},
				},
			},
			{
				Id:     "question-3",
				Weight: 1,
				Answers: []*quizpb.QuizAnswer{
					{Id: "question-3-answer-1", Correct: true},
					{Id: "question-3-answer-2", Correct: false},
				},
			},
			{
				Id:     "question-4",
				Weight: 2,
				Answers: []*quizpb.QuizAnswer{
					{Id: "question-4-answer-1", Correct: true},
					{Id: "question-4-answer-2", Correct: false},
					{Id: "question-4-answer-3", Correct: true},
				},
			},
		},
	}

	tests := []struct {
		name       string
		evaluation PreparedRecordQuizEvaluation
		want       *quizpb.QuizEvaluationAttempt
	}{
		{
			name: "full score",
			evaluation: PreparedRecordQuizEvaluation{
				Quiz: quiz.GetId(),
				Answers: map[string][]string{
					"question-1": {"question-1-answer-1", "question-1-answer-3"}, // correct
					"question-2": {"question-2-answer-2"},                        // correct
					"question-3": {"question-3-answer-1"},                        // correct
					"question-4": {"question-4-answer-1", "question-4-answer-3"}, // correct
				},
			},
			want: &quizpb.QuizEvaluationAttempt{
				Score: 100,
				Pass:  true,
				Corrects: map[string]*generalpb.StringArray{
					"question-1": {Values: []string{"question-1-answer-1", "question-1-answer-3"}},
					"question-2": {Values: []string{"question-2-answer-2"}},
					"question-3": {Values: []string{"question-3-answer-1"}},
					"question-4": {Values: []string{"question-4-answer-1", "question-4-answer-3"}},
				},
				Selects: map[string]*generalpb.StringArray{
					"question-1": {Values: []string{"question-1-answer-1", "question-1-answer-3"}},
					"question-2": {Values: []string{"question-2-answer-2"}},
					"question-3": {Values: []string{"question-3-answer-1"}},
					"question-4": {Values: []string{"question-4-answer-1", "question-4-answer-3"}},
				},
			},
		},
		{
			name: "full score with one question",
			evaluation: PreparedRecordQuizEvaluation{
				Quiz: quiz.GetId(),
				Answers: map[string][]string{
					"question-1": {"question-1-answer-1", "question-1-answer-3"},
				},
			},
			want: &quizpb.QuizEvaluationAttempt{
				Score: 100,
				Pass:  true,
				Corrects: map[string]*generalpb.StringArray{
					"question-1": {Values: []string{"question-1-answer-1", "question-1-answer-3"}},
				},
				Selects: map[string]*generalpb.StringArray{
					"question-1": {Values: []string{"question-1-answer-1", "question-1-answer-3"}},
				},
			},
		},
		{
			name: "zero score",
			evaluation: PreparedRecordQuizEvaluation{
				Quiz: quiz.GetId(),
				Answers: map[string][]string{
					"question-1": {"question-1-answer-2", "question-1-answer-4"}, // false
					"question-2": {"question-2-answer-1"},                        // false
					"question-3": {"question-3-answer-2"},                        // false
					"question-4": {"question-4-answer-2"},                        // false
				},
			},
			want: &quizpb.QuizEvaluationAttempt{
				Score: 0,
				Pass:  false,
				Corrects: map[string]*generalpb.StringArray{
					"question-1": {Values: []string{"question-1-answer-1", "question-1-answer-3"}},
					"question-2": {Values: []string{"question-2-answer-2"}},
					"question-3": {Values: []string{"question-3-answer-1"}},
					"question-4": {Values: []string{"question-4-answer-1", "question-4-answer-3"}},
				},
				Selects: map[string]*generalpb.StringArray{
					"question-1": {Values: []string{"question-1-answer-2", "question-1-answer-4"}},
					"question-2": {Values: []string{"question-2-answer-1"}},
					"question-3": {Values: []string{"question-3-answer-2"}},
					"question-4": {Values: []string{"question-4-answer-2"}},
				},
			},
		},
		{
			name: "score 50 some",
			evaluation: PreparedRecordQuizEvaluation{
				Quiz: quiz.GetId(),
				Answers: map[string][]string{
					"question-2": {"question-2-answer-1"},                        // false
					"question-4": {"question-4-answer-1", "question-4-answer-3"}, // correct
				},
			},
			want: &quizpb.QuizEvaluationAttempt{
				Score: 50,
				Pass:  true,
				Corrects: map[string]*generalpb.StringArray{
					"question-2": {Values: []string{"question-2-answer-2"}},
					"question-4": {Values: []string{"question-4-answer-1", "question-4-answer-3"}},
				},
				Selects: map[string]*generalpb.StringArray{
					"question-2": {Values: []string{"question-2-answer-1"}},
					"question-4": {Values: []string{"question-4-answer-1", "question-4-answer-3"}},
				},
			},
		},
		{
			name: "score 50 all",
			evaluation: PreparedRecordQuizEvaluation{
				Quiz: quiz.GetId(),
				Answers: map[string][]string{
					"question-1": {"question-1-answer-1", "question-1-answer-3"}, // correct
					"question-2": {"question-2-answer-1"},                        // false
					"question-3": {"question-3-answer-2"},                        // false
					"question-4": {"question-4-answer-1", "question-4-answer-3"}, // correct
				},
			},
			want: &quizpb.QuizEvaluationAttempt{
				Score: 50,
				Pass:  true,
				Corrects: map[string]*generalpb.StringArray{
					"question-1": {Values: []string{"question-1-answer-1", "question-1-answer-3"}},
					"question-2": {Values: []string{"question-2-answer-2"}},
					"question-3": {Values: []string{"question-3-answer-1"}},
					"question-4": {Values: []string{"question-4-answer-1", "question-4-answer-3"}},
				},
				Selects: map[string]*generalpb.StringArray{
					"question-1": {Values: []string{"question-1-answer-1", "question-1-answer-3"}},
					"question-2": {Values: []string{"question-2-answer-1"}},
					"question-3": {Values: []string{"question-3-answer-2"}},
					"question-4": {Values: []string{"question-4-answer-1", "question-4-answer-3"}},
				},
			},
		},
		{
			name: "score 33 all",
			evaluation: PreparedRecordQuizEvaluation{
				Quiz: quiz.GetId(),
				Answers: map[string][]string{
					"question-1": {"question-1-answer-1", "question-1-answer-3"}, // correct
					"question-2": {"question-2-answer-1"},                        // false
					"question-3": {"question-3-answer-1"},                        // correct
					"question-4": {"question-4-answer-2"},                        // false
				},
			},
			want: &quizpb.QuizEvaluationAttempt{
				Score: 33,
				Pass:  false,
				Corrects: map[string]*generalpb.StringArray{
					"question-1": {Values: []string{"question-1-answer-1", "question-1-answer-3"}},
					"question-2": {Values: []string{"question-2-answer-2"}},
					"question-3": {Values: []string{"question-3-answer-1"}},
					"question-4": {Values: []string{"question-4-answer-1", "question-4-answer-3"}},
				},
				Selects: map[string]*generalpb.StringArray{
					"question-1": {Values: []string{"question-1-answer-1", "question-1-answer-3"}},
					"question-2": {Values: []string{"question-2-answer-1"}},
					"question-3": {Values: []string{"question-3-answer-1"}},
					"question-4": {Values: []string{"question-4-answer-2"}},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actual := NewPBQuizEvaluationAttemptForRecord(1, tt.evaluation, quiz)
			assert.Equal(t, tt.want.Score, actual.Score, fmt.Sprintf("Score(%d)", tt.want.Score))
			assert.Equal(t, tt.want.Pass, actual.Pass, fmt.Sprintf("Pass(%t)", tt.want.Pass))

			if eq := reflect.DeepEqual(tt.want.GetCorrects(), actual.GetCorrects()); !eq {
				t.Errorf("corrects got %v, want %v", actual.GetCorrects(), tt.want.GetCorrects())
			}

			if eq := reflect.DeepEqual(tt.want.GetSelects(), actual.GetSelects()); !eq {
				t.Errorf("selects got %v, want %v", actual.GetSelects(), tt.want.GetSelects())
			}
		})
	}
}

func TestNewPreparedAttempt(t *testing.T) {
	tests := []struct {
		name           string
		validationType quiz.ValidationType
		attempt        *quizpb.QuizEvaluationAttempt
		want           PreparedAttempt
	}{
		{
			name:           "ValidationTypeDetailed",
			validationType: quiz.ValidationTypeDetailed,
			attempt: &quizpb.QuizEvaluationAttempt{
				Corrects: map[string]*generalpb.StringArray{
					"question-1": {Values: []string{"question-1-answer-1", "question-1-answer-3"}},
					"question-2": {Values: []string{"question-2-answer-2"}},
					"question-3": {Values: []string{"question-3-answer-1"}},
					"question-4": {Values: []string{"question-4-answer-1", "question-4-answer-3"}},
				},
				Selects: map[string]*generalpb.StringArray{
					"question-1": {Values: []string{"question-1-answer-1", "question-1-answer-3"}}, // correct
					"question-2": {Values: []string{"question-2-answer-1"}},                        // false
					"question-3": {Values: []string{"question-3-answer-2"}},                        // false
					"question-4": {Values: []string{"question-4-answer-1", "question-4-answer-3"}}, // correct
				},
			},
			want: PreparedAttempt{
				Corrects: map[string][]string{
					"question-1": {"question-1-answer-1", "question-1-answer-3"},
					"question-2": {"question-2-answer-2"},
					"question-3": {"question-3-answer-1"},
					"question-4": {"question-4-answer-1", "question-4-answer-3"},
				},
				Selects: map[string][]string{
					"question-1": {"question-1-answer-1", "question-1-answer-3"}, // correct
					"question-2": {"question-2-answer-1"},                        // false
					"question-3": {"question-3-answer-2"},                        // false
					"question-4": {"question-4-answer-1", "question-4-answer-3"}, // correct
				},
			},
		},
		{
			name:           "ValidationTypeStandard",
			validationType: quiz.ValidationTypeStandard,
			attempt: &quizpb.QuizEvaluationAttempt{
				Corrects: map[string]*generalpb.StringArray{
					"question-1": {Values: []string{"question-1-answer-1", "question-1-answer-3"}},
					"question-2": {Values: []string{"question-2-answer-2"}},
					"question-3": {Values: []string{"question-3-answer-1"}},
					"question-4": {Values: []string{"question-4-answer-1", "question-4-answer-3"}},
				},
				Selects: map[string]*generalpb.StringArray{
					"question-1": {Values: []string{"question-1-answer-1", "question-1-answer-3"}}, // correct
					"question-2": {Values: []string{"question-2-answer-1"}},                        // false
					"question-3": {Values: []string{"question-3-answer-2"}},                        // false
					"question-4": {Values: []string{"question-4-answer-1", "question-4-answer-3"}}, // correct
				},
			},
			want: PreparedAttempt{
				Corrects: map[string][]string{
					"question-1": {"question-1-answer-1", "question-1-answer-3"},
					"question-4": {"question-4-answer-1", "question-4-answer-3"},
				},
				Selects: map[string][]string{
					"question-1": {"question-1-answer-1", "question-1-answer-3"}, // correct
					"question-2": {"question-2-answer-1"},                        // false
					"question-3": {"question-3-answer-2"},                        // false
					"question-4": {"question-4-answer-1", "question-4-answer-3"}, // correct
				},
			},
		},
		{
			name:           "ValidationTypeNone",
			validationType: quiz.ValidationTypeNone,
			attempt: &quizpb.QuizEvaluationAttempt{
				Corrects: map[string]*generalpb.StringArray{
					"question-1": {Values: []string{"question-1-answer-1", "question-1-answer-3"}},
					"question-2": {Values: []string{"question-2-answer-2"}},
					"question-3": {Values: []string{"question-3-answer-1"}},
					"question-4": {Values: []string{"question-4-answer-1", "question-4-answer-3"}},
				},
				Selects: map[string]*generalpb.StringArray{
					"question-1": {Values: []string{"question-1-answer-1", "question-1-answer-3"}}, // correct
					"question-2": {Values: []string{"question-2-answer-1"}},                        // false
					"question-3": {Values: []string{"question-3-answer-2"}},                        // false
					"question-4": {Values: []string{"question-4-answer-1", "question-4-answer-3"}}, // correct
				},
			},
			want: PreparedAttempt{
				Corrects: map[string][]string{},
				Selects: map[string][]string{
					"question-1": {"question-1-answer-1", "question-1-answer-3"}, // correct
					"question-2": {"question-2-answer-1"},                        // false
					"question-3": {"question-3-answer-2"},                        // false
					"question-4": {"question-4-answer-1", "question-4-answer-3"}, // correct
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equalf(t, tt.want, NewPreparedAttempt(tt.validationType, tt.attempt), "NewPreparedAttempt(%v, %v)", tt.validationType, tt.attempt)
		})
	}
}
