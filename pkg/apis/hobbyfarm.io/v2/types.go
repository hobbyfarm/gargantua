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

type ScheduledEventStatus struct {
	VirtualMachineSets []string `json:"vmsets"`
	Active             bool     `json:"active"`
	Provisioned        bool     `json:"provisioned"`
	Ready              bool     `json:"ready"`
	Finished           bool     `json:"finished"`
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
	SharedVirtualMachines   []SharedVirtualMachine    `json:"shared_vms"`
}

type SharedVirtualMachine struct {
	VMId        string `json:"vmId"`
	Name        string `json:"name"`	
	Environment string `json:"environment"`
	VMTemplate  string `json:"vmTemplate"`
}
