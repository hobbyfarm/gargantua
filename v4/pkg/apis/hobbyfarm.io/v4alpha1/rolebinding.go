package v4alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apiserver/pkg/authentication/user"
)

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type RoleBinding struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Role   string   `json:"role"`
	Users  []string `json:"users"`
	Groups []string `json:"groups"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type RoleBindingList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	Items []RoleBinding `json:"items"`
}

func (rb RoleBinding) MatchesUser(user user.Info) bool {
	for _, u := range rb.Users {
		if user.GetName() == u {
			return true
		}
	}

	for _, g := range rb.Groups {
		for _, k := range user.GetGroups() {
			if g == k {
				return true
			}
		}
	}

	return false
}

func (rb RoleBinding) NamespaceScoped() bool {
	return false
}
