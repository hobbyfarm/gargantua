// +k8s:deepcopy-gen=package,register
// +k8s:openapi-gen=true
// +groupName=hobbyfarm.io

package genericcondition

import (
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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
