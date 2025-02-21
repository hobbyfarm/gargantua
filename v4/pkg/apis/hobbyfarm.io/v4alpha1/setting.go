package v4alpha1

import (
	"github.com/hobbyfarm/gargantua/v4/pkg/property"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// Setting is a configuration option for HobbyFarm. Along with the embedded Property struct
// is a Value, which is a string encoding of the value.
type Setting struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	property.Property `json:",inline"`

	// Value is the string encoded value of the setting
	Value string `json:"value"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type SettingList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	Items []Setting `json:"items"`
}

func (c Setting) NamespaceScoped() bool {
	return false
}
