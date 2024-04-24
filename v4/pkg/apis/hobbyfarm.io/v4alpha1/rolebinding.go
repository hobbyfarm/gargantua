package v4alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/apiserver/pkg/authentication/user"
)

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type RoleBinding struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Role   string           `json:"role"`
	Users  sets.Set[string] `json:"users"`
	Groups sets.Set[string] `json:"groups"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type RoleBindingList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	Items []RoleBinding `json:"items"`
}

func (rb RoleBinding) MatchesUser(u user.Info) bool {
	if rb.Users.Has(u.GetName()) {
		return true
	}

	for _, g := range u.GetGroups() {
		if rb.Groups.Has(g) {
			return true
		}
	}

	return false
}
