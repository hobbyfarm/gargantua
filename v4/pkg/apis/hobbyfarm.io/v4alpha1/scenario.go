package v4alpha1

import metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// Scenario represents a unit of learning on a particular topic represented by a series of steps that
// may include interactions with Machine(s). While defined on a separate resource, Steps are where the
// content of a Scenario lives. A Scenario also tracks requirements and configurations for Machines,
// allowed behavior of users, categorization/tagging metadata, and so on.
type Scenario struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec ScenarioSpec `json:"spec"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type ScenarioList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	Items []Scenario `json:"items"`
}

type ScenarioSpec struct {
	// DisplayName is the display (pretty) name of the Scenario
	DisplayName string `json:"string"`

	// Description is the description of the Scenario.
	Description string `json:"description"`

	// Categories is a string slice listing the categories in which this Scenario exists.
	Categories []string `json:"categories"`

	// Tags is a string slice listing the tags that are applicable to this Scenario.
	Tags []string `json:"tags"`

	// Steps is a string slice of object names for Step resources.
	// The order in which they appear in this slice is the order in which they
	// are presented.
	Steps []string `json:"steps"`

	// MachineRequirements details the required machines for this Scenario. This includes both
	// user machines and shared machines.
	MachineRequirements []MachineRequirement `json:"machineRequirements"`

	// KeepaliveDuration is a string representing the period of time a user can go inactive
	// before their machines will be reaped. It can be viewed as a countdown timer that
	// resets each time the user interface submits a keepalive ping to the API.
	// For example if this string is "1h", machines will be reaped one hour after the
	// last keepalive ping is received. If a ping is received before the 1h elapses, the timer
	// starts over with a fresh 1h period.
	//
	// The format of this field is an integer followed by an 'h' or 'm' designation for hours or
	// minutes, respectively. MUST be parseable by time.ParseDuration()
	// Examples include "15m", "5h", "48h", or "120m"
	KeepaliveDuration string `json:"keepaliveDuration"`

	// PauseDuration is a string representing the period of time a user can prevent the reaping of
	// their machines without continuous keepalive pings. While a Course is paused, the keepalive "timer"
	// (described on KeepaliveDuration) is paused and HobbyFarm will not reclaim the Machine.
	// For example, if this string is "1h" and KeepaliveDuration is "30m", if a user pauses their
	// Course and closes their laptop (thus preventing keepalive pings from the UI), the KeepaliveDuration
	// timer will not start until one hour after the Course was paused. Even if 30 minutes elapses
	// machines will not be reaped as the countdown for keepalive is paused for the duration of the pause.
	// A user may un-pause at any time.
	//
	// The format of this field is an integer followed by an 'h' or 'm' designation for hours or minutes,
	// respectively. MUST be parseabl by time.ParseDuration()
	// Examples include "15m", "5h", "48h", or "120m"
	PauseDuration string `json:"pauseDuration"`

	// PauseBehavior describes what sort of behavior affecting pausing should be allowed for this Course.
	// An example is "CanPause" which allowed users to pause their course. "CannotPause" does not.
	PauseBehavior PauseBehavior `json:"pauseBehavior"`
}

func (c Scenario) NamespaceScoped() bool {
	return false
}
