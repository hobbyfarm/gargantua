package v4alpha1

import (
	genericcondition2 "github.com/hobbyfarm/gargantua/v4/pkg/genericcondition"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// Machine is the record of an instance of a MachineTemplate as provisioned via a Provider into an Environment.
// For example, a Machine should have a 1:1 correlation with e.g. an EC2 instance or a DigitalOcean droplet.
// The Machine object carries the configuration from HF and the status from the Provider of the machine.
type Machine struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   MachineSpec   `json:"spec"`
	Status MachineStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type MachineList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`

	Items []Machine `json:"items"`
}

type MachineSpec struct {
	// MachineType defines the type of the machine, e.g. User or Shared.
	MachineType MachineType `json:"machineType"`

	// Provider defines the Provider through which this Machine shall be provisioned.
	Provider string `json:"provider"`

	// ProviderConfiguration defines configuration options by provider.
	// This shall define k/v pairs corresponding to the properties as defined on the ProviderSpec object.
	ProviderConfiguration map[string]string `json:"configurationOptions"`

	// MachineSet identified to which MachineSet this Machine belongs.
	MachineSet string `json:"machineSet"`

	// MachineTemplate identifies of which MachineTemplate this Machine is an instance.
	MachineTemplate string `json:"machineTemplate"`

	// Environment identifies into to which Environment this Machine shall be provisioned.
	Environment string `json:"environment"`

	// ConnectEndpoints is a map of all connection endpoints a user can take to access this machine.
	// The keys in this map are protocols such as 'ssh' or 'rdp'. Values are endpoints
	// that client may contact to establish connection.
	// Any endpoint in this map MUST be available for a user to use.
	// Endpoints present in this map are a combination of protocols supported by the MachineTemplate
	// and Endpoints listed in the Environment.
	ConnectEndpoints map[ConnectProtocol]string `json:"connectEndpoints"`
}

type MachineStatus struct {
	// Conditions is an array of generic conditions that may crop up on this MachineSet.
	// There are NO guarantees made about the conditions that exist in this slice.
	Conditions []genericcondition2.GenericCondition

	// MachineInformation is a map of information about this Machine as reported by the Provider.
	// Keys in this map shall be those defined in the MachineInformation field on the Provider. Values
	// shall be the string representations of those properties.
	MachineInformation map[string]string `json:"machineInformation"`
}

func (c Machine) NamespaceScoped() bool {
	return false
}
