package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type VirtualMachine struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              VirtualMachineSpec `json:"spec"`
	Status			VirtualMachineStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type VirtualMachineList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`
	Items           []VirtualMachine `json:"items"`
}

type VirtualMachineSpec struct {
	VMType			string		`json:"vmtype"`
	KeyPair			string		`json:"keypair_name"` // this refers to the secret name for the keypair
}

type VirtualMachineStatus struct {
	Status				string		`json:"status"` // default is nothing, but could be one of the following: starting, running, stopped, terminated
	Allocated			bool		`json:"allocated"`
	ActiveScenarioID	string		`json:"active_scenario_id"` // should only be populated when `allocated:true`
	PublicIP			string	`json:"public_ip"`
	PrivateIP			string	`json:"private_ip"`
	Hostname			string		`json:"hostname"` // ideally <hostname>.<enviroment dnssuffix> should be the FQDN to this machine
}

// +genclient
// +genclient:noStatus
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type Environment struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              EnvironmentSpec `json:"spec"`
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
	DisplayName				string	`json:"display_name"`
	DNSSuffix   			string	`json:"dnssuffix"`
	Provider				string	`json:"provider"` // aws,vsphere,azure,custom ;)
	VsphereEnvironmentSpec	VsphereEnvironmentSpec `json:"vsphere_env_spec,omitempty"`
	AWSEnvironmentSpec 		AWSEnvironmentSpec `json:"aws_env_spec,omitempty"`
}

type VsphereEnvironmentSpec struct {
	Credentials string `json:"credentials"` // string that refers to the secret name for vSphere environment credentials
}

type AWSEnvironmentSpec struct {
	Credentials 		string `json:"credentials"`
	Region      		string `json:"region"`
	AvailabilityZone	string `json:"az"`
	VPC					string	`json:"vpc"`
	Subnet				string	`json:"subnet"`
	TypeMapping			map[string]string `json:"type_mapping"`
	// @todo: finish filling this in
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
	Name string `json:"name"`
	Description string `json:"description"`
}


// +genclient
// +genclient:noStatus
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type ActiveScenario struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              ActiveScenarioSpec `json:"spec"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type ActiveScenarioList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`
	Items           []ActiveScenario `json:"items"`
}

type ActiveScenarioSpec struct {
	Scenario	string	`json:"scenario"`
	User		string	`json:"user"`
}

type ActiveScenarioStatus struct {
	StartTime	string	`json:"start_time"`
	ExpirationTime	string	`json:"end_time"`
}

// +genclient
// +genclient:noStatus
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type VirtualMachineType struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              VirtualMachineTypeSpec `json:"spec"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type VirtualMachineTypeList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`
	Items           []VirtualMachineType `json:"items"`
}

// VM type is a genercized collection of information about a VM. this includes things like
// cpu, ram, disk, etc.
type VirtualMachineTypeSpec struct {
	Name	string	`json:"name"` // 2x4, etc.
	Image 	string	`json:"image"` // ubuntu-18.04
	CPU		int		`json:"cpu"`
	Memory	int		`json:"memory"`
}

// +genclient
// +genclient:noStatus
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

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
	Code	string	`json:"code"`
	Description string `json:"description"`
	Scenarios		[]string	`json:"scenarios"`
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
	Email string `json:"email"`
	Password string `json:"password"`
	AccessCodes []string `json:"access_codes"`
}