package v4alpha1

import metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// User is any user (person, not machine) that interacts with HobbyFarm.
// Administrators and end-users alike both have User objects to represent them.
type User struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   UserSpec   `json:"spec"`
	Status UserStatus `json:"status"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type UserList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`

	Items []User `json:"items"`
}

type UserSpec struct {
	// Principals holds a slice of all the identifiers for a user from various
	// authentication sources. For example, a local auth value would be of the form
	// local://[user obj name], e.g. local://u-uijkks349d
	// Other auth sources will have their own forms. LDAP may have a form such as
	// ldap://CN=john.doe,OU=Users,DC=example,DC=com or something else like that.
	Principals []string `json:"principals"`

	// DisplayName holds the display name of the user for use in e.g. UIs.
	// In the case of local auth, this field is likely set by the creator of the
	// User object. In the case of 3rd party auth, this field comes from a property
	// on that authentication source.
	DisplayName string `json:"displayName"`

	// LocalAuthDetails holds details about the User's local authentication.
	// Only used when performing local auth.
	LocalAuthDetails *LocalAuthDetails `json:"localAuthDetails,omitempty"`

	// AccessCodes is a slice of all access codes that this user has entered. These
	// values will be the codes themselves, *NOT* the AccessCode object names.
	AccessCodes []string `json:"accessCodes"`

	// Settings holds the settings that the user has configured. Things like terminal theme,
	// dark mode, etc.
	Settings map[string]string `json:"settings"`
}

type LocalAuthDetails struct {
	// Username is the username of the user
	Username string `json:"username"`

	// PasswordSecret is the object name of the Secret that contains the user's password
	PasswordSecret string `json:"passwordSecret"`
}

type UserStatus struct {
	// LastLoginTimestamp is the timestamp of when the user last logged in.
	LastLoginTimestamp metav1.Time `json:"lastLoginTimestamp"`
}

func (c User) NamespaceScoped() bool {
	return false
}
