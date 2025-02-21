package v4alpha1

import (
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

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

func (c ScenarioStep) NamespaceScoped() bool {
	return false
}
