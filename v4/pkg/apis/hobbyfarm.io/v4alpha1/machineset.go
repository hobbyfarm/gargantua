package v4alpha1

import (
	genericcondition2 "github.com/hobbyfarm/gargantua/v4/pkg/genericcondition"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// MachineSet defines a collection of machines that are scheduled together.
// All machines in a MachineSet are of identical configuration, MachineTemplate, etc.
// The settings on the MachineSetSpec struct define how a provider should provision these machines.
type MachineSet struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   MachineSetSpec   `json:"spec"`
	Status MachineSetStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type MachineSetList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`

	Items []MachineSet `json:"items"`
}

type MachineSetSpec struct {
	// AvailabilityConfiguration defines the configuration for end-user availability of the machines in this pool
	AvailabilityConfiguration AvailabilityConfiguration `json:"availabilityConfiguration"`

	// ProvisioningStrategy defines what strategy should be used when provisioning these machines.
	// This will inform HobbyFarm on how to create machine resources so that providers can provision them.
	// For example, if the ProvisioningStrategy is set to AutoScale, and MaxProvisioned is set to 20 with
	// MinAvailable set to 10, then HF will begin by creating 10 Machine objects. As machines get claimed by
	// users, HF will create additional Machine objects up to the MaxProvisioned value of 20.
	// In another example, if the ProvisioningStrategy is set to OnDemand, then MinAvailable is ignored.
	// MaxProvisioned is used as an upper bound for concurrently provisioned machines, and new machines are provisioned
	// only when a matching MachineClaim is created.
	ProvisioningStrategy ProvisioningStrategy `json:"provisioningStrategy"`

	// MaxProvisioned is the maximum number of concurrently provisioned machines that can exist in this
	// MachineSet.
	MaxProvisioned int `json:"maxProvisioned"`

	// MinAvailable is the minimum number of available machines provisioned in this MachineSet. This value is only
	// considered when using ProvisioningStrategyAutoScale. It is the minimum number of concurrently _available_
	// (not just provisioned, but also unclaimed) machines for this set.
	MinAvailable int `json:"minAvailable"`

	// Provider is the string name (not display) of the provider through which these machines shall be provisioned.
	Provider string `json:"provider"`

	// ProviderConfiguration defines configuration options by provider.
	// This shall define k/v pairs corresponding to the properties as defined on the ProviderSpec object.
	ProviderConfiguration map[string]string `json:"providerConfiguration"`

	// MachineTemplate is the identifier of the MachineTemplate that all machines in this set shall use.
	MachineTemplate string `json:"machineTemplate"`

	// Environment is the identifier of the Environment into which all machines in this set shall be provisioned.
	Environment string `json:"environment"`

	// MachineNamePrefix is the prefix given to the name of all Machine objects created using this MachineSet.
	// *Caution*: Setting this value will override any other MachineNamePrefix field set in e.g.
	// MachineTemplate or Environment.
	MachineNamePrefix string `json:"machineNamePrefix,omitempty"`
}

type MachineSetStatus struct {
	// Conditions is an array of generic conditions that may crop up on this MachineSet.
	// There are NO guarantees made about the conditions that exist in this slice.
	Conditions []genericcondition2.GenericCondition

	// Provisioned is the current number of machines that are provisioned in this MachineSet.
	// This is a _total_ that includes both claimed and unclaimed machines.
	Provisioned int `json:"provisioned"`

	// Available is the current number of machines that are provisioned _and unclaimed_ in this MachineSet.
	Available int `json:"available"`
}

func (c MachineSet) NamespaceScoped() bool {
	return false
}
