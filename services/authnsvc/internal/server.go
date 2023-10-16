package authnservice

// TODO: Remove Client Set Import
import (
	"github.com/golang/glog"
	"github.com/gorilla/mux"
)

type AuthServer struct {
	tlsCaPath           string
	internalAuthnServer *GrpcAuthnServer
}

func NewAuthServer(tlsCaPath string, internalAuthnServer *GrpcAuthnServer) (AuthServer, error) {
	a := AuthServer{}
	a.tlsCaPath = tlsCaPath
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
