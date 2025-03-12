package quizevaluation

type PreparedQuizEvaluation struct {
	Id       string            `json:"id"`
	Quiz     string            `json:"quiz"`           // the quiz id
	User     string            `json:"user,omitempty"` // the user id
	Scenario string            `json:"scenario"`       // the scenario id
	Attempts []PreparedAttempt `json:"attempts"`
}

type PreparedAttempt struct {
	CreationTimestamp string              `json:"creation_timestamp"`
	Timestamp         string              `json:"timestamp,omitempty"`
	Attempt           uint32              `json:"attempt"`
	Score             uint32              `json:"score"`
	Pass              bool                `json:"pass"`
	Corrects          map[string][]string `json:"corrects,omitempty"` // key is question id and values are correct answer ids
	Selects           map[string][]string `json:"selects"`            // key is question id and values are answer ids of the answers chosen by the user
}

type PreparedStartQuizEvaluationResult struct {
	Id                string   `json:"id"`
	Quiz              string   `json:"quiz"`     // the quiz id
	Scenario          string   `json:"scenario"` // the scenario id
	CreationTimestamp string   `json:"creation_timestamp"`
	Attempt           uint32   `json:"attempt"`
	Questions         []string `json:"questions"` // the selected question ids by the backend
}

type PreparedRecordQuizEvaluationResult struct {
	Id       string          `json:"id"`
	Quiz     string          `json:"quiz"`     // the quiz id
	Scenario string          `json:"scenario"` // the scenario id
	Attempt  PreparedAttempt `json:"attempt"`
}

type PreparedStartQuizEvaluation struct {
	Quiz     string `json:"quiz"`     // the quiz id
	Scenario string `json:"scenario"` // the scenario id
}

type PreparedRecordQuizEvaluation struct {
	Quiz     string              `json:"quiz"`     // the quiz id
	Scenario string              `json:"scenario"` // the scenario id
	Answers  map[string][]string `json:"answers"`  // key is question id and values are answer ids
}
