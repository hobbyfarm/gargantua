package scheduledeventserver

import (
	"crypto/sha256"
	"encoding/base32"
	"encoding/json"
	"fmt"
	"github.com/golang/glog"
	"github.com/gorilla/mux"
	hfv1 "github.com/hobbyfarm/gargantua/pkg/apis/hobbyfarm.io/v1"
	"github.com/hobbyfarm/gargantua/pkg/authclient"
	hfClientset "github.com/hobbyfarm/gargantua/pkg/client/clientset/versioned"
	"github.com/hobbyfarm/gargantua/pkg/util"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/util/retry"
	"net/http"
	"strings"
	"time"
)

type AdminScheduledEventServer struct {
	auth        *authclient.AuthClient
	hfClientSet *hfClientset.Clientset
}

func NewAdminScheduledEventServer(authClient *authclient.AuthClient, hfClientset *hfClientset.Clientset) (*AdminScheduledEventServer, error) {
	es := AdminScheduledEventServer{}

	es.hfClientSet = hfClientset
	es.auth = authClient

	return &es, nil
}

func (a AdminScheduledEventServer) getScheduledEvent(id string) (hfv1.ScheduledEvent, error) {

	empty := hfv1.ScheduledEvent{}

	if len(id) == 0 {
		return empty, fmt.Errorf("scheduledevent passed in was empty")
	}

	obj, err := a.hfClientSet.HobbyfarmV1().ScheduledEvents().Get(id, metav1.GetOptions{})
	if err != nil {
		return empty, fmt.Errorf("error while retrieving ScheduledEvent by id: %s with error: %v", id, err)
	}

	return *obj, nil

}

func (a AdminScheduledEventServer) SetupRoutes(r *mux.Router) {
	r.HandleFunc("/a/scheduledevent/list", a.ListFunc).Methods("GET")
	r.HandleFunc("/a/scheduledevent/new", a.CreateFunc).Methods("POST")
	r.HandleFunc("/a/scheduledevent/{id}", a.GetFunc).Methods("GET")
	r.HandleFunc("/a/scheduledevent/{id}", a.UpdateFunc).Methods("PUT")
	glog.V(2).Infof("set up routes for admin scheduledevent server")
}

type PreparedScheduledEvent struct {
	ID string `json:"id"`
	hfv1.ScheduledEventSpec
	hfv1.ScheduledEventStatus
}

func (a AdminScheduledEventServer) GetFunc(w http.ResponseWriter, r *http.Request) {
	_, err := a.auth.AuthNAdmin(w, r)
	if err != nil {
		util.ReturnHTTPMessage(w, r, 403, "forbidden", "no access to get scheduledEvent")
		return
	}

	vars := mux.Vars(r)

	scheduledEventId := vars["id"]

	if len(scheduledEventId) == 0 {
		util.ReturnHTTPMessage(w, r, 500, "error", "no scheduledEvent id passed in")
		return
	}

	scheduledEvent, err := a.getScheduledEvent(scheduledEventId)

	if err != nil {
		glog.Errorf("error while retrieving scheduledEvent %v", err)
		util.ReturnHTTPMessage(w, r, 500, "error", "no scheduledEvent found")
		return
	}

	preparedScheduledEvent := PreparedScheduledEvent{scheduledEvent.Name, scheduledEvent.Spec, scheduledEvent.Status}

	encodedScheduledEvent, err := json.Marshal(preparedScheduledEvent)
	if err != nil {
		glog.Error(err)
	}
	util.ReturnHTTPContent(w, r, 200, "success", encodedScheduledEvent)

	glog.V(2).Infof("retrieved scheduledEvent %s", scheduledEvent.Name)
}

func (a AdminScheduledEventServer) ListFunc(w http.ResponseWriter, r *http.Request) {
	_, err := a.auth.AuthNAdmin(w, r)
	if err != nil {
		util.ReturnHTTPMessage(w, r, 403, "forbidden", "no access to get scheduledevents")
		return
	}

	scheduledEvents, err := a.hfClientSet.HobbyfarmV1().ScheduledEvents().List(metav1.ListOptions{})

	if err != nil {
		glog.Errorf("error while retrieving scheduledevents %v", err)
		util.ReturnHTTPMessage(w, r, 500, "error", "no scheduledevents found")
		return
	}

	preparedScheduledEvents := []PreparedScheduledEvent{}
	for _, s := range scheduledEvents.Items {
		preparedScheduledEvents = append(preparedScheduledEvents, PreparedScheduledEvent{s.Name, s.Spec, s.Status})
	}

	encodedScheduledEvents, err := json.Marshal(preparedScheduledEvents)
	if err != nil {
		glog.Error(err)
	}
	util.ReturnHTTPContent(w, r, 200, "success", encodedScheduledEvents)

	glog.V(2).Infof("listed scheduled events")
}

func (a AdminScheduledEventServer) CreateFunc(w http.ResponseWriter, r *http.Request) {
	user, err := a.auth.AuthNAdmin(w, r)
	if err != nil {
		util.ReturnHTTPMessage(w, r, 403, "forbidden", "no access to create scheduledevents")
		return
	}

	name := r.PostFormValue("name")
	if name == "" {
		util.ReturnHTTPMessage(w, r, 400, "badrequest", "no name passed in")
		return
	}
	description := r.PostFormValue("description")
	if description == "" {
		util.ReturnHTTPMessage(w, r, 400, "badrequest", "no description passed in")
		return
	}
	startTime := r.PostFormValue("start_time")
	if startTime == "" {
		util.ReturnHTTPMessage(w, r, 400, "badrequest", "no start time passed in")
		return
	}
	endTime := r.PostFormValue("end_time")
	if endTime == "" {
		util.ReturnHTTPMessage(w, r, 400, "badrequest", "no end time passed in")
		return
	}
	requiredVM := r.PostFormValue("required_vms")
	if requiredVM == "" {
		util.ReturnHTTPMessage(w, r, 400, "badrequest", "no required vm map passed in")
		return
	}
	accessCode := r.PostFormValue("access_code")
	if accessCode == "" {
		util.ReturnHTTPMessage(w, r, 400, "badrequest", "no access code passed in")
		return
	}
	scenariosRaw := r.PostFormValue("scenarios")
	coursesRaw := r.PostFormValue("courses")
	if scenariosRaw == "" && coursesRaw == "" {
		util.ReturnHTTPMessage(w, r, 400, "badrequest", "no scenarios or courses passed in")
		return
	}

	restrictionDisabled := false
	restrictionDisabledRaw := r.PostFormValue("disable_restriction")
	if restrictionDisabledRaw == "" {
		restrictionDisabled = false
	} else {
		if strings.ToLower(restrictionDisabledRaw) == "false" {
			restrictionDisabled = false
		} else {
			restrictionDisabled = true
		}
	}

	requiredVMUnmarshaled := map[string]map[string]int{}

	err = json.Unmarshal([]byte(requiredVM), &requiredVMUnmarshaled)
	if err != nil {
		glog.Errorf("error while unmarshalling required VM's %v", err)
		util.ReturnHTTPMessage(w, r, 500, "internalerror", "error parsing")
		return
	}

	scenarios := []string{}
	if scenariosRaw != "" {
		err = json.Unmarshal([]byte(scenariosRaw), &scenarios)
		if err != nil {
			glog.Errorf("error while unmarshalling scenarios %v", err)
			util.ReturnHTTPMessage(w, r, 500, "internalerror", "error parsing")
			return
		}
	}

	courses := []string{}
	if coursesRaw != "" {
		err = json.Unmarshal([]byte(coursesRaw), &courses)
		if err != nil {
			glog.Errorf("error while unmarshalling courses %v", err)
			util.ReturnHTTPMessage(w, r, 500, "internalerror", "error parsing")
			return
		}
	}

	scheduledEvent := &hfv1.ScheduledEvent{}

	hasher := sha256.New()
	hasher.Write([]byte(name))
	sha := base32.StdEncoding.WithPadding(-1).EncodeToString(hasher.Sum(nil))[:10]
	scheduledEvent.Name = "se-" + strings.ToLower(sha)

	scheduledEvent.Spec.Name = name
	scheduledEvent.Spec.Description = description
	scheduledEvent.Spec.Creator = user.Spec.Id
	scheduledEvent.Spec.StartTime = startTime
	scheduledEvent.Spec.EndTime = endTime
	scheduledEvent.Spec.RequiredVirtualMachines = requiredVMUnmarshaled
	scheduledEvent.Spec.AccessCode = accessCode

	if scenariosRaw != "" {
		scheduledEvent.Spec.Scenarios = scenarios
	}

	if coursesRaw != "" {
		scheduledEvent.Spec.Courses = courses
	}

	scheduledEvent.Status.Active = true
	scheduledEvent.Status.Finished = false
	scheduledEvent.Status.Ready = false
	scheduledEvent.Status.Provisioned = false
	scheduledEvent.Status.AccessCodeId = ""
	scheduledEvent.Status.VirtualMachineSets = []string{}

	if restrictionDisabled {
		scheduledEvent.Spec.RestrictedBind = false
	} else {
		scheduledEvent.Spec.RestrictedBind = true
		scheduledEvent.Spec.RestrictedBindValue = "se-" + strings.ToLower(sha)
	}

	scheduledEvent, err = a.hfClientSet.HobbyfarmV1().ScheduledEvents().Create(scheduledEvent)
	if err != nil {
		glog.Errorf("error creating scheduled event %v", err)
		util.ReturnHTTPMessage(w, r, 500, "internalerror", "error creating scehduled event")
		return
	}

	util.ReturnHTTPMessage(w, r, 201, "created", scheduledEvent.Name)
	return
}

func (a AdminScheduledEventServer) UpdateFunc(w http.ResponseWriter, r *http.Request) {
	user, err := a.auth.AuthNAdmin(w, r)
	if err != nil {
		util.ReturnHTTPMessage(w, r, 403, "forbidden", "no access to update scheduledevents")
		return
	}

	vars := mux.Vars(r)

	id := vars["id"]
	if id == "" {
		util.ReturnHTTPMessage(w, r, 400, "badrequest", "no ID passed in")
		return
	}

	retryErr := retry.RetryOnConflict(retry.DefaultRetry, func() error {
		scheduledEvent, err := a.hfClientSet.HobbyfarmV1().ScheduledEvents().Get(id, metav1.GetOptions{})
		if err != nil {
			glog.Error(err)
			util.ReturnHTTPMessage(w, r, 400, "badrequest", "no ID found")
			return fmt.Errorf("bad")
		}

		if scheduledEvent.Spec.Creator != user.Spec.Id {
			util.ReturnHTTPMessage(w, r, 403, "forbidden", "not creator")
			return fmt.Errorf("bad")
		}

		beginTime, err := time.Parse(time.UnixDate, scheduledEvent.Spec.StartTime)
		if err != nil {
			return err
		}

		if beginTime.Before(time.Now()) {
			util.ReturnHTTPMessage(w, r, 303, "toolate", "too late")
			return fmt.Errorf("too late")
		}

		name := r.PostFormValue("name")
		description := r.PostFormValue("description")
		startTime := r.PostFormValue("start_time")
		endTime := r.PostFormValue("end_time")
		requiredVM := r.PostFormValue("required_vms")
		accessCode := r.PostFormValue("access_code")
		scenariosRaw := r.PostFormValue("scenarios")
		coursesRaw := r.PostFormValue("courses")

		if name != "" {
			scheduledEvent.Spec.Name = name
		}
		if description != "" {
			scheduledEvent.Spec.Description = description
		}
		if startTime != "" {
			scheduledEvent.Spec.StartTime = startTime
		}
		if endTime != "" {
			scheduledEvent.Spec.EndTime = endTime
		}

		if accessCode != "" {
			scheduledEvent.Spec.AccessCode = accessCode
		}

		if requiredVM != "" {
			requiredVMUnmarshaled := map[string]map[string]int{}

			err = json.Unmarshal([]byte(requiredVM), &requiredVMUnmarshaled)
			if err != nil {
				glog.Errorf("error while unmarshaling required VM's %v", err)
				return fmt.Errorf("bad")
			}
			scheduledEvent.Spec.RequiredVirtualMachines = requiredVMUnmarshaled

			//if scheduledEvent.Status.Provisioned {
			//	scheduledEvent.Status.Provisioned = false
			//}
		}

		if coursesRaw != "" {
			if !scheduledEvent.Status.Provisioned {
				courses := []string{}
				err = json.Unmarshal([]byte(coursesRaw), &courses)
				if err != nil {
					glog.Errorf("error while unmarshaling courses %v", err)
					return fmt.Errorf("bad")
				}
				scheduledEvent.Spec.Courses = courses
			}
		}

		if scenariosRaw != "" {
			if !scheduledEvent.Status.Provisioned { // we can't change the scenarios after the scheduled event was provisioned
				scenarios := []string{}
				err = json.Unmarshal([]byte(scenariosRaw), &scenarios)
				if err != nil {
					glog.Errorf("error while unmarshaling scenarios %v", err)
					return fmt.Errorf("bad")
				}
				scheduledEvent.Spec.Scenarios = scenarios
			}
		}

		_, updateErr := a.hfClientSet.HobbyfarmV1().ScheduledEvents().Update(scheduledEvent)
		return updateErr
	})

	if retryErr != nil {
		util.ReturnHTTPMessage(w, r, 500, "error", "error attempting to update")
		return
	}

	util.ReturnHTTPMessage(w, r, 200, "updated", "")
	return
}
