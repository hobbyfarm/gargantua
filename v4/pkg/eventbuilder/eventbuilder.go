package eventbuilder

import (
	"context"
	"github.com/hobbyfarm/gargantua/v4/pkg/apis/hobbyfarm.io/v4alpha1"
	"github.com/hobbyfarm/gargantua/v4/pkg/names"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"log/slog"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"time"
)

// EventBuilder is a helper to build v4alpha1.Event objects
type EventBuilder struct {
	e *v4alpha1.Event
}

func create(t v4alpha1.EventType) *EventBuilder {
	now := metav1.NewMicroTime(time.Now())
	return &EventBuilder{
		e: &v4alpha1.Event{
			EventType: t,
			EventTime: &now,
		},
	}
}

// Info creates a new instance of EventBuilder with the EventType
// of the underlying v4alpha1.Event set to v4alpha1.EventTypeInfo
func Info() *EventBuilder {
	return create(v4alpha1.EventTypeInfo)
}

// Warning creates a new instance of EventBuilder with the EventType
// of the underlying v4alpha1.Event set to v4alpha1.EventTypeWarning
func Warning() *EventBuilder {
	return create(v4alpha1.EventTypeWarning)
}

// Error creates a new instance of EventBuilder with the EventType
// of the underlying v4alpha1.Event set to v4alpha1.EventTypeError
func Error() *EventBuilder {
	return create(v4alpha1.EventTypeError)
}

// WriteOrLog attempts to write the event using the provided client.
// Unlike Write, this method does not return an error. Instead
// it uses log/slog to write the error if one should occur.
func (e *EventBuilder) WriteOrLog(kclient client.Client) {
	if err := e.Write(kclient); err != nil {
		slog.Error(err.Error())
	}
}

// Write attempts to write the event using the provided client.
func (e *EventBuilder) Write(kclient client.Client) error {
	ev := e.ToEvent()
	return kclient.Create(context.Background(), ev)
}

// Type sets the EventType of the Event.
func (e *EventBuilder) Type(eventType v4alpha1.EventType) *EventBuilder {
	e.e.EventType = eventType

	return e
}

// For sets the v4alpha1.ObjectReference of the Event
func (e *EventBuilder) For(obj runtime.Object) *EventBuilder {
	var name string
	if thing, ok := obj.(client.Object); !ok {
		name = "UNKNOWN"
	} else {
		name = thing.GetName()
	}

	e.e.ObjectReference = v4alpha1.ObjectReference{
		Name:     name,
		APIGroup: obj.GetObjectKind().GroupVersionKind().Group,
		Kind:     obj.GetObjectKind().GroupVersionKind().Kind,
	}

	return e
}

// At sets the time at which the event occurred.
// If this method is not called, the time the *EventBuilder was created
// is used instead.
func (e *EventBuilder) At(t time.Time) *EventBuilder {
	mt := metav1.NewMicroTime(t)
	e.e.EventTime = &mt

	return e
}

// By sets the reporting information for the Event. This is the details of the
// controller which generated this event.
// reportingName is the string name of the controller
// reportingInstance is the string ID of the instance of the controller (not name)
func (e *EventBuilder) By(reportingName string, reportingInstance string) *EventBuilder {
	e.e.ReportingController = reportingName
	e.e.ReportingInstance = reportingInstance

	return e
}

// Reason sets the Reason field on the event.
func (e *EventBuilder) Reason(reason string) *EventBuilder {
	e.e.Reason = reason

	return e
}

// Note sets the Note field on the event.
func (e *EventBuilder) Note(note string) *EventBuilder {
	e.e.Note = note

	return e
}

// ToEvent converts the EventBuilder to an Event and returns it
func (e *EventBuilder) ToEvent() *v4alpha1.Event {
	e.e.GenerateName = names.EventGenerateNamePrefix

	return e.e
}
