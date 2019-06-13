package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +genclient
// +genclient:nonNamespaced
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
	Id						string		`json:"id"`
	VirtualMachineTemplateId string		`json:"vm_template_id"`
	KeyPair			       string 		`json:"keypair_name"` // this refers to the secret name for the keypair
	VirtualMachineClaimId	string		`json:"vm_claim_id"`
	UserId					string		`json:"user"`
	Provision				bool		`json:"provision"`
	VirtualMachineSetId		string		`json:"vm_set_id"`
}

type VirtualMachineStatus struct {
	Status				string		`json:"status"` // default is nothing, but could be one of the following: readyforprovision, provisioning, starting, running, stopped, terminating
	Allocated			bool		`json:"allocated"`
	Tainted				bool		`json:"tainted"`
	PublicIP			string		`json:"public_ip"`
	PrivateIP			string		`json:"private_ip"`
	EnvironmentId		string		`json:"environment_id"`
	Hostname			string		`json:"hostname"` // ideally <hostname>.<enviroment dnssuffix> should be the FQDN to this machine
	TFState				string		`json:"tfstate,omitempty"` // Terraform state name
}

// +genclient
// +genclient:nonNamespaced
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type VirtualMachineClaim struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              VirtualMachineClaimSpec `json:"spec"`
	Status			VirtualMachineClaimStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type VirtualMachineClaimList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`
	Items           []VirtualMachineClaim `json:"items"`
}

type VirtualMachineClaimSpec struct {
	Id		string	`json:"id"`
	UserId  string  `json:"user"`
	VirtualMachines map[string]VirtualMachineClaimVM `json:"vm"`
	VirtualMachineClassId string `json:"vm_class_id"`
}

type VirtualMachineClaimStatus struct {
	Bound	bool	`json:"bound"`
	Ready   bool 	`json:"ready"`
}

type VirtualMachineClaimVM struct {
	Template    string  `json:"template"`
	VirtualMachineId	string	`json:"vm_id"`
}

// +genclient
// +genclient:noStatus
// +genclient:nonNamespaced
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
	Id		string 	`json:"id"`
	Name	string	`json:"name"` // 2x4, etc.
	Image 	string	`json:"image"` // ubuntu-18.04
	Resources CMSStruct `json:"resources"`
}

// +genclient
// +genclient:nonNamespaced
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type Environment struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              EnvironmentSpec `json:"spec"`
	Status 			  EnvironmentStatus `json:"status"`
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
	TemplateMapping			map[string]map[string]string `json:"template_mapping"` //  lol
	EnvironmentSpecifics		map[string]string `json:"environment_specifics"`
	IPTranslationMap		map[string]string `json:"ip_translation_map"`
}

type EnvironmentStatus struct {
	Capacity					CMSStruct `json:"capacity"`
	Used						CMSStruct `json:"used"`
	AvailableCount				map[string]int `json:"available_count"`
}

type CMSStruct struct {
	CPU		int `json:"cpu"` // cores
	Memory  int `json:"memory"` // in MB
	Storage int `json:"storage"` // in GB
}

// +genclient
// +genclient:nonNamespaced
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type VirtualMachineSet struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec	VirtualMachineSetSpec `json:"spec"`
	Status	VirtualMachineSetStatus	`json:"status"`
}

type VirtualMachineSetSpec struct {
	Count int `json:"count"`
	Environment string `json:"environment"`
	VMTemplate string `json:"vm_template"`
	BaseName	string	`json:"base_name"`
}

type VirtualMachineSetStatus struct {
	Machines []VirtualMachineProvision	`json:"machines"`
	AvailableCount int	`json:"available"`
	ProvisionedCount	int	`json:"provisioned"`
}

type VirtualMachineProvision struct {
	VirtualMachineName string `json:"vm_name"`
	TFControllerState string `json:"tfc_state"`
	TFControllerCM	string	`json:"tfc_cm"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type VirtualMachineSetList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items	[]VirtualMachineSet	`json:"items"`
}

// +genclient
// +genclient:noStatus
// +genclient:nonNamespaced
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
	Id string	`json:"id"`
	Name string `json:"name"`
	Description string `json:"description"`
	Steps []ScenarioStep `json:"steps"`
	VirtualMachines []map[string]string `json:"virtualmachines"`
}

type ScenarioStep struct {
	Title string `json:"title"`
	Content string `json:"content"`
}

// +genclient
// +genclient:nonNamespaced
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type ScenarioSession struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              ScenarioSessionSpec `json:"spec"`
	Status			  ScenarioSessionStatus `json:"status"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type ScenarioSessionList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`
	Items           []ScenarioSession `json:"items"`
}

type ScenarioSessionSpec struct {
	Id 			string			`json:"id"`
	ScenarioId	string			`json:"scenario"`
	UserId		string			`json:"user"`
	VmClaimSet		[]string	`json:"vm_claim"`
}

type ScenarioSessionStatus struct {
	Active 		bool		`json:"active"`
	StartTime	string		`json:"start_time"`
	ExpirationTime	string	`json:"end_time"`
}

// +genclient
// +genclient:noStatus
// +genclient:nonNamespaced
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
// +genclient:nonNamespaced
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
	Id string `json:"id"`
	Email string `json:"email"`
	Password string `json:"password"`
	AccessCodes []string `json:"access_codes"`
}