package courseservice

import (
	"github.com/gorilla/mux"
	accesscodepb "github.com/hobbyfarm/gargantua/v3/protos/accesscode"
	authnpb "github.com/hobbyfarm/gargantua/v3/protos/authn"
	authrpb "github.com/hobbyfarm/gargantua/v3/protos/authr"
	scenariopb "github.com/hobbyfarm/gargantua/v3/protos/scenario"
	scheduledeventpb "github.com/hobbyfarm/gargantua/v3/protos/scheduledevent"
	sessionpb "github.com/hobbyfarm/gargantua/v3/protos/session"
)

type CourseServer struct {
	authnClient          authnpb.AuthNClient
	authrClient          authrpb.AuthRClient
	acClient             accesscodepb.AccessCodeSvcClient
	scenarioClient       scenariopb.ScenarioSvcClient
	scheduledEventClient scheduledeventpb.ScheduledEventSvcClient
	sessionClient        sessionpb.SessionSvcClient
	internalCourseServer *GrpcCourseServer
}

func NewCourseServer(
	authnClient authnpb.AuthNClient,
	authrClient authrpb.AuthRClient,
	acClient accesscodepb.AccessCodeSvcClient,
	scenarioClient scenariopb.ScenarioSvcClient,
	scheduledEventClient scheduledeventpb.ScheduledEventSvcClient,
	sessionClient sessionpb.SessionSvcClient,
	internalCourseServer *GrpcCourseServer,
) CourseServer {
	return CourseServer{
		authnClient:          authnClient,
		authrClient:          authrClient,
		acClient:             acClient,
		scenarioClient:       scenarioClient,
		scheduledEventClient: scheduledEventClient,
		sessionClient:        sessionClient,
		internalCourseServer: internalCourseServer,
	}
}

func (c CourseServer) SetupRoutes(r *mux.Router) {
	r.HandleFunc("/course/list/catalog", c.ListCourseCatalog).Methods("GET")
	r.HandleFunc("/course/list/{access_code}", c.ListCoursesForAccesscode).Methods("GET")
	r.HandleFunc("/course/{course_id}", c.GetCourse).Methods("GET")
	r.HandleFunc("/a/course/list", c.ListFunc).Methods("GET")
	r.HandleFunc("/a/course/new", c.CreateFunc).Methods("POST")
	r.HandleFunc("/a/course/{course_id}", c.GetCourse).Methods("GET")
	r.HandleFunc("/a/course/{id}", c.UpdateFunc).Methods("PUT")
	r.HandleFunc("/a/course/{id}", c.DeleteFunc).Methods("DELETE")
	r.HandleFunc("/a/course/previewDynamicScenarios", c.previewDynamicScenarios).Methods("POST")
}
