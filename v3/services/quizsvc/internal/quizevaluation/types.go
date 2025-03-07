package quizevaluation

type PreparedQuizEvaluation struct {
	Id       string            `json:"id"`
	Quiz     string            `json:"quiz"`     // the quiz id
	User     string            `json:"user"`     // the user id
	Scenario string            `json:"scenario"` // the scenario id
	Attempts []PreparedAttempt `json:"attempts"`
}

type PreparedAttempt struct {
	Timestamp string              `json:"timestamp"`
	Attempt   uint32              `json:"attempt"`
	Score     uint32              `json:"score"`
	Pass      bool                `json:"pass"`
	Corrects  map[string][]string `json:"corrects"` // key is question id and values are correct answer ids
	Selects   map[string][]string `json:"selects"`  // key is question id and values are answer ids of the answers chosen by the user
}

type PreparedCreateQuizEvaluation struct {
	Quiz     string              `json:"quiz"`     // the quiz id
	Scenario string              `json:"scenario"` // the scenario id
	Answers  map[string][]string `json:"answers"`  // key is question id and values are answer ids
}
