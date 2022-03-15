package rbacserver

import (
	"github.com/gorilla/mux"
	"github.com/hobbyfarm/gargantua/pkg/authclient"
	"k8s.io/client-go/kubernetes"
)

const (
	k8sRbacGroup = "rbac.authorization.k8s.io"
	rbacManagedLabel   = "rbac.hobbyfarm.io/managed"
	roleResourcePlural = "roles"
	roleBindingResourcePlural = "rolebindings"
)

type Server struct {
	kubeClientSet *kubernetes.Clientset
	auth *authclient.AuthClient
}

func NewRbacServer(kubeClientSet *kubernetes.Clientset, authClient *authclient.AuthClient) *Server {
	return &Server{
		kubeClientSet: kubeClientSet,
		auth: authClient,
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
	r.HandleFunc("/s/rolebindings/{id}", s.DeleteRoleBinding).Methods("DELETE")
}

