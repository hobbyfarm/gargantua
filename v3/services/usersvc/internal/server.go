package userservice

import (
	"github.com/golang/glog"
	"github.com/gorilla/mux"
	"github.com/hobbyfarm/gargantua/v3/protos/authn"
	"github.com/hobbyfarm/gargantua/v3/protos/authr"
	"github.com/hobbyfarm/gargantua/v3/protos/rbac"
)

type UserServer struct {
	authnClient        authn.AuthNClient
	authrClient        authr.AuthRClient
	rbacClient         rbac.RbacSvcClient
	internalUserServer *GrpcUserServer
}

func NewUserServer(authnClient authn.AuthNClient, authrClient authr.AuthRClient, rbacClient rbac.RbacSvcClient, internalUserServer *GrpcUserServer) (UserServer, error) {
	u := UserServer{}
	u.authnClient = authnClient
	u.authrClient = authrClient
	u.rbacClient = rbacClient
	u.internalUserServer = internalUserServer
	return u, nil
}

func (u UserServer) SetupRoutes(r *mux.Router) {
	r.HandleFunc("/a/user/list", u.ListFunc).Methods("GET")
	r.HandleFunc("/a/user/{id}", u.GetFunc).Methods("GET")
	r.HandleFunc("/a/user", u.UpdateFunc).Methods("PUT")
	r.HandleFunc("/a/user/{id}", u.DeleteFunc).Methods("DELETE")
	r.HandleFunc("/a/user/{user}/rolebindings", u.ListRoleBindingsForUser).Methods("GET")
	glog.V(2).Infof("set up routes for User server")
}
