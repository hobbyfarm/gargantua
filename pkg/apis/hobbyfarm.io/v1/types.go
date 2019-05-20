package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"net"
	"time"
)

type VirtualMachineSpec struct {
	VMType			string		`json:"vmtype"`
	KeyPair			string		`json:"keypair_name"` // this refers to the secret name for the keypair
}

type VirtualMachineStatus struct {
	Status				string		`json:"status"` // default is nothing, but could be one of the following: starting, running, stopped, terminated
	Allocated			bool		`json:"allocated"`
	ActiveScenarioID	string		`json:"active_scenario_id"` // should only be populated when `allocated:true`
	PublicIP			net.IPAddr	`json:"public_ip"`
	PrivateIP			net.IPAddr	`json:"private_ip"`
	Hostname			string		`json:"hostname"` // ideally <hostname>.<enviroment dnssuffix> should be the FQDN to this machine
}

type VirtualMachineTypeSpec struct {
	Name	string	`json:"name"` // 2x4, etc.
	Image 	string	`json:"image"` // ubuntu-18.04
	CPU		int		`json:"cpu"`
	Memory	int		`json:"memory"`
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

type ActiveScenarioSpec struct {
	Scenario	string	`json:"scenario"`
	User		string	`json:"user"`
}

type ActiveScenarioStatus struct {
	StartTime	time.Time	`json:"start_time"`
	ExpirationTime	time.Time	`json:"end_time"`
}

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

// +genclient
// +genclient:noStatus
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// VM type is a genercized collection of information about a VM. this includes things like
// cpu, ram, disk, etc.

type VirtualMachineType struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              VirtualMachineTypeSpec `json:"spec"`
}

type VirtualMachineTypeList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`
	Items           []VirtualMachineType `json:"items"`
}