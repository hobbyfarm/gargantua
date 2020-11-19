package courseserver

import (
	"github.com/gorilla/mux"
	hfv1 "github.com/hobbyfarm/gargantua/pkg/apis/hobbyfarm.io/v1"
	"github.com/hobbyfarm/gargantua/pkg/authclient"
	hfClientset "github.com/hobbyfarm/gargantua/pkg/client/clientset/versioned"
)

type AdminCourseServer struct {
	auth *authclient.AuthClient
	hfClientSet *hfClientset.Clientset
}

type PreparedCourse struct {
	Id string `json:"id"`
	hfv1.CourseSpec
}

func NewAdminCourseServer(authClient *authclient.AuthClient, hfClientset *hfClientset.Clientset) (*AdminCourseServer, error) {
	s := AdminCourseServer{}

	s.hfClientSet = hfClientset
	s.auth = authClient

	return &s, nil
}

func (a AdminCourseServer) SetupRoutes(r *mux.Router) {
}








