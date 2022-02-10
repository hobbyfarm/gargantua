package rbac

import (
	"encoding/json"
	"github.com/golang/glog"
	"github.com/gorilla/mux"
	"github.com/hobbyfarm/gargantua/pkg/authclient"
	"github.com/hobbyfarm/gargantua/pkg/util"
	"k8s.io/client-go/informers"
	"net/http"
)

const (
	rbacapigroup = "rbac.authorization.k8s.io"
	all = "*"
)

type RbacServer struct {
	auth *authclient.AuthClient

	userIndex *Index
	groupIndex *Index
}

func NewRbacServer(auth *authclient.AuthClient, kubeInformerFactory informers.SharedInformerFactory) (*RbacServer, error) {
	userIndex, err := NewIndex("User", kubeInformerFactory)
	if err != nil {
		return nil, err
	}

	groupIndex, err := NewIndex("Group", kubeInformerFactory)
	if err != nil {
		return nil, err
	}

	return &RbacServer{
		auth: auth,
		userIndex: userIndex,
		groupIndex: groupIndex,
	}, nil
}

func (rs RbacServer) SetupRoutes(r *mux.Router) {
	r.HandleFunc("/rbac/access", rs.GetAccessSet).Methods(http.MethodGet)
}

func (rs *RbacServer) GetAccessSet(w http.ResponseWriter, r *http.Request) {
	user, err := rs.auth.AuthN(w, r)
	if err != nil {
		util.ReturnHTTPMessage(w, r, http.StatusUnauthorized, "unauthorized", "unauthorized")
		return
	}

	// need to get the user's access set and publish to front end
	as, err := rs.userIndex.GetAccessSet(user.Spec.Email)
	if err != nil {
		util.ReturnHTTPMessage(w, r, http.StatusInternalServerError, "internalerror", "internal error fetching access set")
		glog.Error(err)
		return
	}

	encodedAS, err := json.Marshal(as)
	if err != nil {
		util.ReturnHTTPMessage(w, r, http.StatusInternalServerError, "internalerror", "internal error encoding access set")
		glog.Error(err)
		return
	}

	util.ReturnHTTPContent(w, r, http.StatusOK, "access_set", encodedAS)
}

