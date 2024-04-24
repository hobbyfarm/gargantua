package v4alpha1

import metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// MachineTemplate is an object that represents a specific instance of machine that can be
// provisioned into one or more Environments by one or more Providers. A MachineTemplate
// is meant to bridge the gap between environments by providing a stable representation of a machine
// that environments can utilize.
// For example, a MachineTemplate may be created with the name "ClusterNode-Ubuntu-2004". It is then up to
// each Environment to define options necessary for that environment to allow the creation of a
// `ClusterNode-Ubuntu-2004` machine.
// A common use case here is to define in the MachineTemplate any configuration that can span providers or
// environments. Items such as cloud config or ssh username which are not specific to a provider implementation. Then,
// further configuration can be done on the Environment. Those further configurations could be items such as the
// machine image or AMI, or perhaps the security group, which are both examples of environment-specific options.
type MachineTemplate struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec MachineTemplateSpec `json:"spec"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type MachineTemplateList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`

	Items []MachineTemplate `json:"items"`
}

type MachineTemplateSpec struct {
	// MachineType defines what type of machine is defined in this template.
	// For example, a machine may be MachineTypeUser which is a machine that can be assigned to a user.
	// A MachineTypeShared may be accessible by multiple users.
	MachineType MachineType `json:"machineType"`

	// ProviderConfiguration defines configuration options by provider.
	// The top-level key is a string which SHOULD be the name of a Provider.
	// The lower-level `map[string]string` shall define k/v pairs corresponding to the
	// properties as defined on the ProviderSpec object.
	ProviderConfiguration map[string]map[string]string `json:"providerConfiguration,omitempty"`

	// DisplayName is the display, or pretty, name of the machine template.
	DisplayName string `json:"displayName"`

	// ConnectProtocols is an array of protocols that MAY be supported by the
	// machine. The presence of a protocol in this list, plus an endpoint for that protocol defined
	// in EnvironmentSpec will drive connection options for users.
	ConnectProtocols []ConnectProtocol `json:"connectProtocol"`

	// MachineNamePrefix is a prefix to apply to machines generated using this template.
	// It can be overridden by MachineNamePrefix fields further down the "stack"
	// (e.g. in an Environment)
	MachineNamePrefix string `json:"machineNamePrefix,omitempty"`

	// PredefinedServices lists the object names of all PredefinedService resources that this
	// template SHOULD offer.
	PredefinedServices []string `json:"predefinedServices,omitempty"`

	// ExtraConfig provides a place where future use cases can store configuration items.
	// This can also be used for configuration of a nature that does not meet HobbyFarm uses
	// but may have other requirements by users, providers, or machines.
	ExtraConfig map[string]string `json:"extraConfig,omitempty"`
}

func (c MachineTemplate) NamespaceScoped() bool {
	return false
}
