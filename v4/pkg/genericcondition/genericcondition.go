// +k8s:deepcopy-gen=package,register
// +k8s:openapi-gen=true
// +groupName=hobbyfarm.io

package genericcondition

import (
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"reflect"
	"time"
)

// +k8s:deepcopy-gen=true

type GenericCondition struct {
	// Type of cluster condition.
	Type string `json:"type"`
	// Status of the condition, one of True, False, Unknown.
	Status v1.ConditionStatus `json:"status"`
	// The last time this condition was updated.
	LastUpdateTime metav1.Time `json:"lastUpdateTime,omitempty"`
	// Last time the condition transitioned from one status to another.
	LastTransitionTime metav1.Time `json:"lastTransitionTime,omitempty"`
	// The reason for the condition's last transition.
	Reason string `json:"reason,omitempty"`
	// Human-readable message indicating details about last transition
	Message string `json:"message,omitempty"`
}

func (g *GenericCondition) ChangeCondition(status v1.ConditionStatus, reason string, message string) {
	if g.Status != status {
		// there was a change, note it
		g.LastTransitionTime = metav1.NewTime(time.Now())
		g.Status = status
	}

	g.LastUpdateTime = metav1.NewTime(time.Now())
	g.Message = message
	g.Reason = reason
}

func Update(obj runtime.Object, name string, status v1.ConditionStatus, reason string, message string) {
	if k := get(obj, name); k != nil {
		k.ChangeCondition(status, reason, message)
	}
}

func get(obj runtime.Object, name string) *GenericCondition {
	ptr := reflect.ValueOf(obj).FieldByName("Status").FieldByName("Conditions").Addr()
	v := ptr.Interface().([]GenericCondition)

	for _, vv := range v {
		if vv.Type == name {
			return &vv
		}
	}

	return nil
}

func create(obj runtime.Object, name string) {
	ptr := reflect.ValueOf(obj).FieldByName("Status").FieldByName("Conditions").Addr()
	v := ptr.Interface().([]GenericCondition)

	v = append(v, GenericCondition{
		Type:   name,
		Status: v1.ConditionUnknown,
	})
}

func CreateIfNot(obj runtime.Object, name string) {
	if k := get(obj, name); k == nil {
		create(obj, name)
	}
}
