package v4alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type LdapConfig struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	LdapHost string `json:"ldapHost"`

	BindUsername       string `json:"bindUsername"`
	BindPasswordSecret string `json:"bindPasswordSecret"`

	SearchBase  string          `json:"searchBase"`
	SearchScope LdapSearchScope `json:"searchScope"`

	SearchFilter string `json:"searchFilter"`

	UsernameField   string `json:"usernameField"`
	UserObjectClass string `json:"userObjectClass"`

	DisplayNameField string `json:"displayNameField"`

	GroupObjectClas string `json:"groupObjectClass"`

	GroupLookupField string `json:"groupLookupField"`
}

type LdapSearchScope string

const (
	LdapSearchScopeBaseObject  LdapSearchScope = "BaseObject"
	LdapSearchScopeSingleLevel LdapSearchScope = "SingleLevel"
	LdapSearchScopeSubtree     LdapSearchScope = "Subtree"
)

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
