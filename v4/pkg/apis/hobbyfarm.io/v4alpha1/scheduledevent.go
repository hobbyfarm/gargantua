package v4alpha1

import (
	genericcondition2 "github.com/hobbyfarm/gargantua/v4/pkg/genericcondition"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// ScheduledEvent is the representation of a period of time during which content is available to users.
// A ScheduledEvent not only defines what content is available but also how users shall have access to
// Machine resources during that period of time.
type ScheduledEvent struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ScheduledEventSpec   `json:"spec"`
	Status ScheduledEventStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type ScheduledEventList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`

	Items []ScheduledEvent `json:"items"`
}

// MachineProvisioningRequirement defines for a given MachineType and MachineTemplate, how many of that kind of machine
// are required. For MachineTypeShared, this is a per- ScheduledEvent requirement. For MachineTypeUser this is a
// per-User requirement.
//
// Provisioning scenarios:
//  1. Use any existing pools, do not provision machinesets
//     BindStrategy = Any, CreateMachineSets = nil, PreferRequireMachineSets = nil
//  2. Prefer certain existing pools, do not provision machinesets
//     BindStrategy = PreferMachineSet, CreateMachineSets = nil, PreferRequireMachineSets = []string{preferredSets...}
//  3. Require certain existing pools, do not provision machinesets
//     BindStrategy = RequireMachineSet, CreateMachineSets = nil, PreferRequireMachineSets = []string{requiredSets...}
//  4. Use any existing pools, provision machinesets
//     BindStrategy = Any, CreateMachineSets = []MachineSetSpec{newSets...}, PreferRequireMachineSets = nil
//  5. Prefer certain existing pools, provision machinesets (auto-prefers provisioned machinesets)
//     BindStrategy = PreferMachineSets, CreateMachineSets = []MachineSetSpec{newSets...},
//     PreferRequireMachineSets = []string{preferredSets... + newSets...}
//     (preferredSets may be nil, newSets must not be)
//  6. Require certain existing pools, provision machinesets (auto-requires provisioned machinesets)
//     BindStrategy = RequireMachineSets, CreateMachineSets = []MachineSetSpec{newSets...},
//     PreferRequireMachineSets = []string{requiredSets... + newSets...}
//     (requiredSets may be nil, newSets must not be)
type MachineProvisioningRequirement struct {
	MachineRequirement `json:",inline"`

	// BindStrategy determines how this MachineProvisioningRequirement shall be fulfilled.
	// BindStrategyAny = Fill this requirement from any matching machine set
	// BindStrategyPreferMachineSet = Fill this requirement first from a list of MachineSet, then from any matching machine set
	// BindStrategyRequireMachineSet = Only fill this requirement from a list of MachineSet
	BindStrategy BindStrategy `json:"bindStrategy"`

	// CreateMachineSet defines the spec of a MachineSet to be created for this request.
	CreateMachineSet *MachineSetSpec `json:"createMachineSet,omitempty"`

	// PreferRequireMachineSets is a slice of names of MachineSet objects that are either preferred or required,
	// depending on the value of BindStrategy.
	PreferRequireMachineSets []string `json:"preferRequireMachineSets"`
}

type ScheduledEventSpec struct {
	// DisplayName is the display name of the ScheduledEvent
	DisplayName string `json:"displayName"`

	// StartTime is the time at which the ScheduledEvent will become active. Active here means that users will be able
	// to begin accessing content and claiming machines.
	StartTime metav1.Time `json:"startTime"`

	// ProvisioningStartTime is the time at which HobbyFarm will begin to provision machines. By default this is set
	// to 30 minutes prior to StartTime. It may be adjusted if a provider or environment needs more than 30 minutes
	// notice to prepare the machines. Or, conversely, if no warm-up time is required.
	ProvisioningStartTime metav1.Time `json:"provisioningStartTime"`

	// EndTime is the time at which HobbyFarm will cut off access to the content and machines. After this time, users
	// will no longer be able to claim machines or access scenarios/courses.
	EndTime metav1.Time `json:"endTime"`

	// ExpirationStrategy defines the behavior that HobbyFarm will use when a ScheduledEvent passes its EndTime.
	// For example, if ExpirationStrategy is set to "CutOff", access to machines will be immediately terminated.
	// If ExpirationStrategy is set to "Soft", users will continue to have access while they are actively using
	// machines, but cannot create new machines or access other content than what they are currently using.
	ExpirationStrategy ExpirationStrategy `json:"expirationStrategy"`

	// RequiredMachines defines the number of machines of each MachineTemplate that are required for this
	// ScheduledEvent.
	RequiredMachines []MachineProvisioningRequirement `json:"requiredMachines"`

	// PrintingOption defines if users should be allowed to print out the content to which they have access.
	PrintingOption PrintingOption `json:"printingOption"` // PrintingEnabled, PrintingDisabled

	// Scenarios is a list of Scenario names to which this ScheduledEvent shall grant access
	Scenarios []string `json:"scenarios,omitempty"`

	// Courses is a list of Course names to which this ScheduledEvent shall grant access
	Courses []string `json:"courses,omitempty"`
}

type ScheduledEventStatus struct {
	// Conditions is an array of generic conditions that may crop up on this MachineSet.
	// There are NO guarantees made about the conditions that exist in this slice.
	Conditions []genericcondition2.GenericCondition `json:"conditions"`

	// CreatedMachineSets is a list of the names of MachineSet objects that have been created as a result
	// of this ScheduledEvent. Presence in this list does not imply presence in PreferRequireMachineSets.
	CreatedMachineSets []string `json:"createdMachineSets"`
}

func (c ScheduledEvent) NamespaceScoped() bool {
	return false
}
