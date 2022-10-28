package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type VmStatus string
type CapacityMode string

const (
	VmStatusRFP         VmStatus     = "readyforprovisioning"
	VmStatusProvisioned VmStatus     = "provisioned"
	VmStatusRunning     VmStatus     = "running"
	VmStatusTerminating VmStatus     = "terminating"
	CapacityModeRaw     CapacityMode = "raw"
	CapacityModeCount   CapacityMode = "count"
)

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type VirtualMachine struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              VirtualMachineSpec   `json:"spec"`
	Status            VirtualMachineStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type VirtualMachineList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`
	Items           []VirtualMachine `json:"items"`
}

type VirtualMachineSpec struct {
	Id                       string `json:"id"`
	VirtualMachineTemplateId string `json:"vm_template_id"`
	SshUsername              string `json:"ssh_username"`
	Protocol                 string `json:"protocol"`
	SecretName               string `json:"secret_name"` // this refers to the secret name for the keypair
	VirtualMachineClaimId    string `json:"vm_claim_id"`
	UserId                   string `json:"user"`
	Provision                bool   `json:"provision"`
	VirtualMachineSetId      string `json:"vm_set_id"`
}

type VirtualMachineStatus struct {
	Status        VmStatus `json:"status"` // default is nothing, but could be one of the following: readyforprovisioning, provisioning, running, terminating
	Allocated     bool     `json:"allocated"`
	Tainted       bool     `json:"tainted"`
	PublicIP      string   `json:"public_ip"`
	PrivateIP     string   `json:"private_ip"`
	EnvironmentId string   `json:"environment_id"`
	Hostname      string   `json:"hostname"`          // ideally <hostname>.<enviroment dnssuffix> should be the FQDN to this machine
	TFState       string   `json:"tfstate,omitempty"` // Terraform state name
	WsEndpoint    string   `json:"ws_endpoint"`
}

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type VirtualMachineClaim struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              VirtualMachineClaimSpec   `json:"spec"`
	Status            VirtualMachineClaimStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type VirtualMachineClaimList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`
	Items           []VirtualMachineClaim `json:"items"`
}

type VirtualMachineClaimSpec struct {
	Id                  string                           `json:"id"`
	UserId              string                           `json:"user"`
	RestrictedBind      bool                             `json:"restricted_bind"`
	RestrictedBindValue string                           `json:"restricted_bind_value"`
	VirtualMachines     map[string]VirtualMachineClaimVM `json:"vm"`
	DynamicCapable      bool                             `json:"dynamic_bind_capable"`
	BaseName            string                           `json:"base_name"`
}

type VirtualMachineClaimStatus struct {
	BindMode             string `json:"bind_mode"`
	StaticBindAttempts   int    `json:"static_bind_attempts"`
	DynamicBindRequestId string `json:"dynamic_bind_request_id"`
	Bound                bool   `json:"bound"`
	Ready                bool   `json:"ready"`
	Tainted              bool   `json:"tainted"` // If tainted, we should delete the VM's underneath then delete ourself...
}

type VirtualMachineClaimVM struct {
	Template         string `json:"template"`
	VirtualMachineId string `json:"vm_id"`
}

// +genclient
// +genclient:noStatus
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type VirtualMachineTemplate struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              VirtualMachineTemplateSpec `json:"spec"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type VirtualMachineTemplateList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`
	Items           []VirtualMachineTemplate `json:"items"`
}

// VM type is a genercized collection of information about a VM. this includes things like
// cpu, ram, disk, etc.
type VirtualMachineTemplateSpec struct {
	Id        string            `json:"id"`
	Name      string            `json:"name"`  // 2x4, etc.
	Image     string            `json:"image"` // ubuntu-18.04
	Resources CMSStruct         `json:"resources"`
	ConfigMap map[string]string `json:"config_map"`
}

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type Environment struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              EnvironmentSpec   `json:"spec"`
	Status            EnvironmentStatus `json:"status"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type EnvironmentList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`
	Items           []Environment `json:"items"`
}

// environment is to be like
// AWS-us-west-2
// hermes
type EnvironmentSpec struct {
	DisplayName          string                       `json:"display_name"`
	DNSSuffix            string                       `json:"dnssuffix"`
	Provider             string                       `json:"provider"`         // aws,vsphere,azure,custom ;)
	TemplateMapping      map[string]map[string]string `json:"template_mapping"` //  lol
	EnvironmentSpecifics map[string]string            `json:"environment_specifics"`
	IPTranslationMap     map[string]string            `json:"ip_translation_map"`
	WsEndpoint           string                       `json:"ws_endpoint"`
	CapacityMode         CapacityMode                 `json:"capacity_mode"`
	BurstCapable         bool                         `json:"burst_capable"`
	CountCapacity        map[string]int               `json:"count_capacity"`
	Capacity             CMSStruct                    `json:"capacity"`
	BurstCountCapacity   map[string]int               `json:"burst_count_capacity"`
	BurstCapacity        CMSStruct                    `json:"burst_capacity"`
}

type EnvironmentStatus struct {
	Used           CMSStruct      `json:"used"`
	AvailableCount map[string]int `json:"available_count"`
}

type CMSStruct struct {
	CPU     int `json:"cpu"`     // cores
	Memory  int `json:"memory"`  // in MB
	Storage int `json:"storage"` // in GB
}

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type VirtualMachineSet struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              VirtualMachineSetSpec   `json:"spec"`
	Status            VirtualMachineSetStatus `json:"status"`
}

type VirtualMachineSetSpec struct {
	Count               int    `json:"count"`
	Environment         string `json:"environment"`
	VMTemplate          string `json:"vm_template"`
	BaseName            string `json:"base_name"`
	RestrictedBind      bool   `json:"restricted_bind"`
	RestrictedBindValue string `json:"restricted_bind_value"`
}

type VirtualMachineSetStatus struct {
	Machines         []VirtualMachineProvision `json:"machines"`
	AvailableCount   int                       `json:"available"`
	ProvisionedCount int                       `json:"provisioned"`
}

type VirtualMachineProvision struct {
	VirtualMachineName string `json:"vm_name"`
	TFControllerState  string `json:"tfc_state"`
	TFControllerCM     string `json:"tfc_cm"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type VirtualMachineSetList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []VirtualMachineSet `json:"items"`
}

// +genclient
// +genclient:noStatus
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type Course struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              CourseSpec `json:"spec"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type CourseList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`
	Items           []Course `json:"items"`
}

type CourseSpec struct {
	Id                string              `json:"id"`
	Name              string              `json:"name"`
	Description       string              `json:"description"`
	Scenarios         []string            `json:"scenarios"`
	Categories        []string            `json:"categories"`
	VirtualMachines   []map[string]string `json:"virtualmachines"`
	KeepAliveDuration string              `json:"keepalive_duration"`
	PauseDuration     string              `json:"pause_duration"`
	Pauseable         bool                `json:"pauseable"`
	KeepVM            bool                `json:"keep_vm"`
}

// +genclient
// +genclient:noStatus
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type Scenario struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              ScenarioSpec `json:"spec"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type ScenarioList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`
	Items           []Scenario `json:"items"`
}

type ScenarioSpec struct {
	Id                string              `json:"id"`
	Name              string              `json:"name"`
	Description       string              `json:"description"`
	Steps             []ScenarioStep      `json:"steps"`
	Categories        []string            `json:"categories"`
	Tags              []string            `json:"tags"`
	VirtualMachines   []map[string]string `json:"virtualmachines"`
	KeepAliveDuration string              `json:"keepalive_duration"`
	PauseDuration     string              `json:"pause_duration"`
	Pauseable         bool                `json:"pauseable"`
}

type ScenarioStep struct {
	Title   string `json:"title"`
	Content string `json:"content"`
}

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type Session struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              SessionSpec   `json:"spec"`
	Status            SessionStatus `json:"status"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type SessionList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`
	Items           []Session `json:"items"`
}

type SessionSpec struct {
	Id           string   `json:"id"`
	ScenarioId   string   `json:"scenario"`
	CourseId     string   `json:"course"`
	KeepCourseVM bool     `json:"keep_course_vm"`
	UserId       string   `json:"user"`
	VmClaimSet   []string `json:"vm_claim"`
	AccessCode   string   `json:"access_code"`
}

type SessionStatus struct {
	Paused         bool   `json:"paused"`
	PausedTime     string `json:"paused_time"`
	Active         bool   `json:"active"`
	Finished       bool   `json:"finished"`
	StartTime      string `json:"start_time"`
	ExpirationTime string `json:"end_time"`
}

// +genclient
// +genclient:noStatus
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type Progress struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              ProgressSpec `json:"spec"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type ProgressList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`
	Items           []Progress `json:"items"`
}

type ProgressSpec struct {
	Id          string         `json:"id"`
	CurrentStep int            `json:"current_step"`
	MaxStep     int            `json:"max_step"`
	TotalStep   int            `json:"total_step"`
	Course      string         `json:"course"`
	Scenario    string         `json:"scenario"`
	UserId      string         `json:"user"`
	Started     string         `json:"started"`
	LastUpdate  string         `json:"last_update"`
	Finished    string         `json:"finished"`
	Steps       []ProgressStep `json:"steps"`
}

type ProgressStep struct {
	Step      int    `json:"step"`
	Timestamp string `json:"timestamp"`
}

// +genclient
// +genclient:noStatus
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// AccessCode is used for defining access code for scheduled events

type AccessCode struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              AccessCodeSpec `json:"spec"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type AccessCodeList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`
	Items           []AccessCode `json:"items"`
}

type AccessCodeSpec struct {
	Code                string   `json:"code"`
	Description         string   `json:"description"`
	Scenarios           []string `json:"scenarios"`
	Courses             []string `json:"courses"`
	Expiration          string   `json:"expiration"`
	VirtualMachineSets  []string `json:"vmsets"`
	RestrictedBind      bool     `json:"restricted_bind"`
	RestrictedBindValue string   `json:"restricted_bind_value"`
	Printable           bool     `json:"printable"`
}

// +genclient
// +genclient:noStatus
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type User struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              UserSpec `json:"spec"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type UserList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`
	Items           []User `json:"items"`
}

type UserSpec struct {
	Id          string            `json:"id"`
	Email       string            `json:"email"`
	Password    string            `json:"password"`
	AccessCodes []string          `json:"access_codes"`
	Admin       bool              `json:"admin"`
	Settings    map[string]string `json:"settings"`
}

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type ScheduledEvent struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              ScheduledEventSpec   `json:"spec"`
	Status            ScheduledEventStatus `json:"status"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type ScheduledEventList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`
	Items           []ScheduledEvent `json:"items"`
}

type ScheduledEventSpec struct {
	Creator                 string                    `json:"creator"`
	Name                    string                    `json:"event_name"`
	Description             string                    `json:"description"`
	StartTime               string                    `json:"start_time"`
	EndTime                 string                    `json:"end_time"`
	OnDemand                bool                      `json:"on_demand"`    // whether or not to provision VMs on-demand
	RequiredVirtualMachines map[string]map[string]int `json:"required_vms"` // map of environment to a map of strings it should be environment: vm template: count
	AccessCode              string                    `json:"access_code"`
	RestrictedBind          bool                      `json:"restricted_bind"` // if restricted_bind is true, we need to make the scenario sessions when they get created only bind to vmsets that are created by this scheduledevent
	RestrictedBindValue     string                    `json:"restricted_bind_value"`
	Printable               bool                      `json:"printable"`
	Scenarios               []string                  `json:"scenarios"`
	Courses                 []string                  `json:"courses"`
}

type ScheduledEventStatus struct {
	VirtualMachineSets []string `json:"vmsets"`
	Active             bool     `json:"active"`
	Provisioned        bool     `json:"provisioned"`
	Ready              bool     `json:"ready"`
	Finished           bool     `json:"finished"`
}

// +genclient
// +genclient:noStatus
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type DynamicBindConfiguration struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              DynamicBindConfigurationSpec `json:"spec"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type DynamicBindConfigurationList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`
	Items           []DynamicBindConfiguration `json:"items"`
}

// DynamicBindConfiguration is very similar to a VirtualMachineSet. They should be created side-by-side
// but there is no guarantee the environments will have adequate capacity when creating them.
// The DynamicBindController will watch for VM Objects that get created and will always check to make sure
// that there is adequate vm capacity, it will always choose the environment with the highest capacity before creating a dynamic VM.

type DynamicBindConfigurationSpec struct {
	Id                  string         `json:"id"`
	Environment         string         `json:"environment"`
	BaseName            string         `json:"base_name"`
	RestrictedBind      bool           `json:"restricted_bind"`
	RestrictedBindValue string         `json:"restricted_bind_value"`
	BurstCountCapacity  map[string]int `json:"burst_count_capacity"`
	BurstCapacity       CMSStruct      `json:"burst_capacity"`
}

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type DynamicBindRequest struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              DynamicBindRequestSpec   `json:"spec"`
	Status            DynamicBindRequestStatus `json:"status"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type DynamicBindRequestList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`
	Items           []DynamicBindRequest `json:"items"`
}

type DynamicBindRequestSpec struct {
	Id                  string `json:"id"`
	VirtualMachineClaim string `json:"vm_claim"`
	Attempts            int    `json:"attempts"`
}

type DynamicBindRequestStatus struct {
	CurrentAttempts            int               `json:"current_attempts"`
	Expired                    bool              `json:"expired"`
	Fulfilled                  bool              `json:"fulfilled"`
	DynamicBindConfigurationId string            `json:"dynamic_bind_configuration_id"`
	VirtualMachineIds          map[string]string `json:"virtual_machines_id"`
}
