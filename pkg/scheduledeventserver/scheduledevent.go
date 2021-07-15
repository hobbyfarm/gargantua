package scheduledeventserver

import (
	"crypto/sha256"
	"encoding/base32"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/golang/glog"
	"github.com/gorilla/mux"
	hfv1 "github.com/hobbyfarm/gargantua/pkg/apis/hobbyfarm.io/v1"
	"github.com/hobbyfarm/gargantua/pkg/authclient"
	hfClientset "github.com/hobbyfarm/gargantua/pkg/client/clientset/versioned"
	"github.com/hobbyfarm/gargantua/pkg/util"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/util/retry"
)

type ScheduledEventServer struct {
	auth        *authclient.AuthClient
	hfClientSet hfClientset.Interface
}

func NewScheduledEventServer(authClient *authclient.AuthClient, hfClientset hfClientset.Interface) (*ScheduledEventServer, error) {
	es := ScheduledEventServer{}

	es.hfClientSet = hfClientset
	es.auth = authClient

	return &es, nil
}

func (s ScheduledEventServer) getScheduledEvent(id string) (hfv1.ScheduledEvent, error) {

	empty := hfv1.ScheduledEvent{}

	if len(id) == 0 {
		return empty, fmt.Errorf("scheduledevent passed in was empty")
	}

	obj, err := s.hfClientSet.HobbyfarmV1().ScheduledEvents().Get(id, metav1.GetOptions{})
	if err != nil {
		return empty, fmt.Errorf("error while retrieving ScheduledEvent by id: %s with error: %v", id, err)
	}

	return *obj, nil

}

func (s ScheduledEventServer) SetupRoutes(r *mux.Router) {
	r.HandleFunc("/a/scheduledevent/list", s.ListFunc).Methods("GET")
	r.HandleFunc("/a/scheduledevent/new", s.CreateFunc).Methods("POST")
	r.HandleFunc("/a/scheduledevent/{id}", s.GetFunc).Methods("GET")
	r.HandleFunc("/a/scheduledevent/{id}", s.UpdateFunc).Methods("PUT")
	r.HandleFunc("/a/scheduledevent/delete/{id}", s.DeleteFunc).Methods("DELETE")
	glog.V(2).Infof("set up routes for admin scheduledevent server")
}

type PreparedScheduledEvent struct {
	ID string `json:"id"`
	hfv1.ScheduledEventSpec
	hfv1.ScheduledEventStatus
}

func (s ScheduledEventServer) GetFunc(w http.ResponseWriter, r *http.Request) {
	_, err := s.auth.AuthNAdmin(w, r)
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

	scheduledEvent, err := s.getScheduledEvent(scheduledEventId)

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

func (s ScheduledEventServer) ListFunc(w http.ResponseWriter, r *http.Request) {
	_, err := s.auth.AuthNAdmin(w, r)
	if err != nil {
		util.ReturnHTTPMessage(w, r, 403, "forbidden", "no access to get scheduledevents")
		return
	}

	scheduledEvents, err := s.hfClientSet.HobbyfarmV1().ScheduledEvents().List(metav1.ListOptions{})

	if err != nil {
		glog.Errorf("error while retrieving scheduledevents %v", err)
		util.ReturnHTTPMessage(w, r, 500, "error", "no scheduledevents found")
		return
	}

	preparedScheduledEvents := []PreparedScheduledEvent{} // must be declared this way so as to JSON marshal into [] instead of null
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

func (s ScheduledEventServer) CreateFunc(w http.ResponseWriter, r *http.Request) {
	user, err := s.auth.AuthNAdmin(w, r)
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
	var onDemand bool
	onDemandRaw := r.PostFormValue("on_demand")
	if onDemandRaw == "" {
		glog.Warning("scheduled event without use of on_demand flag is deprecated. please upgrade your client")
		onDemand = false
	} else {
		onDemand, err = strconv.ParseBool(onDemandRaw)
		if err != nil {
			util.ReturnHTTPMessage(w, r, 400, "badrequest", "invalid value for on_demand")
			return
		}
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

	scenarios := []string{} // must be declared this way so as to JSON marshal into [] instead of null
	if scenariosRaw != "" {
		err = json.Unmarshal([]byte(scenariosRaw), &scenarios)
		if err != nil {
			glog.Errorf("error while unmarshalling scenarios %v", err)
			util.ReturnHTTPMessage(w, r, 500, "internalerror", "error parsing")
			return
		}
	}

	courses := []string{} // must be declared this way so as to JSON marshal into [] instead of null
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
	scheduledEvent.Spec.OnDemand = onDemand
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

	scheduledEvent, err = s.hfClientSet.HobbyfarmV1().ScheduledEvents().Create(scheduledEvent)
	if err != nil {
		glog.Errorf("error creating scheduled event %v", err)
		util.ReturnHTTPMessage(w, r, 500, "internalerror", "error creating scheduled event")
		return
	}

	util.ReturnHTTPMessage(w, r, 201, "created", scheduledEvent.Name)
	return
}

func (s ScheduledEventServer) UpdateFunc(w http.ResponseWriter, r *http.Request) {
	user, err := s.auth.AuthNAdmin(w, r)
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
		scheduledEvent, err := s.hfClientSet.HobbyfarmV1().ScheduledEvents().Get(id, metav1.GetOptions{})
		if err != nil {
			glog.Error(err)
			util.ReturnHTTPMessage(w, r, 400, "badrequest", "no ID found")
			return fmt.Errorf("bad")
		}

		if scheduledEvent.Spec.Creator != user.Spec.Id {
			util.ReturnHTTPMessage(w, r, 403, "forbidden", "not creator")
			return fmt.Errorf("bad")
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
		}

		if coursesRaw != "" {
			courses := []string{} // must be declared this way so as to JSON marshal into [] instead of null
			err = json.Unmarshal([]byte(coursesRaw), &courses)
			if err != nil {
				glog.Errorf("error while unmarshaling courses %v", err)
				return fmt.Errorf("bad")
			}
			scheduledEvent.Spec.Courses = courses
		}

		if scenariosRaw != "" {
			scenarios := []string{} // must be declared this way so as to JSON marshal into [] instead of null
			err = json.Unmarshal([]byte(scenariosRaw), &scenarios)
			if err != nil {
				glog.Errorf("error while unmarshaling scenarios %v", err)
				return fmt.Errorf("bad")
			}
			scheduledEvent.Spec.Scenarios = scenarios
		}

		// if our event is already provisioned, we need to undo that and delete the corresponding access code(s) and DBC(s)
		// our scheduledeventcontroller will then provision our scheduledevent with the updated values
		if scheduledEvent.Status.Provisioned {
			now := time.Now()

			beginTime, err := time.Parse(time.UnixDate, scheduledEvent.Spec.StartTime)
			if err != nil {
				return err
			}

			// the SE's begin time has been rescheduled to the future but was already provisioned
			if now.Before(beginTime) && scheduledEvent.Status.Active {
				err = s.deleteVMSetsFromScheduledEvent(scheduledEvent)
				if err != nil {
					return err
				}
			}

			err = s.deleteScheduledEventConfig(scheduledEvent)
			if err != nil {
				return err
			}
			scheduledEvent.Status.Provisioned = false
			scheduledEvent.Status.Ready = false
			scheduledEvent.Status.Finished = false
		}

		_, updateErr := s.hfClientSet.HobbyfarmV1().ScheduledEvents().Update(scheduledEvent)
		return updateErr
	})

	if retryErr != nil {
		util.ReturnHTTPMessage(w, r, 500, "error", "error attempting to update")
		return
	}

	util.ReturnHTTPMessage(w, r, 200, "updated", "")
	return
}

func (s ScheduledEventServer) DeleteFunc(w http.ResponseWriter, r *http.Request) {
	user, err := s.auth.AuthNAdmin(w, r)
	if err != nil {
		util.ReturnHTTPMessage(w, r, 403, "forbidden", "no access to delete scheduledevents")
		return
	}

	vars := mux.Vars(r)

	id := vars["id"]
	if id == "" {
		util.ReturnHTTPMessage(w, r, 400, "badrequest", "no ID passed in")
		return
	}

	scheduledEvent, err := s.hfClientSet.HobbyfarmV1().ScheduledEvents().Get(id, metav1.GetOptions{})
	if err != nil {
		glog.Error(err)
		util.ReturnHTTPMessage(w, r, 400, "badrequest", "no ID found")
		return
	}

	if scheduledEvent.Spec.Creator != user.Spec.Id {
		util.ReturnHTTPMessage(w, r, 403, "forbidden", "not creator")
		return
	}

	err = s.deleteVMSetsFromScheduledEvent(scheduledEvent)
	if err != nil {
		util.ReturnHTTPMessage(w, r, 500, "internalerror", "error deleting scheduled event's vmsets")
		return
	}

	err = s.deleteScheduledEventConfig(scheduledEvent)
	if err != nil {
		util.ReturnHTTPMessage(w, r, 500, "internalerror", "error deleting scheduled event's access code(s) and DBC(s)")
		return
	}

	err = s.hfClientSet.HobbyfarmV1().ScheduledEvents().Delete(scheduledEvent.Name, &metav1.DeleteOptions{})

	if err != nil {
		glog.Errorf("error deleting scheduled event %v", err)
		util.ReturnHTTPMessage(w, r, 500, "internalerror", "error deleting scheduled event")
		return
	}

	util.ReturnHTTPMessage(w, r, 200, "deleted", "Deleted: "+scheduledEvent.Name)
	return
}

func (s ScheduledEventServer) deleteScheduledEventConfig(se *hfv1.ScheduledEvent) error {
	glog.V(6).Infof("ScheduledEvent %s is updated or deleted, deleting corresponding access code(s) and DBC(s)", se.Name)

	// get a list of the DBCs corresponding to this scheduled event
	dbcList, err := s.hfClientSet.HobbyfarmV1().DynamicBindConfigurations().List(metav1.ListOptions{
		LabelSelector: fmt.Sprintf("scheduledevent=%s", se.Name),
	})
	if err != nil {
		return err
	}

	// for each DBC that belongs to this edited/deleted scheduled event, delete that DBC
	for _, dbc := range dbcList.Items {
		err := s.hfClientSet.HobbyfarmV1().DynamicBindConfigurations().Delete(dbc.Name, &metav1.DeleteOptions{})
		if err != nil {
			glog.Errorf("error deleting dbc %v", err)
		}
	}

	// get a list of the access codes corresponding to this scheduled event
	acList, err := s.hfClientSet.HobbyfarmV1().AccessCodes().List(metav1.ListOptions{
		LabelSelector: fmt.Sprintf("scheduledevent=%s", se.Name),
	})
	if err != nil {
		return err
	}

	// for each access code that belongs to this edited/deleted scheduled event, delete that access code
	for _, ac := range acList.Items {
		err := s.hfClientSet.HobbyfarmV1().AccessCodes().Delete(ac.Name, &metav1.DeleteOptions{})
		if err != nil {
			glog.Errorf("error deleting access code %v", err)
		}
	}
	return nil // break (return) here because we're done with this SE.
}

func (s ScheduledEventServer) deleteVMSetsFromScheduledEvent(se *hfv1.ScheduledEvent) error {
	// get a list of the vmsets corresponding to this scheduled event
	vmsList, err := s.hfClientSet.HobbyfarmV1().VirtualMachineSets().List(metav1.ListOptions{
		LabelSelector: fmt.Sprintf("scheduledevent=%s", se.Name),
	})
	if err != nil {
		return err
	}

	// for each vmset that belongs to this to-be-stopped scheduled event, delete that vmset
	for _, vms := range vmsList.Items {
		err := s.hfClientSet.HobbyfarmV1().VirtualMachineSets().Delete(vms.Name, &metav1.DeleteOptions{})
		if err != nil {
			glog.Errorf("error deleting virtualmachineset %v", err)
		}
	}
	return nil
}
