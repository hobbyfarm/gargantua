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
	// NotBefore is the timestamp before which a OneTimeAccessCode is invalid.
	NotBefore *metav1.Time `json:"notBefore"`

	// NotAfter is the timestamp after which a OneTimeAccessCode is invalid.
	NotAfter *metav1.Time `json:"notAfter"`

	// User is the object name of the user for whom the OneTimeAccessCode is intended.
	User string `json:"user"`
}

type OneTimeAccessCodeStatus struct {
	// Redeemed is the timestamp marking when the OneTimeAccessCode was redeemed.
	// nil if not redeemed.
	Redeemed *metav1.Time `json:"redeemed,omitempty"`
}

func (c OneTimeAccessCode) NamespaceScoped() bool {
	return false
}
