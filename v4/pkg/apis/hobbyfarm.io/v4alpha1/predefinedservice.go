package v4alpha1

import metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// PredefinedService represents a service (as in application, or web service) that is
// hosted on a Machine. Predefined
type PredefinedService struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec PredefinedServiceSpec `json:"spec"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type PredefinedServiceList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	Items []PredefinedService `json:"items"`
}

type DisplayOption string

const (
	// HasTab determines if a service gets its own tab in the UI
	HasTab DisplayOption = "HasTab"

	// HasWebInterface determines if a service does or does not have a web interface
	HasWebInterface DisplayOption = "HasWebInterface"
)

type HttpOption string

const (
	// NoRewriteRootPath disables path rewriting from /p/[vmid]/80/path to /path
	NoRewriteRootPath HttpOption = "NoRewriteRootPath"

	// RewriteHostHeader rewrites the host header to the proxy server host
	RewriteHostHeader HttpOption = "RewriteHostHeader"

	// RewriteOriginHeader rewrites the origin to localhost instead of the proxy host
	RewriteOriginHeader HttpOption = "RewriteOriginHeader"

	// DisallowIframe forces opening the service content in a new browser tab instead of iframe
	DisallowIframe HttpOption = "DisallowIframe"
)

type PredefinedServiceSpec struct {
	// DisplayName is the display (pretty) name of the PredefinedService
	DisplayName string `json:"displayName"`

	// Port is the network port of the service
	Port int `json:"port"`

	// DisplayOptions is a list of display (ui) options that this service requires.
	DisplayOptions []DisplayOption `json:"displayOptions"`

	// HttpOptions is a list of http options that this service requires.
	HttpOptions []HttpOption `json:"httpOptions"`

	// Path is the path on the VM that the service is accessible upon
	Path string `json:"path"`

	// CloudConfig contains the cloud-config data used to setup this service on the machine
	CloudConfig string `json:"cloudConfig"`
}

func (c PredefinedService) NamespaceScoped() bool {
	return false
}
