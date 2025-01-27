package v4alpha1

import (
	"github.com/hobbyfarm/gargantua/v4/pkg/property"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// Provider is an object that represents the capability to provision machines.
// Providers are usually implemented as 3rd party operators that interact with HobbyFarm through
// the boundaries of objects such as Machines, MachineSets, and Environments.
type Provider struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec ProviderSpec `json:"spec"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type ProviderList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`

	Items []Provider `json:"items"`
}

type ProviderSpec struct {
	// ProviderConfiguration describes properties that an administrator can configure for the provider
	// For example, an AWS provider may have an item for "AMI", or perhaps "SecurityGroup".
	// This array does not set these properties - it merely defines them for use in
	// resources such as Environments, or MachineTemplates.
	// The presence of a property.Property in this array MAY drive UI elements such as configuration menus.
	ProviderConfiguration map[string]property.Property `json:"providerConfiguration,omitempty"`

	// MachineInformation describes properties that the provider CAN return about a machine.
	// For example, an AWS provider may have an item for "public ip", or perhaps "hostname".
	// This array does not set these properties - it merely defines them for use in
	// the MachineStatus struct. The presence of a property.Property in this array
	// MAY drive UI elements such as variable introspection when writing Scenarios.
	MachineInformation map[string]property.Property `json:"machineInformation,omitempty"`
}

func (c Provider) NamespaceScoped() bool {
	return false
}
