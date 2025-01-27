package v4alpha1

import metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// Event is an object that holds information and object references for something that happened (an event).
// An Event is what is used to communicate information about the processing of objects by HobbyFarm
// and/or other controllers. For example, provisioning steps about Machine resources may be communicated
// via multiple Event objects.
//
// Event objects are only expected to exist in primary storage for the duration of the referenced object's
// lifecycle, plus a little extra. After that they may be garbage collected and moved to longer term storage
// if such storage is available. This is necessary because large deployments necessitate many events being
// created and bogging down primary storage is not good.
type Event struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// ObjectReference holds the reference to the object to which this event applies
	ObjectReference ObjectReference `json:"objectRef"`

	// EventType is the type of this event. Values are simple Info/Warning/Error
	// as dictated by the EventType type.
	EventType EventType `json:"eventType"`

	// EventTime is the time (microsecond) when the event occurred.
	EventTime metav1.MicroTime `json:"eventTime"`

	// ReportingController is the name of the controller that generated the event.
	// This is distinct from ReportingInstance as there may be multiple instances
	// of a controller running.
	ReportingController string `json:"reportingController"`

	// ReportingInstance is the ID of the controller instance that generated the event.
	ReportingInstance string `json:"reportingInstance"`

	// Reason is a short string reason for the event's occurrence.
	// Something like "LDAP bind failed" or "MachineSet failed scale-up"
	// Think of it like a "subject" of an Event's "email"
	Reason string `json:"reason"`

	// Note is where additional information can be supplied to help contextualize
	// or direct next actions for the administrator who reviews the Event.
	// Something like "MachineSet failed scale-up means [x], you can try [y]"
	// Or more details, such as "LDAP bind failed because [x]"
	Note string `json:"note"`
}

type ObjectReference struct {
	// APIGroup is the api group of the referenced object, e.g. hobbyfarm.io
	APIGroup string `json:"apiGroup"`

	// Kind is the kind of the referenced object, e.g. Machine or AccessCode
	Kind string `json:"kind"`

	// Name is the name of the referenced object, e.g. m-djfks or ac-j913s
	Name string `json:"name"`
}

// EventType is a simple typing declaration for events.
// Events can come in one of three flavors: Info, Warning, or Error
type EventType string

const (
	EventTypeNormal  = "Info"
	EventTypeWarning = "Warning"
	EventTypeError   = "Error"
)

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type EventList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`

	Items []Event `json:"items"`
}
