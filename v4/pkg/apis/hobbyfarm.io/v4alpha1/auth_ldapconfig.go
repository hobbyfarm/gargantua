package v4alpha1

import (
	"github.com/hobbyfarm/gargantua/v4/pkg/genericcondition"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	ConditionBindSuccessful = "BindSuccessful"
)

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// LdapConfig stores the configuration for LDAP authentication to a specific LDAP instance.
type LdapConfig struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   LdapConfigSpec   `json:"spec"`
	Status LdapConfigStatus `json:"status,omitempty"`
}

type LdapSearchScope string

const (
	LdapSearchScopeBaseObject  LdapSearchScope = "BaseObject"
	LdapSearchScopeSingleLevel LdapSearchScope = "SingleLevel"
	LdapSearchScopeSubtree     LdapSearchScope = "Subtree"
)

type LdapConfigSpec struct {
	ServerDisplayName string `json:"serverDisplayName"`
	LdapHost          string `json:"ldapHost"`

	BindUsername       string `json:"bindUsername"`
	BindPasswordSecret string `json:"bindPasswordSecret"`

	SearchBase  string          `json:"searchBase"`
	SearchScope LdapSearchScope `json:"searchScope"`

	SearchFilter string `json:"searchFilter"`

	UsernameField   string `json:"usernameField"`
	UserObjectClass string `json:"userObjectClass"`

	DisplayNameField string `json:"displayNameField"`

	GroupObjectClass string `json:"groupObjectClass"`

	GroupLookupField string `json:"groupLookupField"`
}

type LdapConfigStatus struct {
	Conditions map[string]genericcondition.GenericCondition `json:"conditions"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type LdapConfigList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	Items []LdapConfig `json:"items"`
}

func (l LdapSearchScope) ConvertToLdapScope() int {
	switch l {
	case LdapSearchScopeBaseObject:
		return 0
	case LdapSearchScopeSingleLevel:
		return 1
	case LdapSearchScopeSubtree:
		return 2
	}

	return 1
}

func (lc LdapConfig) NamespaceScoped() bool {
	return false
}
