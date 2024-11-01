package v4alpha1

import metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type Course struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec CourseSpec `json:"spec"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type CourseList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	Items []Course `json:"items"`
}

type CourseSpec struct {
	// DisplayName is the display (pretty) name of the course.
	DisplayName string `json:"displayName"`

	// Description is the string description of the course.
	Description string `json:"description"`

	// Scenarios is a slice of object names for scenarios that are in this course.
	Scenarios []string `json:"scenarios"`

	// Categories is a slice of categories (strings) that this course fits into.
	// Categories may be used to filter Course objects in interfaces.
	Categories []string `json:"categories"`

	// Tags is a slice of strings that represent tags applied to this course.
	// Tags may be used to filter Course objects in interfaces.
	Tags []string `json:"tags"`

	// MachineRequirements is a slice of MachineRequirement structs that define
	// what machines are required for this Course.
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
	// respectively. MUST be parseable by time.ParseDuration()
	// Examples include "15m", "5h", "48h", or "120m"
	PauseDuration string `json:"pauseDuration,omitempty"`

	// MachinePersistenceStrategy determines what HobbyFarm should do with the Machine objects provisioned
	// for this Course after a particular Scenario has been completed.
	// When a user completes a Scenario that is *NOT* part of a Course, their Machine resources are
	// reclaimed and recycled for the next user. However, Courses may have multiple Scenarios whose content
	// (and thus, machines) flow together. By setting a persistence strategy of "PersistThroughCourse", machines
	// will *NOT* be reclaimed after a Scenario is completed - they will live for the life of the Course.
	// Conversely, if a persistence strategy is set to "NewPerScenario", old machines will be reclaimed
	// (and new ones provisioned/claimed) for each scenario.
	MachinePersistenceStrategy MachinePersistenceStrategy `json:"machinePersistenceStrategy"`

	// PauseBehavior describes what sort of behavior affecting pausing should be allowed for this Course.
	// An example is "CanPause" which allowed users to pause their course. "CannotPause" does not.
	PauseBehavior PauseBehavior `json:"pauseBehavior"`
}

func (c Course) NamespaceScoped() bool {
	return false
}
