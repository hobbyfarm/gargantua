package rbacserver

import (
	"github.com/gorilla/mux"
	"github.com/hobbyfarm/gargantua/pkg/authclient"
	"github.com/hobbyfarm/gargantua/pkg/rbacclient"
	"k8s.io/client-go/kubernetes"
)

const (
	k8sRbacGroup = "rbac.authorization.k8s.io"
	rbacManagedLabel   = "rbac.hobbyfarm.io/managed"
	userLabel = "hobbyfarm.io/user"
	roleResourcePlural = "roles"
	roleBindingResourcePlural = "rolebindings"
)

type Server struct {
	kubeClientSet *kubernetes.Clientset
	auth *authclient.AuthClient
	rbac *rbacclient.Client
}

func NewRbacServer(kubeClientSet *kubernetes.Clientset, authClient *authclient.AuthClient, rbacClient *rbacclient.Client) *Server {
	return &Server{
		kubeClientSet: kubeClientSet,
		auth: authClient,
		rbac: rbacClient,
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

