package v4alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"time"
)

type AccessCodeState string

const (
	AccessCodeInactive AccessCodeState = "Inactive"
	AccessCodeActive   AccessCodeState = "Active"
)

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// AccessCode stores a string which grants access to any number of Scenarios, Courses, ScheduledEvents, or MachineSets.
// It can be considered a sort of password, a pre-shared key.
type AccessCode struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   AccessCodeSpec   `json:"spec,omitempty"`
	Status AccessCodeStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type AccessCodeList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`

	Items []AccessCode `json:"items"`
}

type AccessCodeSpec struct {
	// Scenarios is a list of the names of scenarios to which this AccessCode shall grant access.
	Scenarios []string `json:"scenarios,omitempty"`

	// Courses is a list of the names of Courses to which this AccessCode shall grant access.
	Courses []string `json:"courses,omitempty"`

	// ScheduledEvents is a list of the names of ScheduledEvents to which this AccessCode shall grant access.
	ScheduledEvents []string `json:"scheduledEvents,omitempty"`

	// MachineSets is a list of the names of MachineSets to which this AccessCode shall grant access.
	MachineSets []string `json:"machineSets,omitempty"`

	// NotBefore is a timestamp that defines the point in time after which an AccessCode may be used.
	// If defined, the AccessCode may not be used before this timestamp. If nil, an AccessCode is considered valid
	// immediately after creation - provided NotAfter has not passed if set.
	NotBefore *metav1.Time `json:"notBefore,omitempty"`

	// NotAfter is a timestamp that defines the point in time after which an AccessCode may NOT be used.
	// If defined, the AccessCode may not be used after this timestamp. If nil, an AccessCode is considered valid
	// forever - provided NotBefore has passed if set.
	NotAfter *metav1.Time `json:"notAfter,omitempty"`
}

type AccessCodeStatus struct {
	// Status is the state of the access code.
	// AccessCodes can be AccessCodeActive or AccessCodeInactive, depending on the values of NotBefore & NotAfter,
	// as well as the current time.
	Status AccessCodeState `json:"status"`
}

func (a AccessCode) NamespaceScoped() bool {
	return false
}

func (a *AccessCode) IsActive() bool {
	if a.Spec.NotBefore != nil {
		if a.Spec.NotBefore.Time.After(time.Now()) {
			return false
		}
	}

	if a.Spec.NotAfter != nil {
		if a.Spec.NotAfter.Time.Before(time.Now()) {
			return false
		}
	}

	return true
}
