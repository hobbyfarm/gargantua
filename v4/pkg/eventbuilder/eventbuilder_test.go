package eventbuilder

import (
	"github.com/hobbyfarm/gargantua/v4/pkg/apis/hobbyfarm.io/v4alpha1"
	"testing"
	"time"
)

func Test_Types(t *testing.T) {
	info := Info()

	if info.ToEvent().EventType != v4alpha1.EventTypeInfo {
		t.Errorf("wrong event type, expected info got %s", info.ToEvent().EventType)
	}

	warning := info.Type(v4alpha1.EventTypeWarning)

	if warning.ToEvent().EventType != v4alpha1.EventTypeWarning {
		t.Errorf("wrong event type, expected warning got %s", warning.ToEvent().EventType)
	}
}

func Test_Time(t *testing.T) {
	t1 := Info()

	oldtime := t1.ToEvent().EventTime
	newtime := t1.At(time.Now()).ToEvent().EventTime

	if oldtime == newtime {
		t.Errorf("wrong event time, expected new time got %s", newtime)
	}

	if oldtime.After(newtime.Time) {
		t.Errorf("wrong event time, expected new time got %s", newtime)
	}
}

func Test_Fields(t *testing.T) {
	e := Info()

	const note = "this is a note"
	const reason = "this is a reason"

	const byName = "fake-controller"
	const byInstance = "fake-instance"

	e.Note(note).Reason(reason).By(byName, byInstance)

	eout := e.ToEvent()

	if eout.Note != note {
		t.Errorf("wrong note, expected %s got %s", note, eout.Note)
	}

	if eout.Reason != reason {
		t.Errorf("wrong reason, expected %s got %s", reason, eout.Reason)
	}

	if eout.ReportingController != byName {
		t.Errorf("wrong reporting controller, expected %s got %s", byName, eout.ReportingController)
	}

	if eout.ReportingInstance != byInstance {
		t.Errorf("wrong reporting instance, expected %s got %s", byInstance, eout.ReportingInstance)
	}
}
