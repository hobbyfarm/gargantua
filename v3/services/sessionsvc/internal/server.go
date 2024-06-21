package sessionservice

import (
	"github.com/golang/glog"
	"github.com/gorilla/mux"
	accesscodepb "github.com/hobbyfarm/gargantua/v3/protos/accesscode"
	authnpb "github.com/hobbyfarm/gargantua/v3/protos/authn"
	authrpb "github.com/hobbyfarm/gargantua/v3/protos/authr"
	coursepb "github.com/hobbyfarm/gargantua/v3/protos/course"
	progresspb "github.com/hobbyfarm/gargantua/v3/protos/progress"
	scenariopb "github.com/hobbyfarm/gargantua/v3/protos/scenario"
	scheduledeventpb "github.com/hobbyfarm/gargantua/v3/protos/scheduledevent"
	vmclaimpb "github.com/hobbyfarm/gargantua/v3/protos/vmclaim"
)

type SessionServer struct {
	authnClient           authnpb.AuthNClient
	authrClient           authrpb.AuthRClient
	acClient              accesscodepb.AccessCodeSvcClient
	courseClient          coursepb.CourseSvcClient
	progressClient        progresspb.ProgressSvcClient
	scenarioClient        scenariopb.ScenarioSvcClient
	scheduledEventClient  scheduledeventpb.ScheduledEventSvcClient
	vmclaimClient         vmclaimpb.VMClaimSvcClient
	internalSessionServer *GrpcSessionServer
}

func NewSessionServer(
	authnClient authnpb.AuthNClient,
	authrClient authrpb.AuthRClient,
	acClient accesscodepb.AccessCodeSvcClient,
	courseClient coursepb.CourseSvcClient,
	progressClient progresspb.ProgressSvcClient,
	scenarioClient scenariopb.ScenarioSvcClient,
	scheduledEventClient scheduledeventpb.ScheduledEventSvcClient,
	vmclaimClient vmclaimpb.VMClaimSvcClient,
	internalSessionServer *GrpcSessionServer,
) SessionServer {
	return SessionServer{
		authnClient:           authnClient,
		authrClient:           authrClient,
		acClient:              acClient,
		courseClient:          courseClient,
		progressClient:        progressClient,
		scenarioClient:        scenarioClient,
		scheduledEventClient:  scheduledEventClient,
		vmclaimClient:         vmclaimClient,
		internalSessionServer: internalSessionServer,
	}
}

func (sss SessionServer) SetupRoutes(r *mux.Router) {
	r.HandleFunc("/session/new", sss.NewSessionFunc).Methods("POST")
	r.HandleFunc("/session/{session_id}", sss.GetSessionFunc).Methods("GET")
	r.HandleFunc("/session/{session_id}/finished", sss.FinishedSessionFunc).Methods("PUT")
	r.HandleFunc("/session/{session_id}/keepalive", sss.KeepAliveSessionFunc).Methods("PUT")
	r.HandleFunc("/session/{session_id}/pause", sss.PauseSessionFunc).Methods("PUT")
	r.HandleFunc("/session/{session_id}/resume", sss.ResumeSessionFunc).Methods("PUT")
	glog.V(2).Infof("set up routes for session server")
}
