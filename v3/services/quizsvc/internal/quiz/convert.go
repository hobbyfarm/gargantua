package quiz

import (
	hfv1 "github.com/hobbyfarm/gargantua/v3/pkg/apis/hobbyfarm.io/v1"
	"github.com/hobbyfarm/gargantua/v3/pkg/util"
	quizpb "github.com/hobbyfarm/gargantua/v3/protos/quiz"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func NewPreparedQuizList(quizzes []*quizpb.Quiz) []PreparedQuiz {
	preparedQuizzes := make([]PreparedQuiz, len(quizzes))
	for i, quiz := range quizzes {
		preparedQuizzes[i] = NewPreparedQuiz(quiz, true)
	}
	return preparedQuizzes
}

func NewPreparedQuiz(quiz *quizpb.Quiz, showCorrect bool) PreparedQuiz {
	questions := make([]PreparedQuestion, len(quiz.GetQuestions()))
	for i, question := range quiz.GetQuestions() {
		questions[i] = NewPreparedQuestion(question, showCorrect)
	}
	return PreparedQuiz{
		Id:               quiz.GetId(),
		Title:            quiz.GetTitle(),
		Issuer:           quiz.GetIssuer(),
		Shuffle:          quiz.GetShuffle(),
		PoolSize:         quiz.GetPoolSize(),
		MaxAttempts:      quiz.GetMaxAttempts(),
		SuccessThreshold: quiz.GetSuccessThreshold(),
		ValidationType:   parseValidationType(quiz.GetValidationType()),
		Questions:        questions,
	}
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
		Weight:         question.GetWeight(),
		Answers:        answers,
	}
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

func NewPBCreateQuiz(quiz PreparedQuiz) *quizpb.CreateQuizRequest {
	questions := make([]*quizpb.CreateQuizQuestion, len(quiz.Questions))
	for i, question := range quiz.Questions {
		questions[i] = NewPBCreateQuizQuestion(question)
	}
	return &quizpb.CreateQuizRequest{
		Title:            quiz.Title,
		Issuer:           quiz.Issuer,
		Shuffle:          quiz.Shuffle,
		PoolSize:         quiz.PoolSize,
		MaxAttempts:      quiz.MaxAttempts,
		SuccessThreshold: quiz.SuccessThreshold,
		ValidationType:   parseValidationType(quiz.ValidationType),
		Questions:        questions,
	}
}

func NewPBCreateQuizQuestion(question PreparedQuestion) *quizpb.CreateQuizQuestion {
	answers := make([]*quizpb.CreateQuizAnswer, len(question.Answers))
	for j, answer := range question.Answers {
		answers[j] = NewPBCreateQuizAnswer(answer)
	}
	return &quizpb.CreateQuizQuestion{
		Title:          question.Title,
		Description:    question.Description,
		Type:           question.Type,
		Shuffle:        question.Shuffle,
		FailureMessage: question.FailureMessage,
		SuccessMessage: question.SuccessMessage,
		Weight:         question.Weight,
		Answers:        answers,
	}
}

func NewPBCreateQuizAnswer(answer PreparedAnswer) *quizpb.CreateQuizAnswer {
	return &quizpb.CreateQuizAnswer{
		Title:   answer.Title,
		Correct: util.DerefOrDefault[bool](answer.Correct),
	}
}

func NewPBUpdateQuiz(id string, quiz PreparedQuiz) *quizpb.UpdateQuizRequest {
	questions := make([]*quizpb.UpdateQuizQuestion, len(quiz.Questions))
	for i, question := range quiz.Questions {
		questions[i] = NewPBUpdateQuizQuestion(question)
	}
	return &quizpb.UpdateQuizRequest{
		Id:               id,
		Title:            quiz.Title,
		Issuer:           quiz.Issuer,
		Shuffle:          quiz.Shuffle,
		PoolSize:         quiz.PoolSize,
		MaxAttempts:      quiz.MaxAttempts,
		SuccessThreshold: quiz.SuccessThreshold,
		ValidationType:   parseValidationType(quiz.ValidationType),
		Questions:        questions,
	}
}

func NewPBUpdateQuizQuestion(question PreparedQuestion) *quizpb.UpdateQuizQuestion {
	answers := make([]*quizpb.UpdateQuizAnswer, len(question.Answers))
	for j, answer := range question.Answers {
		answers[j] = NewPBUpdateQuizAnswer(answer)
	}
	return &quizpb.UpdateQuizQuestion{
		Title:          question.Title,
		Description:    question.Description,
		Type:           question.Type,
		Shuffle:        question.Shuffle,
		FailureMessage: question.FailureMessage,
		SuccessMessage: question.SuccessMessage,
		Weight:         question.Weight,
		Answers:        answers,
	}
}

func NewPBUpdateQuizAnswer(answer PreparedAnswer) *quizpb.UpdateQuizAnswer {
	return &quizpb.UpdateQuizAnswer{
		Title:   answer.Title,
		Correct: util.DerefOrDefault[bool](answer.Correct),
	}
}

func NewPBQuizList(quizzes []hfv1.Quiz) []*quizpb.Quiz {
	preparedQuizzes := make([]*quizpb.Quiz, len(quizzes))
	for i, quiz := range quizzes {
		preparedQuizzes[i] = NewPBQuiz(&quiz)
	}
	return preparedQuizzes
}

func NewPBQuiz(quiz *hfv1.Quiz) *quizpb.Quiz {
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
			Weight:         question.Weight,
			Answers:        answers,
		}
	}

	return &quizpb.Quiz{
		Id:               quiz.Name,
		Uid:              string(quiz.UID),
		Title:            quiz.Spec.Title,
		Issuer:           quiz.Spec.Issuer,
		Shuffle:          quiz.Spec.Shuffle,
		PoolSize:         quiz.Spec.PoolSize,
		MaxAttempts:      quiz.Spec.MaxAttempts,
		SuccessThreshold: quiz.Spec.SuccessThreshold,
		ValidationType:   parseValidationType(quiz.Spec.ValidationType),
		Questions:        questions,
	}
}

func NewQuizFromCreate(id string, quiz *quizpb.CreateQuizRequest) *hfv1.Quiz {
	questions := make([]hfv1.QuizQuestion, len(quiz.GetQuestions()))
	for i, question := range quiz.GetQuestions() {
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
			Weight:         question.GetWeight(),
			Answers:        answers,
		}
	}

	return &hfv1.Quiz{
		ObjectMeta: metav1.ObjectMeta{
			Name: id,
		},
		Spec: hfv1.QuizSpec{
			Title:            quiz.GetTitle(),
			Issuer:           quiz.GetIssuer(),
			Shuffle:          quiz.GetShuffle(),
			PoolSize:         quiz.GetPoolSize(),
			MaxAttempts:      quiz.GetMaxAttempts(),
			SuccessThreshold: quiz.GetSuccessThreshold(),
			ValidationType:   parseValidationType(quiz.GetValidationType()),
			Questions:        questions,
		},
	}
}

func NewQuizFromUpdate(req *quizpb.UpdateQuizRequest, source *hfv1.Quiz) *hfv1.Quiz {
	source.Spec.Title = req.GetTitle()
	source.Spec.Issuer = req.GetIssuer()
	source.Spec.Shuffle = req.GetShuffle()
	source.Spec.PoolSize = req.GetPoolSize()
	source.Spec.MaxAttempts = req.GetMaxAttempts()
	source.Spec.SuccessThreshold = req.GetSuccessThreshold()
	source.Spec.ValidationType = parseValidationType(req.GetValidationType())

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
			Weight:         question.GetWeight(),
			Answers:        answers,
		}
	}

	source.Spec.Questions = questions

	return source
}
