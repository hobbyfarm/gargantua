package v4alpha1

import (
	"github.com/hobbyfarm/gargantua/v3/pkg/property"
	genericcondition2 "github.com/hobbyfarm/gargantua/v4/pkg/genericcondition"
	"github.com/rancher/wrangler/pkg/condition"
	v1 "k8s.io/api/core/v1"
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

// MachineType defines what type of machine is defined in this template.
// For example, a machine may be MachineTypeUser which is a machine that can be assigned to a user.
// A MachineTypeShared may be accessible by multiple users.
type MachineType string

const (
	MachineTypeUser   MachineType = "User"
	MachineTypeShared MachineType = "Shared"
)

// ConnectProtocol is an array of protocols that MAY be supported by the
// machine. The presence of a protocol in this list, plus an endpoint for that protocol defined
// in EnvironmentSpec will drive connection options for users.
type ConnectProtocol string

const (
	ConnectProtocolSSH  ConnectProtocol = "ssh"
	ConnectProtocolGuac ConnectProtocol = "guac"
	ConnectProtocolVNC  ConnectProtocol = "vnc"
	ConnectProtocolRDP  ConnectProtocol = "rdp"
)

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
	Conditions []genericcondition2.GenericCondition
}

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

type MachineSetAvailability string

const (
	MachineSetAvailabilityAccessCode     MachineSetAvailability = "AccessCode"
	MachineSetAvailabilityScheduledEvent MachineSetAvailability = "ScheduledEvent"
	MachineSetAvailabilityPool           MachineSetAvailability = "Pool"
)

type ProvisioningStrategy string

const (
	ProvisioningStrategyAutoScale ProvisioningStrategy = "AutoScale"
	ProvisioningStrategyDynamic   ProvisioningStrategy = "OnDemand"
)

type AvailabilityConfiguration struct {
	// Availability defines what strategy will be used for making machines available to users.
	Availability MachineSetAvailability `json:"availability"`

	// Value defines a string identifier related to the Availability. For example,
	// in the case of ScheduledEvent availability, this value may be the name of
	// the associated ScheduledEvent.
	Value string `json:"value,omitempty"`
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

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// MachineClaim is an object representing a User's desire to claim a Machine for their exclusive use.
// A MachineClaim does not necessarily represent the fulfillment of this desire. For example, a MachineClaim
// is created any time a user wants to use a Machine. However, if no Machine is available that matches the
// claim criteria, then the claim may fail.
type MachineClaim struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   MachineClaimSpec   `json:"spec"`
	Status MachineClaimStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type MachineClaimList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`

	Items []MachineClaim `json:"items"`
}

type BindStrategy string

const (
	BindStrategyAnyAvailable       BindStrategy = "Any"
	BindStrategyPreferMachineSets  BindStrategy = "PreferMachineSets"
	BindStrategyRequireMachineSets BindStrategy = "RequireMachineSets"
)

type MachineClaimSpec struct {
	// MachineTemplate is the identifier of the template for which we need a Machine.
	MachineTemplate string `json:"machineTemplate"`

	// User is the identifier of the User requesting the Machine.
	User string `json:"user"`

	// AccessCode is an optional access code that can be used to provide access to a MachineSet.
	// For example, if a user has an AccessCode that provides them direct access to a Scenario, there may also be a
	// corresponding MachineSet for that AccessCode that shall grant a user a Machine.
	AccessCode string `json:"accessCode,omitempty"`

	// ScheduledEvent is an optional identifier of a ScheduledEvent that can be used to provide access to a
	// MachineSet.
	// For example, if a user is taking part in a ScheduledEvent, this field may be filled with the name of that SE.
	// HobbyFarm will then grant them access to MachineSets that are created to support that ScheduledEvent.
	ScheduledEvent string `json:"scheduledEvent,omitempty"`

	// BindStrategy determines how the user wants HobbyFarm to fulfill their request for a machine.
	// For example, if the BindStrategy is set to BindStrategyAnyAvailable, then HobbyFarm will attempt to claim
	// any matching Machine (based on the criteria herein) for the user. If the BindStrategy is set to
	// BindStrategyRequireMachineSet, then only machines from the specified MachineSets will be attempted to claim.
	BindStrategy BindStrategy `json:"bindStrategy"`

	// BindStrategyMachineSets is a list of the names of MachineSet objects we either want to prefer or require
	// our binding to come from.
	PreferRequireMachineSets []string `json:"bindStrategyMachineSets"`
}

type MachineClaimPhase string

const (
	MachineClaimPhaseRequested  MachineClaimPhase = "Requested"
	MachineClaimPhaseBound      MachineClaimPhase = "Bound"
	MachineClaimPhaseFailed     MachineClaimPhase = "Failed"
	MachineClaimPhaseTerminated MachineClaimPhase = "Terminated"
)

type MachineClaimStatus struct {
	// Phase represents the phase of the MachineClaim. MachineClaims can go through four stages,
	// Available, Bound, Failed, and Terminated.
	// Every MachineClaim starts as Available. This is the period where HobbyFarm is attempting to match
	// a MachineClaim to a Machine. If it is able to do so, then the MachineClaimPhase changes to Bound.
	// If it is unable to do so after a period of time, then the MachineClaimPhase changes to Failed.
	// When a user is done with a Machine, they indicate to HobbyFarm by changing the MachineClaimPhase to Terminated.
	// The MachineClaimPhase can also change to Terminated via other mechanisms such as the user's time expiring
	// or the Machine becoming unavailable.
	Phase MachineClaimPhase `json:"phase"`

	// Machine is the object name of the Machine resource claimed by this MachineClaim.
	// Field will be empty unless MachineClaimPhase is either Bound or Terminated.
	Machine string `json:"machine,omitempty"`

	// Conditions is an array of generic conditions that may crop up on this MachineSet.
	// There are NO guarantees made about the conditions that exist in this slice.
	Conditions []genericcondition2.GenericCondition
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// ScheduledEvent is the representation of a period of time during which content is available to users.
// A ScheduledEvent not only defines what content is available but also how users shall have access to
// Machine resources during that period of time.
type ScheduledEvent struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ScheduledEventSpec   `json:"spec"`
	Status ScheduledEventStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type ScheduledEventList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`

	Items []ScheduledEvent `json:"items"`
}

type PrintingOption string

const (
	PrintingEnabled  PrintingOption = "PrintingEnabled"
	PrintingDisabled PrintingOption = "PrintingDisabled"
)

type ExpirationStrategy string

const (
	ExpirationStrategyCutOff = "CutOff"
	ExpirationStrategySoft   = "Soft"
)

// MachineRequirement defines for a given MachineType and MachineTemplate, how many are required.
// This struct is split out from MachineProvisioningRequirement because it is re-used
// in contexts that do not require provisioning strategies, such as Course and Scenario.
type MachineRequirement struct {
	// MachineTemplate is the name of the required MachineTemplate
	MachineTemplate string `json:"machineTemplate"`

	// Count is the number of required Machines either per-User (when MachineType = MachineTypeUser)
	// or per- ScheduledEvent (when MachineType = MachineTypeShared)
	Count int `json:"count"`

	// MachineType is the type of machine to require
	MachineType MachineType `json:"machineType"`
}

// MachineProvisioningRequirement defines for a given MachineType and MachineTemplate, how many of that kind of machine
// are required. For MachineTypeShared, this is a per- ScheduledEvent requirement. For MachineTypeUser this is a
// per-User requirement.
//
// Provisioning scenarios:
//  1. Use any existing pools, do not provision machinesets
//     BindStrategy = Any, CreateMachineSets = nil, PreferRequireMachineSets = nil
//  2. Prefer certain existing pools, do not provision machinesets
//     BindStrategy = PreferMachineSet, CreateMachineSets = nil, PreferRequireMachineSets = []string{preferredSets...}
//  3. Require certain existing pools, do not provision machinesets
//     BindStrategy = RequireMachineSet, CreateMachineSets = nil, PreferRequireMachineSets = []string{requiredSets...}
//  4. Use any existing pools, provision machinesets
//     BindStrategy = Any, CreateMachineSets = []MachineSetSpec{newSets...}, PreferRequireMachineSets = nil
//  5. Prefer certain existing pools, provision machinesets (auto-prefers provisioned machinesets)
//     BindStrategy = PreferMachineSets, CreateMachineSets = []MachineSetSpec{newSets...},
//     PreferRequireMachineSets = []string{preferredSets... + newSets...}
//     (preferredSets may be nil, newSets must not be)
//  6. Require certain existing pools, provision machinesets (auto-requires provisioned machinesets)
//     BindStrategy = RequireMachineSets, CreateMachineSets = []MachineSetSpec{newSets...},
//     PreferRequireMachineSets = []string{requiredSets... + newSets...}
//     (requiredSets may be nil, newSets must not be)
type MachineProvisioningRequirement struct {
	MachineRequirement `json:",inline"`

	// BindStrategy determines how this MachineProvisioningRequirement shall be fulfilled.
	// BindStrategyAny = Fill this requirement from any matching machine set
	// BindStrategyPreferMachineSet = Fill this requirement first from a list of MachineSet, then from any matching machine set
	// BindStrategyRequireMachineSet = Only fill this requirement from a list of MachineSet
	BindStrategy BindStrategy `json:"bindStrategy"`

	// CreateMachineSet defines the spec of a MachineSet to be created for this request.
	CreateMachineSet *MachineSetSpec `json:"createMachineSet,omitempty"`

	// PreferRequireMachineSets is a slice of names of MachineSet objects that are either preferred or required,
	// depending on the value of BindStrategy.
	PreferRequireMachineSets []string `json:"preferRequireMachineSets"`
}

type ScheduledEventSpec struct {
	// DisplayName is the display name of the ScheduledEvent
	DisplayName string `json:"displayName"`

	// StartTime is the time at which the ScheduledEvent will become active. Active here means that users will be able
	// to begin accessing content and claiming machines.
	StartTime metav1.Time `json:"startTime"`

	// ProvisioningStartTime is the time at which HobbyFarm will begin to provision machines. By default this is set
	// to 30 minutes prior to StartTime. It may be adjusted if a provider or environment needs more than 30 minutes
	// notice to prepare the machines. Or, conversely, if no warm-up time is required.
	ProvisioningStartTime metav1.Time `json:"provisioningStartTime"`

	// EndTime is the time at which HobbyFarm will cut off access to the content and machines. After this time, users
	// will no longer be able to claim machines or access scenarios/courses.
	EndTime metav1.Time `json:"endTime"`

	// ExpirationStrategy defines the behavior that HobbyFarm will use when a ScheduledEvent passes its EndTime.
	// For example, if ExpirationStrategy is set to "CutOff", access to machines will be immediately terminated.
	// If ExpirationStrategy is set to "Soft", users will continue to have access while they are actively using
	// machines, but cannot create new machines or access other content than what they are currently using.
	ExpirationStrategy ExpirationStrategy `json:"expirationStrategy"`

	// RequiredMachines defines the number of machines of each MachineTemplate that are required for this
	// ScheduledEvent.
	RequiredMachines []MachineProvisioningRequirement `json:"requiredMachines"`

	// PrintingOption defines if users should be allowed to print out the content to which they have access.
	PrintingOption PrintingOption `json:"printingOption"` // PrintingEnabled, PrintingDisabled

	// Scenarios is a list of Scenario names to which this ScheduledEvent shall grant access
	Scenarios []string `json:"scenarios,omitempty"`

	// Courses is a list of Course names to which this ScheduledEvent shall grant access
	Courses []string `json:"courses,omitempty"`
}

type ScheduledEventStatus struct {
	// Conditions is an array of generic conditions that may crop up on this MachineSet.
	// There are NO guarantees made about the conditions that exist in this slice.
	Conditions []genericcondition2.GenericCondition `json:"conditions"`

	// CreatedMachineSets is a list of the names of MachineSet objects that have been created as a result
	// of this ScheduledEvent. Presence in this list does not imply presence in PreferRequireMachineSets.
	CreatedMachineSets []string `json:"createdMachineSets"`
}

type AccessCodeState string

const (
	AccessCodeInactive AccessCodeState = "Inactive"
	AccessCodeActive   AccessCodeState = "Active"
)

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// AccessCode stores a string which grants access to any number of Scenarios, Courses, ScheduledEvents, or MachineSets.
// It can be considered a sort of password, a pre-shared key.
type AccessCode struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   AccessCodeSpec   `json:"spec,omitempty"`
	Status AccessCodeStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type AccessCodeList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`

	Items []AccessCode `json:"items"`
}

type AccessCodeSpec struct {
	// Scenarios is a list of the names of scenarios to which this AccessCode shall grant access.
	Scenarios []string `json:"scenarios,omitempty"`

	// Courses is a list of the names of Courses to which this AccessCode shall grant access.
	Courses []string `json:"courses,omitempty"`

	// ScheduledEvents is a list of the names of ScheduledEvents to which this AccessCode shall grant access.
	ScheduledEvents []string `json:"scheduledEvents,omitempty"`

	// MachineSets is a list of the names of MachineSets to which this AccessCode shall grant access.
	MachineSets []string `json:"machineSets,omitempty"`

	// NotBefore is a timestamp that defines the point in time after which an AccessCode may be used.
	// If defined, the AccessCode may not be used before this timestamp. If nil, an AccessCode is considered valid
	// immediately after creation - provided NotAfter has not passed if set.
	NotBefore *metav1.Time `json:"notBefore,omitempty"`

	// NotAfter is a timestamp that defines the point in time after which an AccessCode may NOT be used.
	// If defined, the AccessCode may not be used after this timestamp. If nil, an AccessCode is considered valid
	// forever - provided NotBefore has passed if set.
	NotAfter *metav1.Time `json:"notAfter,omitempty"`
}

type AccessCodeStatus struct {
	// Status is the state of the access code.
	// AccessCodes can be AccessCodeActive or AccessCodeInactive, depending on the values of NotBefore & NotAfter,
	// as well as the current time.
	Status AccessCodeState `json:"status"`
}

type MachinePersistenceStrategy string

const (
	PersistThroughCourse MachinePersistenceStrategy = "PersistThroughCourse"
	NewPerScenario       MachinePersistenceStrategy = "NewPerScenario"
)

const (
	ConditionActive   = condition.Cond("Active")
	ConditionInactive = condition.Cond("Inactive")
	ConditionPaused   = condition.Cond("Paused")
)

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// Session holds the information used to track a user's engagement with HobbyFarm on a particular
// Scenario or Course. It contains information such as the step that the user is on, what
// machines are assigned to them, when they started, etc.
type Session struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   SessionSpec   `json:"spec"`
	Status SessionStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type SessionList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`

	Items []Session `json:"items"`
}

type SessionSpec struct {
	// Scenario is the object name of the scenario that this session is tracking.
	Scenario string `json:"scenario,omitempty"`

	// Course is the object name of the course that this session is tracking.
	Course string `json:"course,omitempty"`

	// PersistenceStrategy defines what shall happen to the machines being used
	// in this session after the scenario has ended.
	PersistenceStrategy MachinePersistenceStrategy `json:"persistenceStrategy"`

	// User is the object name of the user to whom this Session belongs.
	User string `json:"user"`

	// AccessCode is the code (not obj name) used to get access to this scenario.
	// May be empty if access code was not used.
	AccessCode string `json:"accessCode,omitempty"`

	// ScheduledEvent is the object name of the scheduled event from which
	// this Session was started.
	ScheduledEvent string `json:"scheduledEvent,omitempty"`
}

type SessionStatus struct {
	// Conditions is a slice of conditions that may impact this session.
	Conditions []genericcondition2.GenericCondition `json:"conditions"`

	// MachineClaim is the object name of the MachineClaim generated from this
	// session.
	MachineClaim string `json:"machineClaim"`

	// Progress is a slice of Progress structs that detail, for a given scenario, what
	// position the user is in.
	Progress []Progress `json:"progress"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// Progress holds the progress of a user through a particular scenario. It tracks items such as
// the step the user is currently on, when they started and ended, etc.
type Progress struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ProgressSpec   `json:"spec"`
	Status ProgressStatus `json:"status"`
}

type ProgressSpec struct {
	// User is the object name of the user to whom this Progress belongs
	User string `json:"user"`

	// Scenario is the object name of the scenario to which this Progress
	// relates.
	Scenario string `json:"scenario"`

	// Course is the object name of the course to which this Progress relates.
	Course string `json:"course"`

	// TotalStep is the total number of steps in the Scenario. It is copied into
	// this struct to make visualizations and reasoning about step progress easier.
	TotalStep int `json:"totalStep"`
}

type ProgressStatus struct {
	// CurrentStep is the 0-indexed step of the scenario that the user
	// is currently on, or was on as of LastUpdate.
	CurrentStep int `json:"currentStep"`

	// MaxStep is the 0-indexed *furthest* step the user has gotten to.
	// May not be the current step the user is on, if they have backtracked.
	MaxStep int `json:"maxStep"`

	// Started is the timestamp when the user started the scenario.
	Started metav1.Time `json:"started"`

	// LastUpdate is the timestamp when the user interface last updated this struct.
	LastUpdate metav1.Time `json:"lasteUpdate"`

	// Finished is the timestamp when the user finished the scenario.
	Finished metav1.Time `json:"finished"`

	// StepTimes stores the timestamps when a user entered a particular step.
	StepTimes []StepTime `json:"stepTimes"`
}

type StepTime struct {
	// Step is the number of the step
	Step int `json:"step"`

	// Time is the time at which the step was "started" (e.g. loaded in the UI)
	Time metav1.Time `json:"time"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type ProgressList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	Items []Progress `json:"items"`
}

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

	// AccessCodes is a slice of all access codes that this user has entered. These
	// values will be the codes themselves, *NOT* the AccessCode object names.
	AccessCodes []string `json:"accessCodes"`

	// Settings holds the settings that the user has configured. Things like terminal theme,
	// dark mode, etc.
	Settings map[string]string `json:"settings"`
}

type UserStatus struct {
	// LastLoginTimestamp is the timestamp of when the user last logged in.
	LastLoginTimestamp metav1.Time `json:"lastLoginTimestamp"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// ServiceAccount functions largely the same as a Kubernetes SA. It is an account that
// services can use to authenticate with HobbyFarm.
//
// "Why not use k8s corev1 ServiceAccount?"
// Because if k8s is used as a backing store for HobbyFarm, there is a collision with the
// HF api server instance of ServiceAccount and the backing store. The k8s cluster storing the data
// will start operating on that ServiceAccount by generating tokens (that wouldn't be valid for
// the HF apiserver, only for that k8s cluster), or giving it access to things in that k8s cluster.
type ServiceAccount struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// Secrets is a list of secret object names that contain the tokens used to authenticate
	// a ServiceAccount to the HobbyFarm apiserver.
	Secrets []string `json:"secrets"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// PredefinedService represents a service (as in application, or web service) that is
// hosted on a Machine. Predefined
type PredefinedService struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec PredefinedServiceSpec `json:"spec"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type PredefinedServiceList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	Items []PredefinedService `json:"items"`
}

type DisplayOption string

const (
	// HasTab determines if a service gets its own tab in the UI
	HasTab DisplayOption = "HasTab"

	// HasWebInterface determines if a service does or does not have a web interface
	HasWebInterface DisplayOption = "HasWebInterface"
)

type HttpOption string

const (
	// NoRewriteRootPath disables path rewriting from /p/[vmid]/80/path to /path
	NoRewriteRootPath HttpOption = "NoRewriteRootPath"

	// RewriteHostHeader rewrites the host header to the proxy server host
	RewriteHostHeader HttpOption = "RewriteHostHeader"

	// RewriteOriginHeader rewrites the origin to localhost instead of the proxy host
	RewriteOriginHeader HttpOption = "RewriteOriginHeader"

	// DisallowIframe forces opening the service content in a new browser tab instead of iframe
	DisallowIframe HttpOption = "DisallowIframe"
)

type PredefinedServiceSpec struct {
	// DisplayName is the display (pretty) name of the PredefinedService
	DisplayName string `json:"displayName"`

	// Port is the network port of the service
	Port int `json:"port"`

	// DisplayOptions is a list of display (ui) options that this service requires.
	DisplayOptions []DisplayOption `json:"displayOptions"`

	// HttpOptions is a list of http options that this service requires.
	HttpOptions []HttpOption `json:"httpOptions"`

	// Path is the path on the VM that the service is accessible upon
	Path string `json:"path"`

	// CloudConfig contains the cloud-config data used to setup this service on the machine
	CloudConfig string `json:"cloudConfig"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type Course struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec CourseSpec `json:"spec"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type CourseList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	Items []Course `json:"items"`
}

type PauseBehavior string

const (
	// CanPause means a user CAN pause their course/scenario
	CanPause PauseBehavior = "CanPause"

	// CannotPause means a user CANNOT pause their course/scenario
	CannotPause PauseBehavior = "CannotPause"
)

type CourseSpec struct {
	// DisplayName is the display (pretty) name of the course.
	DisplayName string `json:"displayName"`

	// Description is the string description of the course.
	Description string `json:"description"`

	// Scenarios is a slice of object names for scenarios that are in this course.
	Scenarios []string `json:"scenarios"`

	// Categories is a slice of categories (strings) that this course fits into.
	// Categories may be used to filter Course objects in interfaces.
	Categories []string `json:"categories"`

	// Tags is a slice of strings that represent tags applied to this course.
	// Tags may be used to filter Course objects in interfaces.
	Tags []string `json:"tags"`

	// MachineRequirements is a slice of MachineRequirement structs that define
	// what machines are required for this Course.
	MachineRequirements []MachineRequirement `json:"machineRequirements"`

	// KeepaliveDuration is a string representing the period of time a user can go inactive
	// before their machines will be reaped. It can be viewed as a countdown timer that
	// resets each time the user interface submits a keepalive ping to the API.
	// For example if this string is "1h", machines will be reaped one hour after the
	// last keepalive ping is received. If a ping is received before the 1h elapses, the timer
	// starts over with a fresh 1h period.
	//
	// The format of this field is an integer followed by an 'h' or 'm' designation for hours or
	// minutes, respectively. MUST be parseable by time.ParseDuration()
	// Examples include "15m", "5h", "48h", or "120m"
	KeepaliveDuration string `json:"keepaliveDuration"`

	// PauseDuration is a string representing the period of time a user can prevent the reaping of
	// their machines without continuous keepalive pings. While a Course is paused, the keepalive "timer"
	// (described on KeepaliveDuration) is paused and HobbyFarm will not reclaim the Machine.
	// For example, if this string is "1h" and KeepaliveDuration is "30m", if a user pauses their
	// Course and closes their laptop (thus preventing keepalive pings from the UI), the KeepaliveDuration
	// timer will not start until one hour after the Course was paused. Even if 30 minutes elapses
	// machines will not be reaped as the countdown for keepalive is paused for the duration of the pause.
	// A user may un-pause at any time.
	//
	// The format of this field is an integer followed by an 'h' or 'm' designation for hours or minutes,
	// respectively. MUST be parseabl by time.ParseDuration()
	// Examples include "15m", "5h", "48h", or "120m"
	PauseDuration string `json:"pauseDuration,omitempty"`

	// MachinePersistenceStrategy determines what HobbyFarm should do with the Machine objects provisioned
	// for this Course after a particular Scenario has been completed.
	// When a user completes a Scenario that is *NOT* part of a Course, their Machine resources are
	// reclaimed and recycled for the next user. However, Courses may have multiple Scenarios whose content
	// (and thus, machines) flow together. By setting a persistence strategy of "PersistThroughCourse", machines
	// will *NOT* be reclaimed after a Scenario is completed - they will live for the life of the Course.
	// Conversely, if a persistence strategy is set to "NewPerScenario", old machines will be reclaimed
	// (and new ones provisioned/claimed) for each scenario.
	MachinePersistenceStrategy MachinePersistenceStrategy `json:"machinePersistenceStrategy"`

	// PauseBehavior describes what sort of behavior affecting pausing should be allowed for this Course.
	// An example is "CanPause" which allowed users to pause their course. "CannotPause" does not.
	PauseBehavior PauseBehavior `json:"pauseBehavior"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// Scenario represents a unit of learning on a particular topic represented by a series of steps that
// may include interactions with Machine(s). While defined on a separate resource, Steps are where the
// content of a Scenario lives. A Scenario also tracks requirements and configurations for Machines,
// allowed behavior of users, categorization/tagging metadata, and so on.
type Scenario struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec ScenarioSpec `json:"spec"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type ScenarioList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	Items []Scenario `json:"items"`
}

type ScenarioSpec struct {
	// DisplayName is the display (pretty) name of the Scenario
	DisplayName string `json:"string"`

	// Description is the description of the Scenario.
	Description string `json:"description"`

	// Categories is a string slice listing the categories in which this Scenario exists.
	Categories []string `json:"categories"`

	// Tags is a string slice listing the tags that are applicable to this Scenario.
	Tags []string `json:"tags"`

	// Steps is a string slice of object names for Step resources.
	// The order in which they appear in this slice is the order in which they
	// are presented.
	Steps []string `json:"steps"`

	// MachineRequirements details the required machines for this Scenario. This includes both
	// user machines and shared machines.
	MachineRequirements []MachineRequirement `json:"machineRequirements"`

	// KeepaliveDuration is a string representing the period of time a user can go inactive
	// before their machines will be reaped. It can be viewed as a countdown timer that
	// resets each time the user interface submits a keepalive ping to the API.
	// For example if this string is "1h", machines will be reaped one hour after the
	// last keepalive ping is received. If a ping is received before the 1h elapses, the timer
	// starts over with a fresh 1h period.
	//
	// The format of this field is an integer followed by an 'h' or 'm' designation for hours or
	// minutes, respectively. MUST be parseable by time.ParseDuration()
	// Examples include "15m", "5h", "48h", or "120m"
	KeepaliveDuration string `json:"keepaliveDuration"`

	// PauseDuration is a string representing the period of time a user can prevent the reaping of
	// their machines without continuous keepalive pings. While a Course is paused, the keepalive "timer"
	// (described on KeepaliveDuration) is paused and HobbyFarm will not reclaim the Machine.
	// For example, if this string is "1h" and KeepaliveDuration is "30m", if a user pauses their
	// Course and closes their laptop (thus preventing keepalive pings from the UI), the KeepaliveDuration
	// timer will not start until one hour after the Course was paused. Even if 30 minutes elapses
	// machines will not be reaped as the countdown for keepalive is paused for the duration of the pause.
	// A user may un-pause at any time.
	//
	// The format of this field is an integer followed by an 'h' or 'm' designation for hours or minutes,
	// respectively. MUST be parseabl by time.ParseDuration()
	// Examples include "15m", "5h", "48h", or "120m"
	PauseDuration string `json:"pauseDuration"`

	// PauseBehavior describes what sort of behavior affecting pausing should be allowed for this Course.
	// An example is "CanPause" which allowed users to pause their course. "CannotPause" does not.
	PauseBehavior PauseBehavior `json:"pauseBehavior"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// ScenarioStep holds the content (and title) of a step in one or more scenarios.
type ScenarioStep struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ScenarioStepSpec   `json:"spec"`
	Status ScenarioStepStatus `json:"status"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type ScenarioStepList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	Items []ScenarioStep `json:"items"`
}

type ScenarioStepSpec struct {
	// Title is the base64-encoded title of a Step.
	Title string `json:"title"`

	// Content is the base64-encoded content of a Step.
	Content string `json:"content"`
}

type ScenarioStepStatus struct {
	// ReferringScenarios is a list of all the Scenario objects that reference this step.
	ReferringScenarios []v1.ObjectReference `json:"referringScenarios"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// OneTimeAccessCode is the representation of a single-use access code
type OneTimeAccessCode struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   OneTimeAccessCodeSpec   `json:"spec"`
	Status OneTimeAccessCodeStatus `json:"status"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type OneTimeAccessCodeList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	Items []OneTimeAccessCode `json:"items"`
}

type OneTimeAccessCodeSpec struct {
	// NotBefore is the timestamp before which a OneTimeAccessCode is invalid.
	NotBefore *metav1.Time `json:"notBefore"`

	// NotAfter is the timestamp after which a OneTimeAccessCode is invalid.
	NotAfter *metav1.Time `json:"notAfter"`

	// User is the object name of the user for whom the OneTimeAccessCode is intended.
	User string `json:"user"`
}

type OneTimeAccessCodeStatus struct {
	// Redeemed is the timestamp marking when the OneTimeAccessCode was redeemed.
	// nil if not redeemed.
	Redeemed *metav1.Time `json:"redeemed,omitempty"`
}

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

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// Scope is a construct that provides grouping of Setting resources and their visibility in various
// UIs.
type Scope struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// DisplayName is the pretty name of the Scope
	DisplayName string `json:"displayName"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type ScopeList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	Items []Scope `json:"items"`
}
