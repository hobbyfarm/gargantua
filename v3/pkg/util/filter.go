package util

import (
	scheduledeventpb "github.com/hobbyfarm/gargantua/v3/protos/scheduledevent"
	sessionpb "github.com/hobbyfarm/gargantua/v3/protos/session"
)

// ScenarioProvider is an interface for any object that can provide scenarios. E. g. ScheduledEvents or Courses.
type ScenarioProvider interface {
	GetScenarios() []string
}

// Implemented by FilterByScenario and FilterByCourse
type EventFilterfunc func(se *scheduledeventpb.ScheduledEvent, objId string) bool

// Implemented by IsSessionOfScenario and IsSessionOfCourse
type SessionFilterfunc func(se *sessionpb.Session, objId string) bool

// Filter a ScheduledEventList to find SEs that are a) active and b) using the course/scenario specified
func FilterScheduledEvents(
	objId string,
	seList *scheduledeventpb.ListScheduledEventsResponse,
	filterFunc EventFilterfunc,
) []*scheduledeventpb.ScheduledEvent {
	outList := make([]*scheduledeventpb.ScheduledEvent, 0)
	for _, se := range seList.GetScheduledevents() {
		if se.GetStatus().GetFinished() {
			continue
		} else if filterFunc(se, objId) {
			outList = append(outList, se)
		}
	}

	return outList
}

func FilterSessions(
	objId string,
	sessList *sessionpb.ListSessionsResponse,
	filterFunc SessionFilterfunc,
) []*sessionpb.Session {
	outList := make([]*sessionpb.Session, 0)
	for _, sess := range sessList.GetSessions() {
		if filterFunc(sess, objId) {
			outList = append(outList, sess)
		}
	}

	return outList
}

// Specific filter function for scenarios
func FilterByScenario[T ScenarioProvider](obj T, scenario string) bool {
	return containsObjId(obj.GetScenarios(), scenario)
}

// Specific filter function for courses
func FilterByCourse(se *scheduledeventpb.ScheduledEvent, course string) bool {
	return containsObjId(se.GetCourses(), course)
}

// Returns true if session is assigned to the provided course
func IsSessionOfCourse(sess *sessionpb.Session, course string) bool {
	return sess.GetCourse() == course
}

// Returns true if session is assigned to the provided scenario
func IsSessionOfScenario(sess *sessionpb.Session, scenario string) bool {
	return sess.GetScenario() == scenario
}

// Helper function which returns if a string slice contains a specific object id
func containsObjId(objIds []string, obj string) bool {
	for _, o := range objIds {
		if o == obj {
			return true
		}
	}
	return false
}
