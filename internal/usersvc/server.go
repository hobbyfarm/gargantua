package userservice

import (
	"github.com/golang/glog"
	"github.com/gorilla/mux"
)

type UserServer struct {
	tlsCaPath          string
	internalUserServer *GrpcUserServer
}

func NewUserServer(tlsCaPath string, internalUserServer *GrpcUserServer) (UserServer, error) {
	u := UserServer{}
	u.tlsCaPath = tlsCaPath
	u.internalUserServer = internalUserServer
	return u, nil
}

func (u UserServer) SetupRoutes(r *mux.Router) {
	r.HandleFunc("/a/user/list", u.ListFunc).Methods("GET")
	r.HandleFunc("/a/user/{id}", u.GetFunc).Methods("GET")
	r.HandleFunc("/a/user", u.UpdateFunc).Methods("PUT")
	r.HandleFunc("/a/user/{id}", u.DeleteFunc).Methods("DELETE")
	glog.V(2).Infof("set up routes for User server")
}
