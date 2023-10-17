package settingservice

import (
	"net/http"

	"github.com/golang/glog"
	"github.com/gorilla/mux"
	"github.com/hobbyfarm/gargantua/v3/protos/authn"
	"github.com/hobbyfarm/gargantua/v3/protos/authr"
)

type SettingServer struct {
	authnClient           authn.AuthNClient
	authrClient           authr.AuthRClient
	internalSettingServer *GrpcSettingServer
}

func NewSettingServer(authnClient authn.AuthNClient, authrClient authr.AuthRClient, internalSettingServer *GrpcSettingServer) (SettingServer, error) {
	s := SettingServer{}
	s.authnClient = authnClient
	s.authrClient = authrClient
	s.internalSettingServer = internalSettingServer
	return s, nil
}

func (s SettingServer) SetupRoutes(r *mux.Router) {
	r.HandleFunc("/setting/list/{scope}", s.ListFunc).Methods(http.MethodGet)
	r.HandleFunc("/setting/update/{setting_id}", s.UpdateFunc).Methods(http.MethodPut)
	r.HandleFunc("/setting/updatecollection", s.UpdateCollection).Methods(http.MethodPut)
	r.HandleFunc("/scope/list", s.ListScopeFunc).Methods(http.MethodGet)
	glog.V(2).Infof("set up routes for Setting server")
}
