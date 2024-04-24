package v4alpha1

import metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// ServiceAccount functions largely the same as a Kubernetes SA. It is an account that
// services can use to authenticate with HobbyFarm.
//
// "Why not use k8s corev1 ServiceAccount?"
// Because if k8s is used as a backing store for HobbyFarm, there is a collision with the
// HF api server instance of ServiceAccount and the backing store. The k8s cluster storing the data
// will start operating on that ServiceAccount by generating tokens (that wouldn't be valid for
// the HF apiserver, only for that k8s cluster), or giving it access to things in that k8s cluster.
type ServiceAccount struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// Secrets is a list of secret object names that contain the tokens used to authenticate
	// a ServiceAccount to the HobbyFarm apiserver.
	Secrets []string `json:"secrets"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type ServiceAccountList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	Items []ServiceAccount `json:"items"`
}

func (c ServiceAccount) NamespaceScoped() bool {
	return false
}
