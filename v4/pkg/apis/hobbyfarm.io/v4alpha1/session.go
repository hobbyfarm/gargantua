package v4alpha1

import (
	genericcondition2 "github.com/hobbyfarm/gargantua/v4/pkg/genericcondition"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// Session holds the information used to track a user's engagement with HobbyFarm on a particular
// Scenario or Course. It contains information such as the step that the user is on, what
// machines are assigned to them, when they started, etc.
type Session struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   SessionSpec   `json:"spec"`
	Status SessionStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type SessionList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`

	Items []Session `json:"items"`
}

type SessionSpec struct {
	// Scenario is the object name of the scenario that this session is tracking.
	Scenario string `json:"scenario,omitempty"`

	// Course is the object name of the course that this session is tracking.
	Course string `json:"course,omitempty"`

	// PersistenceStrategy defines what shall happen to the machines being used
	// in this session after the scenario has ended.
	PersistenceStrategy MachinePersistenceStrategy `json:"persistenceStrategy"`

	// User is the object name of the user to whom this Session belongs.
	User string `json:"user"`

	// AccessCode is the code (not obj name) used to get access to this scenario.
	// May be empty if access code was not used.
	AccessCode string `json:"accessCode,omitempty"`

	// ScheduledEvent is the object name of the scheduled event from which
	// this Session was started.
	ScheduledEvent string `json:"scheduledEvent,omitempty"`
}

type SessionStatus struct {
	// Conditions is a slice of conditions that may impact this session.
	Conditions []genericcondition2.GenericCondition `json:"conditions"`

	// MachineClaim is the object name of the MachineClaim generated from this
	// session.
	MachineClaim string `json:"machineClaim"`

	// Progress is a slice of Progress structs that detail, for a given scenario, what
	// position the user is in.
	Progress []Progress `json:"progress"`
}

func (c Session) NamespaceScoped() bool {
	return false
}
