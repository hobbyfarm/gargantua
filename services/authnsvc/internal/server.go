package authnservice

import (
	"github.com/golang/glog"
	"github.com/gorilla/mux"
	"github.com/hobbyfarm/gargantua/v3/protos/accesscode"
	"github.com/hobbyfarm/gargantua/v3/protos/rbac"
	"github.com/hobbyfarm/gargantua/v3/protos/setting"
	"github.com/hobbyfarm/gargantua/v3/protos/user"
)

type AuthServer struct {
	acClient            accesscode.AccessCodeSvcClient
	userClient          user.UserSvcClient
	settingClient       setting.SettingSvcClient
	rbacClient          rbac.RbacSvcClient
	internalAuthnServer *GrpcAuthnServer
}

func NewAuthServer(accesscodeClient accesscode.AccessCodeSvcClient, userClient user.UserSvcClient, settingCLient setting.SettingSvcClient, rbacClient rbac.RbacSvcClient, internalAuthnServer *GrpcAuthnServer) (AuthServer, error) {
	a := AuthServer{}
	a.acClient = accesscodeClient
	a.userClient = userClient
	a.settingClient = settingCLient
	a.rbacClient = rbacClient
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
	glog.V(2).Infof("set up route")
}
