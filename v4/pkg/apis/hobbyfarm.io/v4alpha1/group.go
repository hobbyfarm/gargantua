package v4alpha1

import metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// Group represents a grouping of users and/or authentication provider principals.
// This MAY be a 1:1 mapping with an authN provider group, e.g. LDAP group. However,
// this is not a requirement.
// A group may contain references to multiple authN provider groups.
// For example, an administrator may wish to create a "Hobbyfarm Administrators" group,
// and add multiple local users (via spec.userMembers), and multiple ldap
// groups (via spec.providerMembers["ldap"]).
type Group struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec GroupSpec `json:"spec"`
}

type GroupSpec struct {
	// DisplayName is the pretty (display) name of the Group
	DisplayName string `json:"displayName"`

	// UserMembers contains the object name (not username, not auth principal)
	// of members of this group.
	UserMembers []string `json:"userMembers"`

	// ProviderMembers contains slices of members from specific providers.
	// The key is the name of the provider (e.g. "ldap"),
	// and the string slice contains principals for that provider
	// (e.g. for ldap, the user or group DN) (ex. cn=hf-admins,ou=users,dc=it,dc=example,dc=org"
	ProviderMembers map[string][]string `json:"providerMembers"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type GroupList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`

	Items []Group `json:"items"`
}

func (g Group) NamespaceScoped() bool {
	return false
}
