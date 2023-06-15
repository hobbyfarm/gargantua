package authserver

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/hobbyfarm/gargantua/pkg/accesscode"
	"github.com/hobbyfarm/gargantua/pkg/rbac"

	"github.com/golang/glog"
	"github.com/gorilla/mux"
	hfClientset "github.com/hobbyfarm/gargantua/pkg/client/clientset/versioned"
	"github.com/hobbyfarm/gargantua/pkg/util"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type AuthServer struct {
	tlsCA            string
	hfClientSet      hfClientset.Interface
	accessCodeClient *accesscode.AccessCodeClient
	ctx              context.Context
}

func NewAuthServer(tlsCA string, hfClientSet hfClientset.Interface, ctx context.Context, acClient *accesscode.AccessCodeClient) (AuthServer, error) {
	a := AuthServer{}
	a.tlsCA = tlsCA
	a.hfClientSet = hfClientSet
	a.ctx = ctx
	a.accessCodeClient = acClient
	return a, nil
}

func (a AuthServer) SetupRoutes(r *mux.Router) {
	r.HandleFunc("/auth/scheduledevents", a.ListScheduledEventsFunc).Methods("GET")
	glog.V(2).Infof("set up route")
}

func (a AuthServer) ListScheduledEventsFunc(w http.ResponseWriter, r *http.Request) {
	user, err := rbac.AuthenticateRequest(r, a.tlsCA)
	if err != nil {
		util.ReturnHTTPMessage(w, r, 403, "forbidden", "no access to list suitable scheduledevents")
		return
	}

	accessCodes, err := a.accessCodeClient.GetAccessCodes(user.GetAccessCodes())
	if err != nil {
		util.ReturnHTTPMessage(w, r, 500, "error", "error while retrieving access codes")
		return
	}

	accessCodeScheduledEvent := make(map[string]string)

	//Getting single SEs should be faster than listing all of them and iterating them in O(n^2), in most cases users only have a hand full of accessCodes.
	for _, ac := range accessCodes {
		se, err := a.hfClientSet.HobbyfarmV1().ScheduledEvents(util.GetReleaseNamespace()).Get(a.ctx, ac.Labels[util.ScheduledEventLabel], metav1.GetOptions{})
		if err != nil {
			glog.Error(err)
			continue
		}
		accessCodeScheduledEvent[ac.Spec.Code] = se.Spec.Name
	}

	encodedMap, err := json.Marshal(accessCodeScheduledEvent)
	if err != nil {
		glog.Error(err)
	}
	util.ReturnHTTPContent(w, r, 200, "success", encodedMap)
}
