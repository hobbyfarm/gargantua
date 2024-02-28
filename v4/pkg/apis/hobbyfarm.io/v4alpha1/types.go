// +k8s:deepcopy-gen=package

package v4alpha1

import (
	"github.com/hobbyfarm/gargantua/v3/pkg/property"
	"github.com/rancher/wrangler/pkg/condition"
	"github.com/rancher/wrangler/pkg/genericcondition"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Provider is an object that represents the capability to provision machines.
// Providers are usually implemented as 3rd party operators that interact with HobbyFarm through
// the boundaries of objects such as Machines, MachineSets, and Environments.
type Provider struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
}

type ProviderList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`

	Items []Provider `json:"items"`
}

type ProviderSpec struct {
	// ConfigurationItems describes properties that an administrator can configure for the provider
	// For example, an AWS provider may have an item for "AMI", or perhaps "SecurityGroup".
	// This array does not set these properties - it merely defines them for use in
	// resources such as Environments, or MachineTemplates.
	// The presence of a property.Property in this array MAY drive UI elements such as configuration menus.
	ConfigurationItems []property.Property `json:"configurationItems,omitempty"`

	// MachineInformation describes properties that the provider CAN return about a machine.
	// For example, an AWS provider may have an item for "public ip", or perhaps "hostname".
	// This array does not set these properties - it merely defines them for use in
	// the MachineStatus struct. The presence of a property.Property in this array
	// MAY drive UI elements such as variable introspection when writing Scenarios.
	MachineInformation []property.Property `json:"machineInformation,omitempty"`
}

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
}

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
	Conditions []genericcondition.GenericCondition
}

// MachineSet defines a collection of machines that are scheduled together.
// All machines in a MachineSet are of identical configuration, MachineTemplate, etc.
// The settings on the MachineSetSpec struct define how a provider should provision these machines.
type MachineSet struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   MachineSetSpec   `json:"spec"`
	Status MachineSetStatus `json:"status,omitempty"`
}

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
	Conditions []genericcondition.GenericCondition

	// Provisioned is the current number of machines that are provisioned in this MachineSet.
	// This is a _total_ that includes both claimed and unclaimed machines.
	Provisioned int `json:"provisioned"`

	// Available is the current number of machines that are provisioned _and unclaimed_ in this MachineSet.
	Available int `json:"available"`
}

// Machine is the record of an instance of a MachineTemplate as provisioned via a Provider into an Environment.
// For example, a Machine should have a 1:1 correlation with e.g. an EC2 instance or a DigitalOcean droplet.
// The Machine object carries the configuration from HF and the status from the Provider of the machine.
type Machine struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   MachineSpec   `json:"spec"`
	Status MachineStatus `json:"status,omitempty"`
}

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
	Conditions []genericcondition.GenericCondition

	// MachineInformation is a map of information about this Machine as reported by the Provider.
	// Keys in this map shall be those defined in the MachineInformation field on the Provider. Values
	// shall be the string representations of those properties.
	MachineInformation map[string]string `json:"machineInformation"`
}

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
	MachineClaimPhaseRequested   MachineClaimPhase = "Requested"
	MachineClaimPhaseBound       MachineClaimPhase = "Bound"
	MachineClaimPhaseFailed      MachineClaimPhase = "Failed"
	MachineClaimPhaseTerminating MachineClaimPhase = "Terminated"
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

	// Conditions is an array of generic conditions that may crop up on this MachineSet.
	// There are NO guarantees made about the conditions that exist in this slice.
	Conditions []genericcondition.GenericCondition
}

// ScheduledEvent is the representation of a period of time during which content is available to users.
// A ScheduledEvent not only defines what content is available but also how users shall have access to
// Machine resources during that period of time.
type ScheduledEvent struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ScheduledEventSpec   `json:"spec"`
	Status ScheduledEventStatus `json:"status,omitempty"`
}

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

// MachineRequirement defines for a given MachineType and MachineTemplate, how many of that kind of machine
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
type MachineRequirement struct {
	// MachineTemplate is the name of the required MachineTemplate
	MachineTemplate string `json:"machineTemplate"`

	// Count is the number of required Machines either per-User (when MachineType = MachineTypeUser)
	// or per- ScheduledEvent (when MachineType = MachineTypeShared)
	Count int `json:"count"`

	// MachineType is the type of machine to require
	MachineType MachineType `json:"machineType"`

	// BindStrategy determines how this MachineRequirement shall be fulfilled.
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
	RequiredMachines []MachineRequirement `json:"requiredMachines"`

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
	Conditions []genericcondition.GenericCondition `json:"conditions"`

	// CreatedMachineSets is a list of the names of MachineSet objects that have been created as a result
	// of this ScheduledEvent. Presence in this list does not imply presence in PreferRequireMachineSets.
	CreatedMachineSets []string `json:"createdMachineSets"`
}

type AccessCodeState string

const (
	AccessCodeInactive AccessCodeState = "Inactive"
	AccessCodeActive   AccessCodeState = "Active"
)

// AccessCode stores a string which grants access to any number of Scenarios, Courses, ScheduledEvents, or MachineSets.
// It can be considered a sort of password, a pre-shared key.
type AccessCode struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   AccessCodeSpec   `json:"spec,omitempty"`
	Status AccessCodeStatus `json:"status,omitempty"`
}

type AccessCodeList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`

	Items []AccessCode `json:"items"`
}

type AccessCodeSpec struct {
	// Code is the actual access code.
	Code string `json:"code"`

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

type Session struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   SessionSpec   `json:"spec"`
	Status SessionStatus `json:"status,omitempty"`
}

type SessionList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`

	Items []Session `json:"items"`
}

type SessionSpec struct {
	Scenario            string                     `json:"scenario,omitempty"`
	Course              string                     `json:"course,omitempty"`
	PersistenceStrategy MachinePersistenceStrategy `json:"persistenceStrategy"`
	User                string                     `json:"user"`
	AccessCode          string                     `json:"accessCode,omitempty"`
	ScheduledEvent      string                     `json:"scheduledEvent,omitempty"`
}

type SessionStatus struct {
	Conditions   []genericcondition.GenericCondition `json:"conditions"`
	MachineClaim string                              `json:"machineClaim"`
}
