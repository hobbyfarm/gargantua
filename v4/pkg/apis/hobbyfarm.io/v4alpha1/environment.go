package v4alpha1

import (
	genericcondition2 "github.com/hobbyfarm/gargantua/v4/pkg/genericcondition"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// Environment is the representation of a specific instance of a provider.
// Here, "specific instance" means an account or location or instance into which machines can be provisioned
// by the defined Provider for this Environment. An example of an Environment for an AWS provider would be
// a particular AWS account, or particular region within an AWS account.
// Environments encapsulate not only the configuration of the provider but also how users should connect to their
// machines.
// It is not a requirement that separate Environments have separate endpoints for connection, or that they use
// separate cloud provider credentials. Instead, Environments should be used anywhere that top-level configuration
// items for machine provisioning may change. Or, separate Environments can be used to provide lower-level or more
// restricted capacity to public or untrusted users.
type Environment struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   EnvironmentSpec   `json:"spec"`
	Status EnvironmentStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type EnvironmentList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`

	Items []Environment `json:"items"`
}

type EnvironmentSpec struct {
	// Capacity is a numeric count of the capacity in the environment for a particular MachineTemplate.
	// The key is the string ID (not display name) of the MachineTemplate. The int value is the number
	// of concurrently-running machines supported by this environment.
	// This value MUST be user-defined. Users must take into account their capacity in the environment
	// as well as the size of the machine being provisioned. For example, an AWS account may support
	// more t3.large instances than it supports t3.xxlarge.
	// Because of the wide variety of environments and provider implementations it is
	// not feasible for HobbyFarm to calculate sane values for this field.
	Capacity map[string]int `json:"capacity,omitempty"`

	// Endpoints defines a map of endpoints keyed by ConnectProtocol. The value of the k/v pair
	// shall be a network endpoint that a client may call to obtain a connection to a machine in this
	// environment. An example may be a key of "ssh" and a value of "wss://my-host" which corresponds
	// to a secure websocket endpoint to obtain an SSH connection.
	// Previous versions of HobbyFarm called this field "WsEndpoint", and only supported SSH.
	Endpoints map[ConnectProtocol]string `json:"endpoints,omitempty"`

	// Provider is the string name (object name, not display name) of the provider in use for this environment.
	Provider string `json:"provider"`

	// ProviderConfiguration defines k/v configuration for the provider in use for this environment.
	// Keys in this map shall correspond to property names defined in the ProviderSpec of the Provider identified
	// in this struct. Values shall be string representations of the data/value types for that property.
	ProviderConfiguration map[string]string `json:"providerConfiguration,omitempty"`

	// TemplateConfiguration defines k/v configuration for the provider in use for this environment AND
	// for a particular MachineTemplate. The top-level key in this map shall be the name (object name, not display) of
	// a MachineTemplate. The `map[string]string` value shall be k/v configuration for the provider.
	// Keys in that map shall correspond to property names defined in the ProviderSpec of the Provider identified
	// in the parent EnvironmentSpec struct. Values shall be string representations of the data/value types
	// for that property.
	// You may be asking yourself - why does this exist when ProviderConfiguration is also a field?
	// MachineTemplate also defines configuration, so why here too?
	// This is a flexibility thing. Users may wish to define top-level configuration items at the MachineTemplate level.
	// Items such as AMIs that could be valid for all AWS environments. But what if ONE specific environment needs
	// a different AMI for ONE specific template? This field handles that case.
	// There is a hierarchy to configuration items. In increasing order of precedence, it goes:
	// MachineTemplate, Environment(ProviderConfiguration), Environment(TemplateConfiguration).
	// Thus, any configuration item defined in this field shall preempt the same configuration item defined in any other
	// field of related nature on Environment or MachineTemplate objects.
	TemplateConfiguration map[string]map[string]string `json:"templateSpecific,omitempty"`

	// DisplayName is the display, or pretty, name of the Environment.
	DisplayName string `json:"displayName"`

	// MachineNamePrefix is the prefix given to all Machine objects created using this Environment.
	// *Caution*: This will override any template-specific MachineNamePrefix field that has been set.
	// Use with caution.
	MachineNamePrefix string `json:"machineNamePrefix,omitempty"`
}

type EnvironmentStatus struct {
	// Conditions is an array of generic conditions that may crop up on this Environment.
	// There are NO guarantees made about the conditions that exist in this slice.
	Conditions []genericcondition2.GenericCondition `json:"conditions,omitempty"`
}

func (c Environment) NamespaceScoped() bool {
	return false
}
