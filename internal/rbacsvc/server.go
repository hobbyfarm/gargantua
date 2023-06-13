package rbac

import (
	"github.com/gorilla/mux"
	"k8s.io/client-go/kubernetes"
)

const (
	k8sRbacGroup              = "rbac.authorization.k8s.io"
	roleResourcePlural        = "roles"
	roleBindingResourcePlural = "rolebindings"
)

type Server struct {
	kubeClientSet      *kubernetes.Clientset
	internalRbacServer *GrpcRbacServer
	tlsCA              string
}

func NewRbacServer(kubeClientSet *kubernetes.Clientset, internalRbacServer *GrpcRbacServer, tlsCA string) *Server {
	return &Server{
		kubeClientSet:      kubeClientSet,
		internalRbacServer: internalRbacServer,
		tlsCA:              tlsCA,
	}
}

func (s Server) SetupRoutes(r *mux.Router) {
	r.HandleFunc("/a/roles/list", s.ListRoles).Methods("GET")
	r.HandleFunc("/a/roles/{id}", s.GetRole).Methods("GET")
	r.HandleFunc("/a/roles/create", s.CreateRole).Methods("POST")
	r.HandleFunc("/a/roles/{id}", s.UpdateRole).Methods("PUT")
	r.HandleFunc("/a/roles/{id}", s.DeleteRole).Methods("DELETE")
	r.HandleFunc("/a/user/{user}/rolebindings", s.ListRoleBindingsForUser).Methods("GET")
	r.HandleFunc("/a/rolebindings/list", s.ListRoleBindings).Methods("GET")
	r.HandleFunc("/a/rolebindings/{id}", s.GetRoleBinding).Methods("GET")
	r.HandleFunc("/a/rolebindings/create", s.CreateRoleBinding).Methods("POST")
	r.HandleFunc("/a/rolebindings/{id}", s.UpdateRoleBinding).Methods("PUT")
	r.HandleFunc("/a/rolebindings/{id}", s.DeleteRoleBinding).Methods("DELETE")
}
