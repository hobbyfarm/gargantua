package scheduledeventserver

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/hobbyfarm/gargantua/pkg/rbacclient"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/golang/glog"
	"github.com/gorilla/mux"
	hfv1 "github.com/hobbyfarm/gargantua/pkg/apis/hobbyfarm.io/v1"
	"github.com/hobbyfarm/gargantua/pkg/authclient"
	hfClientset "github.com/hobbyfarm/gargantua/pkg/client/clientset/versioned"
	"github.com/hobbyfarm/gargantua/pkg/controllers/scheduledevent"
	"github.com/hobbyfarm/gargantua/pkg/sessionserver"
	"github.com/hobbyfarm/gargantua/pkg/util"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/util/retry"
)

const (
	resourcePlural = "scheduledevents"
)

type ScheduledEventServer struct {
	auth        *authclient.AuthClient
	hfClientSet hfClientset.Interface
	ctx         context.Context
}

func NewScheduledEventServer(authClient *authclient.AuthClient, hfClientset hfClientset.Interface, ctx context.Context) (*ScheduledEventServer, error) {
	es := ScheduledEventServer{}

	es.hfClientSet = hfClientset
	es.auth = authClient
	es.ctx = ctx

	return &es, nil
}

func (s ScheduledEventServer) getScheduledEvent(id string) (hfv1.ScheduledEvent, error) {

	empty := hfv1.ScheduledEvent{}

	if len(id) == 0 {
		return empty, fmt.Errorf("scheduledevent passed in was empty")
	}

	obj, err := s.hfClientSet.HobbyfarmV1().ScheduledEvents(util.GetReleaseNamespace()).Get(s.ctx, id, metav1.GetOptions{})
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
	_, err := s.auth.AuthGrant(rbacclient.RbacRequest().HobbyfarmPermission(resourcePlural, rbacclient.VerbGet), w, r)

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
		util.ReturnHTTPMessage(w, r, http.StatusNotFound, "error", "no scheduledEvent with given ID found")
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
	_, err := s.auth.AuthGrant(rbacclient.RbacRequest().HobbyfarmPermission(resourcePlural, rbacclient.VerbList), w, r)
	if err != nil {
		util.ReturnHTTPMessage(w, r, 403, "forbidden", "no access to get scheduledevents")
		return
	}

	scheduledEvents, err := s.hfClientSet.HobbyfarmV1().ScheduledEvents(util.GetReleaseNamespace()).List(s.ctx, metav1.ListOptions{})

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
	user, err := s.auth.AuthGrant(rbacclient.RbacRequest().HobbyfarmPermission(resourcePlural, rbacclient.VerbCreate), w, r)

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

	var printable bool
	printableRaw := r.PostFormValue("printable")
	if printableRaw == "" {
		glog.Warning("scheduled event without use of printable flag is deprecated. please upgrade your client")
		printable = false
	} else {
		printable, err = strconv.ParseBool(printableRaw)
		if err != nil {
			util.ReturnHTTPMessage(w, r, 400, "badrequest", "invalid value for printable")
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
	random := util.RandStringRunes(16)
	scheduledEvent.Name = "se-" + util.GenerateResourceName("se", random, 10)

	scheduledEvent.Spec.Name = name
	scheduledEvent.Spec.Description = description
	scheduledEvent.Spec.Creator = user.Spec.Id
	scheduledEvent.Spec.StartTime = startTime
	scheduledEvent.Spec.EndTime = endTime
	scheduledEvent.Spec.OnDemand = onDemand
	scheduledEvent.Spec.Printable = printable
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
		scheduledEvent.Spec.RestrictedBindValue = scheduledEvent.Name
	}

	scheduledEvent, err = s.hfClientSet.HobbyfarmV1().ScheduledEvents(util.GetReleaseNamespace()).Create(s.ctx, scheduledEvent, metav1.CreateOptions{})
	if err != nil {
		glog.Errorf("error creating scheduled event %v", err)
		util.ReturnHTTPMessage(w, r, 500, "internalerror", "error creating scheduled event")
		return
	}

	util.ReturnHTTPMessage(w, r, 201, "created", scheduledEvent.Name)
	return
}

func (s ScheduledEventServer) UpdateFunc(w http.ResponseWriter, r *http.Request) {
	user, err := s.auth.AuthGrant(rbacclient.RbacRequest().HobbyfarmPermission(resourcePlural, rbacclient.VerbUpdate), w, r)
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
		scheduledEvent, err := s.hfClientSet.HobbyfarmV1().ScheduledEvents(util.GetReleaseNamespace()).Get(s.ctx, id, metav1.GetOptions{})
		if err != nil {
			glog.Error(err)
			util.ReturnHTTPMessage(w, r, 404, "badrequest", "no scheduledEvent found with given ID")
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
		onDemandRaw := r.PostFormValue("on_demand")
		restrictionDisabledRaw := r.PostFormValue("disable_restriction")
		printableRaw := r.PostFormValue("printable")

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

		restrictionDisabled := scheduledEvent.Spec.RestrictedBind

		if restrictionDisabledRaw != "" {
			if strings.ToLower(restrictionDisabledRaw) == "false" {
				restrictionDisabled = false
			} else {
				restrictionDisabled = true
			}
		}
		if restrictionDisabled {
			scheduledEvent.Spec.RestrictedBind = false
			scheduledEvent.Spec.RestrictedBindValue = ""
		} else {
			scheduledEvent.Spec.RestrictedBind = true
			scheduledEvent.Spec.RestrictedBindValue = scheduledEvent.Name
		}

		onDemand := scheduledEvent.Spec.OnDemand
		onDemandBeforeUpdate := onDemand

		if onDemandRaw != "" {
			onDemand, err = strconv.ParseBool(onDemandRaw)
			if err != nil {
				util.ReturnHTTPMessage(w, r, 400, "badrequest", "invalid value for on_demand")
				return err
			}
			if onDemand && !scheduledEvent.Spec.OnDemand {
				glog.Errorf("ScheduledEvent %s changed to onDemand, deleting corresponding VMSets.", scheduledEvent.Name)
				err = s.deleteVMSetsFromScheduledEvent(scheduledEvent)
				if err != nil {
					glog.Errorf("Deleting vmset failed: %v", err)
				}
			}
		}
		scheduledEvent.Spec.OnDemand = onDemand

		var printable bool
		if printableRaw != "" {
			printable, err = strconv.ParseBool(printableRaw)
			if err != nil {
				util.ReturnHTTPMessage(w, r, 400, "badrequest", "invalid value for on_demand")
				return err
			} else {
				scheduledEvent.Spec.Printable = printable
			}
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
			// OR the on demand setting has been removed completely.
			if (now.Before(beginTime) && scheduledEvent.Status.Active) || (!onDemandBeforeUpdate && onDemand) {
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

		_, updateErr := s.hfClientSet.HobbyfarmV1().ScheduledEvents(util.GetReleaseNamespace()).Update(s.ctx, scheduledEvent, metav1.UpdateOptions{})
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
	user, err := s.auth.AuthGrant(rbacclient.RbacRequest().HobbyfarmPermission(resourcePlural, rbacclient.VerbDelete), w, r)
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

	scheduledEvent, err := s.hfClientSet.HobbyfarmV1().ScheduledEvents(util.GetReleaseNamespace()).Get(s.ctx, id, metav1.GetOptions{})
	if err != nil {
		glog.Error(err)
		util.ReturnHTTPMessage(w, r, 404, "badrequest", "no scheduledEvent found with given ID")
		return
	}

	if scheduledEvent.Spec.Creator != user.Spec.Id {
		util.ReturnHTTPMessage(w, r, 403, "forbidden", "not creator")
		return
	}

	err = s.deleteVMSetsFromScheduledEvent(scheduledEvent)
	if err != nil {
		util.ReturnHTTPMessage(w, r, 500, "internalerror", "error deleting scheduled event's VMSets")
		return
	}

	err = s.finishSessions(scheduledEvent)
	if err != nil {
		util.ReturnHTTPMessage(w, r, 500, "internalerror", err.Error())
		return
	}

	err = s.deleteScheduledEventConfig(scheduledEvent)
	if err != nil {
		util.ReturnHTTPMessage(w, r, 500, "internalerror", "error deleting scheduled event's access code(s) and DBC(s)")
		return
	}

	err = s.hfClientSet.HobbyfarmV1().ScheduledEvents(util.GetReleaseNamespace()).Delete(s.ctx, scheduledEvent.Name, metav1.DeleteOptions{})

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

	// delete all DBCs corresponding to this scheduled event
	err := s.hfClientSet.HobbyfarmV1().DynamicBindConfigurations(util.GetReleaseNamespace()).DeleteCollection(s.ctx, metav1.DeleteOptions{}, metav1.ListOptions{
		LabelSelector: fmt.Sprintf("%s=%s", scheduledevent.ScheduledEventLabel, se.Name),
	})
	if err != nil {
		return err
	}

	// for each access code that belongs to this edited/deleted scheduled event, delete that access code
	err = s.hfClientSet.HobbyfarmV1().AccessCodes(util.GetReleaseNamespace()).DeleteCollection(s.ctx, metav1.DeleteOptions{}, metav1.ListOptions{
		LabelSelector: fmt.Sprintf("%s=%s", scheduledevent.ScheduledEventLabel, se.Name),
	})
	if err != nil {
		return err
	}

	return nil // break (return) here because we're done with this SE.
}

func (s ScheduledEventServer) deleteVMSetsFromScheduledEvent(se *hfv1.ScheduledEvent) error {
	// delete all vmsets corresponding to this scheduled event
	err := s.hfClientSet.HobbyfarmV1().VirtualMachineSets(util.GetReleaseNamespace()).DeleteCollection(s.ctx, metav1.DeleteOptions{}, metav1.ListOptions{
		LabelSelector: fmt.Sprintf("%s=%s", scheduledevent.ScheduledEventLabel, se.Name),
	})
	if err != nil {
		return err
	}

	return nil
}

func (s ScheduledEventServer) finishSessions(se *hfv1.ScheduledEvent) error {
	// get a list of sessions for the user
	sessionList, err := s.hfClientSet.HobbyfarmV1().Sessions(util.GetReleaseNamespace()).List(s.ctx, metav1.ListOptions{
		LabelSelector: fmt.Sprintf("%s=%s", sessionserver.AccessCodeLabel, se.Spec.AccessCode),
	})

	now := time.Now().Format(time.UnixDate)

	for _, session := range sessionList.Items {
		retryErr := retry.RetryOnConflict(retry.DefaultRetry, func() error {
			result, getErr := s.hfClientSet.HobbyfarmV1().Sessions(util.GetReleaseNamespace()).Get(s.ctx, session.Spec.Id, metav1.GetOptions{})
			if getErr != nil {
				return fmt.Errorf("error retrieving latest version of session %s: %v", session.Spec.Id, getErr)
			}

			result.Status.ExpirationTime = now
			result.Status.Active = false
			result.Status.Finished = false

			_, updateErr := s.hfClientSet.HobbyfarmV1().Sessions(util.GetReleaseNamespace()).Update(s.ctx, result, metav1.UpdateOptions{})
			glog.V(4).Infof("updated result for session")

			return updateErr
		})

		if retryErr != nil {
			glog.Errorf("error updating session %v", err)
			return fmt.Errorf("error attempting to update")
		}
	}
	return nil
}
