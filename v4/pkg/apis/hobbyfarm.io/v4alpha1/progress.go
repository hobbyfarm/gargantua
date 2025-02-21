package v4alpha1

import metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// Progress holds the progress of a user through a particular scenario. It tracks items such as
// the step the user is currently on, when they started and ended, etc.
type Progress struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ProgressSpec   `json:"spec"`
	Status ProgressStatus `json:"status"`
}

type ProgressSpec struct {
	// User is the object name of the user to whom this Progress belongs
	User string `json:"user"`

	// Scenario is the object name of the scenario to which this Progress
	// relates.
	Scenario string `json:"scenario"`

	// Course is the object name of the course to which this Progress relates.
	Course string `json:"course"`

	// TotalStep is the total number of steps in the Scenario. It is copied into
	// this struct to make visualizations and reasoning about step progress easier.
	TotalStep int `json:"totalStep"`
}

type ProgressStatus struct {
	// CurrentStep is the 0-indexed step of the scenario that the user
	// is currently on, or was on as of LastUpdate.
	CurrentStep int `json:"currentStep"`

	// MaxStep is the 0-indexed *furthest* step the user has gotten to.
	// May not be the current step the user is on, if they have backtracked.
	MaxStep int `json:"maxStep"`

	// Started is the timestamp when the user started the scenario.
	Started metav1.Time `json:"started"`

	// LastUpdate is the timestamp when the user interface last updated this struct.
	LastUpdate metav1.Time `json:"lasteUpdate"`

	// Finished is the timestamp when the user finished the scenario.
	Finished metav1.Time `json:"finished"`

	// StepTimes stores the timestamps when a user entered a particular step.
	StepTimes []StepTime `json:"stepTimes"`
}

type StepTime struct {
	// Step is the number of the step
	Step int `json:"step"`

	// Time is the time at which the step was "started" (e.g. loaded in the UI)
	Time metav1.Time `json:"time"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type ProgressList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	Items []Progress `json:"items"`
}

func (c Progress) NamespaceScoped() bool {
	return false
}
