package rbac

import (
	"github.com/gorilla/mux"
	"github.com/hobbyfarm/gargantua/v3/protos/authn"
	"github.com/hobbyfarm/gargantua/v3/protos/authr"
)

type Server struct {
	internalRbacServer *GrpcRbacServer
	authnClient        authn.AuthNClient
	authrClient        authr.AuthRClient
}

func NewRbacServer(internalRbacServer *GrpcRbacServer, authnClient authn.AuthNClient, authrClient authr.AuthRClient) *Server {
	return &Server{
		internalRbacServer: internalRbacServer,
		authnClient:        authnClient,
		authrClient:        authrClient,
	}
}

func (s Server) SetupRoutes(r *mux.Router) {
	r.HandleFunc("/a/roles/list", s.ListRoles).Methods("GET")
	r.HandleFunc("/a/roles/{id}", s.GetRole).Methods("GET")
	r.HandleFunc("/a/roles/create", s.CreateRole).Methods("POST")
	r.HandleFunc("/a/roles/{id}", s.UpdateRole).Methods("PUT")
	r.HandleFunc("/a/roles/{id}", s.DeleteRole).Methods("DELETE")
	r.HandleFunc("/a/rolebindings/list", s.ListRoleBindings).Methods("GET")
	r.HandleFunc("/a/rolebindings/{id}", s.GetRoleBinding).Methods("GET")
	r.HandleFunc("/a/rolebindings/create", s.CreateRoleBinding).Methods("POST")
	r.HandleFunc("/a/rolebindings/{id}", s.UpdateRoleBinding).Methods("PUT")
	r.HandleFunc("/a/rolebindings/{id}", s.DeleteRoleBinding).Methods("DELETE")
}
