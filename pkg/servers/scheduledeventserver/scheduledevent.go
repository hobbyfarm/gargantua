package scheduledeventserver

import (
	"encoding/json"
	"fmt"
	"github.com/golang/glog"
	"github.com/gorilla/mux"
	hfv1 "github.com/hobbyfarm/gargantua/pkg/apis/hobbyfarm.io/v1"
	"github.com/hobbyfarm/gargantua/pkg/authclient"
	hfClientset "github.com/hobbyfarm/gargantua/pkg/client/clientset/versioned"
	"github.com/hobbyfarm/gargantua/pkg/util"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"net/http"
)

type ScheduledEventServer struct {
	auth        *authclient.AuthClient
	hfClientSet *hfClientset.Clientset
}

func NewScheduledEventServer(authClient *authclient.AuthClient, hfClientset *hfClientset.Clientset) (*ScheduledEventServer, error) {
	es := ScheduledEventServer{}

	es.hfClientSet = hfClientset
	es.auth = authClient

	return &es, nil
}

func (se ScheduledEventServer) getScheduledEvent(id string) (hfv1.ScheduledEvent, error) {

	empty := hfv1.ScheduledEvent{}

	if len(id) == 0 {
		return empty, fmt.Errorf("scheduledevent passed in was empty")
	}

	obj, err := se.hfClientSet.HobbyfarmV1().ScheduledEvents().Get(id, metav1.GetOptions{})
	if err != nil {
		return empty, fmt.Errorf("error while retrieving ScheduledEvent by id: %s with error: %v", id, err)
	}

	return *obj, nil

}

func (se ScheduledEventServer) SetupRoutes(r *mux.Router) {
	r.HandleFunc("/scheduledevent/{scheduledevent_id}", se.GetScheduledEventFunc).Methods("GET")
	glog.V(2).Infof("set up routes for scheduledevent server")
}

type PreparedScheduledEvent struct {
	hfv1.ScheduledEventSpec
	hfv1.ScheduledEventStatus
}

func (se ScheduledEventServer) GetScheduledEventFunc(w http.ResponseWriter, r *http.Request) {
	_, err := se.auth.AuthNAdmin(w, r)
	if err != nil {
		util.ReturnHTTPMessage(w, r, 403, "forbidden", "no access to get scheduledEvent")
		return
	}

	vars := mux.Vars(r)

	scheduledEventId := vars["scheduledevent_id"]

	if len(scheduledEventId) == 0 {
		util.ReturnHTTPMessage(w, r, 500, "error", "no scheduledEvent id passed in")
		return
	}

	scheduledEvent, err := se.getScheduledEvent(scheduledEventId)

	if err != nil {
		glog.Errorf("error while retrieving scheduledEvent %v", err)
		util.ReturnHTTPMessage(w, r, 500, "error", "no scheduledEvent found")
		return
	}

	preparedScheduledEvent := PreparedScheduledEvent{scheduledEvent.Spec, scheduledEvent.Status}

	encodedScheduledEvent, err := json.Marshal(preparedScheduledEvent)
	if err != nil {
		glog.Error(err)
	}
	util.ReturnHTTPContent(w, r, 200, "success", encodedScheduledEvent)

	glog.V(2).Infof("retrieved scheduledEvent %s", scheduledEvent.Name)
}
