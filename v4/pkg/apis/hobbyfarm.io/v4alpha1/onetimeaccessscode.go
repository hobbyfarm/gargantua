package v4alpha1

import metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// OneTimeAccessCode is the representation of a single-use access code
type OneTimeAccessCode struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   OneTimeAccessCodeSpec   `json:"spec"`
	Status OneTimeAccessCodeStatus `json:"status"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type OneTimeAccessCodeList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	Items []OneTimeAccessCode `json:"items"`
}

type OneTimeAccessCodeSpec struct {
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

	// AccessDuration defines how long the code will remain active *after redemption*.
	// In other words, from the moment a user redeems an OTAC, this field determines for how long they
	// shall have access to the resources described in this resource (courses, events, etc.).
	AccessDuration *metav1.Duration `json:"accessDuration,omitempty"`
}

type OneTimeAccessCodeStatus struct {
	// Redeemed is the timestamp marking when the OneTimeAccessCode was redeemed.
	// nil if not redeemed.
	Redeemed *metav1.Time `json:"redeemed,omitempty"`

	// Expires is the timestamp marking when the OneTimeAccessCode shall stop providing
	// access to resources. This value is status.redeemed + spec.accessDuration.
	AccessExpires *metav1.Time `json:"accessExpires,omitempty"`

	// User is the object name of the User who has consumed this OTAC.
	User string `json:"user"`
}

func (c OneTimeAccessCode) NamespaceScoped() bool {
	return false
}
