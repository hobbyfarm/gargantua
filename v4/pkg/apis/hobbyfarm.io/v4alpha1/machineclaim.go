package v4alpha1

import (
	genericcondition2 "github.com/hobbyfarm/gargantua/v4/pkg/genericcondition"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// MachineClaim is an object representing a User's desire to claim a Machine for their exclusive use.
// A MachineClaim does not necessarily represent the fulfillment of this desire. For example, a MachineClaim
// is created any time a user wants to use a Machine. However, if no Machine is available that matches the
// claim criteria, then the claim may fail.
type MachineClaim struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   MachineClaimSpec   `json:"spec"`
	Status MachineClaimStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type MachineClaimList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`

	Items []MachineClaim `json:"items"`
}

type MachineClaimSpec struct {
	// MachineTemplate is the identifier of the template for which we need a Machine.
	MachineTemplate string `json:"machineTemplate"`

	// User is the identifier of the User requesting the Machine.
	User string `json:"user"`

	// AccessCode is an optional access code that can be used to provide access to a MachineSet.
	// For example, if a user has an AccessCode that provides them direct access to a Scenario, there may also be a
	// corresponding MachineSet for that AccessCode that shall grant a user a Machine.
	AccessCode string `json:"accessCode,omitempty"`

	// ScheduledEvent is an optional identifier of a ScheduledEvent that can be used to provide access to a
	// MachineSet.
	// For example, if a user is taking part in a ScheduledEvent, this field may be filled with the name of that SE.
	// HobbyFarm will then grant them access to MachineSets that are created to support that ScheduledEvent.
	ScheduledEvent string `json:"scheduledEvent,omitempty"`

	// BindStrategy determines how the user wants HobbyFarm to fulfill their request for a machine.
	// For example, if the BindStrategy is set to BindStrategyAnyAvailable, then HobbyFarm will attempt to claim
	// any matching Machine (based on the criteria herein) for the user. If the BindStrategy is set to
	// BindStrategyRequireMachineSet, then only machines from the specified MachineSets will be attempted to claim.
	BindStrategy BindStrategy `json:"bindStrategy"`

	// BindStrategyMachineSets is a list of the names of MachineSet objects we either want to prefer or require
	// our binding to come from.
	PreferRequireMachineSets []string `json:"bindStrategyMachineSets"`
}

type MachineClaimStatus struct {
	// Phase represents the phase of the MachineClaim. MachineClaims can go through four stages,
	// Available, Bound, Failed, and Terminated.
	// Every MachineClaim starts as Available. This is the period where HobbyFarm is attempting to match
	// a MachineClaim to a Machine. If it is able to do so, then the MachineClaimPhase changes to Bound.
	// If it is unable to do so after a period of time, then the MachineClaimPhase changes to Failed.
	// When a user is done with a Machine, they indicate to HobbyFarm by changing the MachineClaimPhase to Terminated.
	// The MachineClaimPhase can also change to Terminated via other mechanisms such as the user's time expiring
	// or the Machine becoming unavailable.
	Phase MachineClaimPhase `json:"phase"`

	// Machine is the object name of the Machine resource claimed by this MachineClaim.
	// Field will be empty unless MachineClaimPhase is either Bound or Terminated.
	Machine string `json:"machine,omitempty"`

	// Conditions is an array of generic conditions that may crop up on this MachineSet.
	// There are NO guarantees made about the conditions that exist in this slice.
	Conditions []genericcondition2.GenericCondition
}

func (c MachineClaim) NamespaceScoped() bool {
	return false
}
