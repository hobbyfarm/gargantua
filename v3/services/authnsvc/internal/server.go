package authnservice

import (
	"github.com/golang/glog"
	"github.com/gorilla/mux"
	accesscodepb "github.com/hobbyfarm/gargantua/v3/protos/accesscode"
	rbacpb "github.com/hobbyfarm/gargantua/v3/protos/rbac"
	scheduledeventpb "github.com/hobbyfarm/gargantua/v3/protos/scheduledevent"
	settingpb "github.com/hobbyfarm/gargantua/v3/protos/setting"
	userpb "github.com/hobbyfarm/gargantua/v3/protos/user"
)

type AuthServer struct {
	acClient             accesscodepb.AccessCodeSvcClient
	rbacClient           rbacpb.RbacSvcClient
	scheduledEventClient scheduledeventpb.ScheduledEventSvcClient
	settingClient        settingpb.SettingSvcClient
	userClient           userpb.UserSvcClient
	internalAuthnServer  *GrpcAuthnServer
}

func NewAuthServer(
	accesscodeClient accesscodepb.AccessCodeSvcClient,
	rbacClient rbacpb.RbacSvcClient,
	scheduledEventClient scheduledeventpb.ScheduledEventSvcClient,
	settingClient settingpb.SettingSvcClient,
	userClient userpb.UserSvcClient,
	internalAuthnServer *GrpcAuthnServer,
) (AuthServer, error) {
	a := AuthServer{}
	a.acClient = accesscodeClient
	a.rbacClient = rbacClient
	a.scheduledEventClient = scheduledEventClient
	a.settingClient = settingClient
	a.userClient = userClient
	a.internalAuthnServer = internalAuthnServer
	return a, nil
}

func (a AuthServer) SetupRoutes(r *mux.Router) {
	r.HandleFunc("/auth/registerwithaccesscode", a.RegisterWithAccessCodeFunc).Methods("POST")
	r.HandleFunc("/auth/accesscode", a.ListAccessCodeFunc).Methods("GET")
	r.HandleFunc("/auth/accesscode", a.AddAccessCodeFunc).Methods("POST")
	r.HandleFunc("/auth/accesscode/{access_code}", a.RemoveAccessCodeFunc).Methods("DELETE")
	r.HandleFunc("/auth/changepassword", a.ChangePasswordFunc).Methods("POST")
	r.HandleFunc("/auth/settings", a.RetreiveSettingsFunc).Methods("GET")
	r.HandleFunc("/auth/settings", a.UpdateSettingsFunc).Methods("POST")
	r.HandleFunc("/auth/authenticate", a.LoginFunc).Methods("POST")
	r.HandleFunc("/auth/access", a.GetAccessSet).Methods("GET")
	r.HandleFunc("/auth/scheduledevents", a.ListScheduledEventsFunc).Methods("GET")
	glog.V(2).Infof("set up route")
}
