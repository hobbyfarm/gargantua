package v2

import metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

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
	Email       string            `json:"email"`
	Password    string            `json:"password"`
	AccessCodes []string          `json:"access_codes"`
	Settings    map[string]string `json:"settings"`
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
	Name              string              `json:"name"`
	Description       string              `json:"description"`
	Steps             []ScenarioStep      `json:"steps"`
	Categories        []string            `json:"categories"`
	Tags              []string            `json:"tags"`
	VirtualMachines   []map[string]string `json:"virtualmachines"`
	KeepAliveDuration string              `json:"keepalive_duration"`
	PauseDuration     string              `json:"pause_duration"`
	Pauseable         bool                `json:"pauseable"`
	Tasks			  []VirtualMachineTasks `json:"vm_tasks"`	

}

type VirtualMachineTasks struct{	
	VMName					string 	`json:"vm_name"`
	Tasks 					[]Task	`json:"task_command"`
}
type Task struct{
	Name 					string  `json:"name"`
	Description				string  `json:"description"`
    Command 				string 	`json:"command"`
    ExpectedOutputValue		string 	`json:"expected_output_value"`
	ExpectedReturnCode 		int 	`json:"expected_return_code"`	
}

type ScenarioStep struct {
	Title   string `json:"title"`
	Content string `json:"content"`
}