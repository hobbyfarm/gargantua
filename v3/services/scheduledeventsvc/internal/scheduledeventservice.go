package eventservice

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	hferrors "github.com/hobbyfarm/gargantua/v3/pkg/errors"
	hflabels "github.com/hobbyfarm/gargantua/v3/pkg/labels"
	"github.com/hobbyfarm/gargantua/v3/pkg/rbac"
	"github.com/hobbyfarm/gargantua/v3/pkg/util"
	accesscodepb "github.com/hobbyfarm/gargantua/v3/protos/accesscode"
	generalpb "github.com/hobbyfarm/gargantua/v3/protos/general"
	scheduledeventpb "github.com/hobbyfarm/gargantua/v3/protos/scheduledevent"
	sessionpb "github.com/hobbyfarm/gargantua/v3/protos/session"
	"google.golang.org/protobuf/types/known/wrapperspb"

	"github.com/golang/glog"
	"github.com/gorilla/mux"
)

const (
	resourcePlural = rbac.ResourcePluralEvent
)

type PreparedScheduledEvent struct {
	Id                      string                       `json:"id"`
	Creator                 string                       `json:"creator"`
	Name                    string                       `json:"event_name"`
	Description             string                       `json:"description"`
	StartTime               string                       `json:"start_time"`
	EndTime                 string                       `json:"end_time"`
	OnDemand                bool                         `json:"on_demand"`    // whether or not to provision VMs on-demand
	RequiredVirtualMachines map[string]map[string]uint32 `json:"required_vms"` // map of environment to a map of strings it should be environment: vm template: count
	AccessCode              string                       `json:"access_code"`
	RestrictedBind          bool                         `json:"restricted_bind"` // if restricted_bind is true, we need to make the scenario sessions when they get created only bind to vmsets that are created by this scheduledevent
	RestrictedBindValue     string                       `json:"restricted_bind_value"`
	Printable               bool                         `json:"printable"`
	Scenarios               []string                     `json:"scenarios"`
	Courses                 []string                     `json:"courses"`
	*scheduledeventpb.ScheduledEventStatus
}

type PreparedOTAC struct {
	Id                string `json:"name"`
	User              string `json:"user"`
	RedeemedTimestamp string `json:"redeemed_timestamp"`
	MaxDuration       string `json:"max_duration"`
}

func (s ScheduledEventServer) getPreparedScheduledEvent(scheduledEvent *scheduledeventpb.ScheduledEvent) PreparedScheduledEvent {
	return PreparedScheduledEvent{
		Id:                      scheduledEvent.GetId(),
		Creator:                 scheduledEvent.GetCreator(),
		Name:                    scheduledEvent.GetName(),
		Description:             scheduledEvent.GetDescription(),
		StartTime:               scheduledEvent.GetStartTime(),
		EndTime:                 scheduledEvent.GetEndTime(),
		OnDemand:                scheduledEvent.GetOnDemand(),
		RequiredVirtualMachines: util.ConvertMapStruct(scheduledEvent.GetRequiredVms(), util.GetRawVMTemplateCountMap),
		AccessCode:              scheduledEvent.GetAccessCode(),
		RestrictedBind:          scheduledEvent.GetRestrictedBind(),
		RestrictedBindValue:     scheduledEvent.GetRestrictedBindValue(),
		Printable:               scheduledEvent.GetPrintable(),
		Scenarios:               scheduledEvent.GetScenarios(),
		Courses:                 scheduledEvent.GetCourses(),
		ScheduledEventStatus:    scheduledEvent.GetStatus(),
	}

}

func (s ScheduledEventServer) GetFunc(w http.ResponseWriter, r *http.Request) {
	user, err := rbac.AuthenticateRequest(r, s.authnClient)
	if err != nil {
		util.ReturnHTTPMessage(w, r, 401, "unauthorized", "authentication failed")
		return
	}

	impersonatedUserId := user.GetId()
	authrResponse, err := rbac.AuthorizeSimple(r, s.authrClient, impersonatedUserId, rbac.HobbyfarmPermission(resourcePlural, rbac.VerbGet))
	if err != nil || !authrResponse.Success {
		util.ReturnHTTPMessage(w, r, 403, "forbidden", "no access to get scheduledEvent")
		return
	}

	vars := mux.Vars(r)

	scheduledEventId := vars["id"]

	if len(scheduledEventId) == 0 {
		util.ReturnHTTPMessage(w, r, 400, "bad request", "no scheduledEvent id passed in")
		return
	}

	scheduledEvent, err := s.internalScheduledEventServer.GetScheduledEvent(r.Context(), &generalpb.GetRequest{Id: scheduledEventId})

	if err != nil {
		glog.Errorf("error while retrieving scheduledEvent: %s", hferrors.GetErrorMessage(err))
		if hferrors.IsGrpcNotFound(err) {
			errMsg := fmt.Sprintf("error while retrieving scheduledEvent: scheduledEvent %s not found", scheduledEventId)
			util.ReturnHTTPMessage(w, r, 404, "not found", errMsg)
			return
		}
		util.ReturnHTTPMessage(w, r, http.StatusInternalServerError, "internalerror", "internal error while retrieving scheduledEvent")
		return
	}

	preparedScheduledEvent := s.getPreparedScheduledEvent(scheduledEvent)

	encodedScheduledEvent, err := json.Marshal(preparedScheduledEvent)
	if err != nil {
		glog.Error(err)
	}
	util.ReturnHTTPContent(w, r, 200, "success", encodedScheduledEvent)

	glog.V(2).Infof("retrieved scheduledEvent %s", scheduledEvent.GetId())
}

func (s ScheduledEventServer) ListFunc(w http.ResponseWriter, r *http.Request) {
	user, err := rbac.AuthenticateRequest(r, s.authnClient)
	if err != nil {
		util.ReturnHTTPMessage(w, r, 401, "unauthorized", "authentication failed")
		return
	}

	impersonatedUserId := user.GetId()
	authrResponse, err := rbac.AuthorizeSimple(r, s.authrClient, impersonatedUserId, rbac.HobbyfarmPermission(resourcePlural, rbac.VerbList))
	if err != nil || !authrResponse.Success {
		util.ReturnHTTPMessage(w, r, 403, "forbidden", "no access to list scheduledEvents")
		return
	}

	scheduledEventList, err := s.internalScheduledEventServer.ListScheduledEvent(r.Context(), &generalpb.ListOptions{})

	if err != nil {
		glog.Errorf("error while retrieving scheduledevents: %s", hferrors.GetErrorMessage(err))
		util.ReturnHTTPMessage(w, r, 500, "error", "no scheduledevents found")
		return
	}

	scheduledEvents := scheduledEventList.GetScheduledevents()
	preparedScheduledEvents := make([]PreparedScheduledEvent, 0, len(scheduledEvents))
	for _, event := range scheduledEventList.GetScheduledevents() {
		preparedScheduledEvents = append(preparedScheduledEvents, s.getPreparedScheduledEvent(event))
	}

	encodedScheduledEvents, err := json.Marshal(preparedScheduledEvents)
	if err != nil {
		glog.Error(err)
	}
	util.ReturnHTTPContent(w, r, 200, "success", encodedScheduledEvents)

	glog.V(2).Infof("listed scheduled events")
}

func (s ScheduledEventServer) CreateFunc(w http.ResponseWriter, r *http.Request) {
	user, err := rbac.AuthenticateRequest(r, s.authnClient)
	if err != nil {
		util.ReturnHTTPMessage(w, r, 401, "unauthorized", "authentication failed")
		return
	}

	impersonatedUserId := user.GetId()
	authrResponse, err := rbac.AuthorizeSimple(r, s.authrClient, impersonatedUserId, rbac.HobbyfarmPermission(resourcePlural, rbac.VerbCreate))
	if err != nil || !authrResponse.Success {
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
	sharedVmsRaw := r.PostFormValue("shared_vms")

	// restrictedBind := strings.ToLower(restrictionDisabledRaw) == "false" || restrictionDisabled == ""
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

	eventId, err := s.internalScheduledEventServer.CreateScheduledEvent(r.Context(), &scheduledeventpb.CreateScheduledEventRequest{
		Name:           name,
		Description:    description,
		Creator:        user.GetId(),
		StartTime:      startTime,
		EndTime:        endTime,
		OnDemand:       onDemand,
		Printable:      printable,
		RestrictedBind: !restrictionDisabled,
		RequiredVmsRaw: requiredVM,
		AccessCode:     accessCode,
		ScenariosRaw:   scenariosRaw,
		CoursesRaw:     coursesRaw,
		SharedVmsRaw:   sharedVmsRaw,
	})

	if err != nil {
		glog.Errorf("error creating scheduled event: %s", hferrors.GetErrorMessage(err))
		util.ReturnHTTPMessage(w, r, 500, "internalerror", "error creating scheduled event")
		return
	}

	id := eventId.GetId()

	_, err = s.internalScheduledEventServer.UpdateScheduledEventStatus(r.Context(), &scheduledeventpb.UpdateScheduledEventStatusRequest{
		Id:          id,
		Vmsets:      &scheduledeventpb.VMSetsWrapper{Value: []string{}},
		Active:      wrapperspb.Bool(true),
		Provisioned: wrapperspb.Bool(false),
		Ready:       wrapperspb.Bool(false),
		Finished:    wrapperspb.Bool(false),
	})

	if err != nil {
		glog.Errorf("error updating status for scheduled event: %s", hferrors.GetErrorMessage(err))
		util.ReturnHTTPMessage(w, r, 500, "internalerror", "error creating scheduled event")
		return
	}

	util.ReturnHTTPMessage(w, r, 201, "created", id)
}

func (s ScheduledEventServer) UpdateFunc(w http.ResponseWriter, r *http.Request) {
	user, err := rbac.AuthenticateRequest(r, s.authnClient)
	if err != nil {
		util.ReturnHTTPMessage(w, r, 401, "unauthorized", "authentication failed")
		return
	}

	impersonatedUserId := user.GetId()
	authrResponse, err := rbac.AuthorizeSimple(r, s.authrClient, impersonatedUserId, rbac.HobbyfarmPermission(resourcePlural, rbac.VerbUpdate))
	if err != nil || !authrResponse.Success {
		util.ReturnHTTPMessage(w, r, 403, "forbidden", "no access to update scheduledevents")
		return
	}

	vars := mux.Vars(r)

	id := vars["id"]
	if id == "" {
		util.ReturnHTTPMessage(w, r, 400, "badrequest", "no ID passed in")
		return
	}

	name := r.PostFormValue("name")
	description := r.PostFormValue("description")
	startTime := r.PostFormValue("start_time")
	endTime := r.PostFormValue("end_time")
	requiredVM := r.PostFormValue("required_vms")
	accessCode := r.PostFormValue("access_code")
	scenariosRaw := r.PostFormValue("scenarios")
	coursesRaw := r.PostFormValue("courses")
	sharedVmsRaw := r.PostFormValue("shared_vms")
	onDemandRaw := r.PostFormValue("on_demand")
	restrictionDisabledRaw := r.PostFormValue("disable_restriction")
	printableRaw := r.PostFormValue("printable")

	var onDemandWrapper *wrapperspb.BoolValue
	if onDemandRaw != "" {
		onDemand, err := strconv.ParseBool(onDemandRaw)
		if err != nil {
			util.ReturnHTTPMessage(w, r, 400, "badrequest", "invalid value for on_demand")
			return
		}
		onDemandWrapper = wrapperspb.Bool(onDemand)
	}

	var printableWrapper *wrapperspb.BoolValue
	if printableRaw != "" {
		printable, err := strconv.ParseBool(printableRaw)
		if err != nil {
			util.ReturnHTTPMessage(w, r, 400, "badrequest", "invalid value for printable")
			return
		}
		printableWrapper = wrapperspb.Bool(printable)
	}

	var restrictedBindWrapper *wrapperspb.BoolValue
	if restrictionDisabledRaw != "" {
		// if restrictionDisabledRaw == false -> restricted bind is not disabled and thus true
		restrictedBind := strings.ToLower(restrictionDisabledRaw) == "false"
		restrictedBindWrapper = wrapperspb.Bool(restrictedBind)
	}

	_, err = s.internalScheduledEventServer.UpdateScheduledEvent(r.Context(), &scheduledeventpb.UpdateScheduledEventRequest{
		Id:             id,
		Name:           name,
		Description:    description,
		StartTime:      startTime,
		EndTime:        endTime,
		OnDemand:       onDemandWrapper,
		Printable:      printableWrapper,
		RestrictedBind: restrictedBindWrapper,
		RequiredVmsRaw: requiredVM,
		AccessCode:     accessCode,
		ScenariosRaw:   scenariosRaw,
		CoursesRaw:     coursesRaw,
		SharedVmsRaw:   sharedVmsRaw,
	})

	if err != nil {
		glog.Error(hferrors.GetErrorMessage(err))
		util.ReturnHTTPMessage(w, r, 500, "error", "error attempting to update")
		return
	}

	util.ReturnHTTPMessage(w, r, 200, "updated", "")
}

func (s ScheduledEventServer) DeleteFunc(w http.ResponseWriter, r *http.Request) {
	user, err := rbac.AuthenticateRequest(r, s.authnClient)
	if err != nil {
		util.ReturnHTTPMessage(w, r, 401, "unauthorized", "authentication failed")
		return
	}

	impersonatedUserId := user.GetId()
	authrResponse, err := rbac.AuthorizeSimple(r, s.authrClient, impersonatedUserId, rbac.HobbyfarmPermission(resourcePlural, rbac.VerbDelete))
	if err != nil || !authrResponse.Success {
		util.ReturnHTTPMessage(w, r, 403, "forbidden", "no access to delete scheduledevents")
		return
	}

	vars := mux.Vars(r)

	id := vars["id"]
	if id == "" {
		util.ReturnHTTPMessage(w, r, 400, "badrequest", "no ID passed in")
		return
	}

	scheduledEvent, err := s.internalScheduledEventServer.GetScheduledEvent(r.Context(), &generalpb.GetRequest{Id: id})
	if err != nil {
		glog.Error(hferrors.GetErrorMessage(err))
		if hferrors.IsGrpcNotFound(err) {
			errMsg := fmt.Sprintf("error deleting scheduled event: scheduled event %s not found", id)
			util.ReturnHTTPMessage(w, r, 404, "not found", errMsg)
			return
		} else {
			util.ReturnHTTPMessage(w, r, 500, "internalerror", "error deleting scheduled event")
			return
		}
	}

	_, err = s.vmsetClient.DeleteCollectionVMSet(r.Context(), &generalpb.ListOptions{
		LabelSelector: fmt.Sprintf("%s=%s", hflabels.ScheduledEventLabel, id),
	})
	if err != nil {
		util.ReturnHTTPMessage(w, r, 500, "internalerror", "error deleting scheduled event's VMSets")
		return
	}

	err = s.finishSessions(r.Context(), scheduledEvent)
	if err != nil {
		util.ReturnHTTPMessage(w, r, 500, "internalerror", "error terminating scheduled event's sessions")
		return
	}

	_, err = s.progressClient.DeleteCollectionProgress(r.Context(), &generalpb.ListOptions{
		LabelSelector: fmt.Sprintf("%s=%s", hflabels.ScheduledEventLabel, id),
	})
	if err != nil {
		util.ReturnHTTPMessage(w, r, 500, "internalerror", "error deleting scheduled event's progress data")
		return
	}

	_, err = s.dbconfigClient.DeleteCollectionDynamicBindConfig(r.Context(), &generalpb.ListOptions{
		LabelSelector: fmt.Sprintf("%s=%s", hflabels.ScheduledEventLabel, id),
	})
	if err != nil {
		util.ReturnHTTPMessage(w, r, 500, "internalerror", "error deleting scheduled event's db configs")
		return
	}

	_, err = s.acClient.DeleteCollectionAc(r.Context(), &generalpb.ListOptions{
		LabelSelector: fmt.Sprintf("%s=%s", hflabels.ScheduledEventLabel, id),
	})
	if err != nil {
		util.ReturnHTTPMessage(w, r, 500, "internalerror", "error deleting scheduled event's access codes")
		return
	}

	_, err = s.internalScheduledEventServer.DeleteScheduledEvent(r.Context(), &generalpb.ResourceId{Id: id})
	if err != nil {
		glog.Errorf("error deleting scheduled event %s", hferrors.GetErrorMessage(err))
		util.ReturnHTTPMessage(w, r, 500, "internalerror", "error deleting scheduled event")
		return
	}

	util.ReturnHTTPMessage(w, r, 200, "deleted", fmt.Sprintf("Deleted: %s", id))
}

func (s ScheduledEventServer) GetOTACsFunc(w http.ResponseWriter, r *http.Request) {
	user, err := rbac.AuthenticateRequest(r, s.authnClient)
	if err != nil {
		util.ReturnHTTPMessage(w, r, 401, "unauthorized", "authentication failed")
		return
	}

	impersonatedUserId := user.GetId()
	authrResponse, err := rbac.AuthorizeSimple(r, s.authrClient, impersonatedUserId, rbac.HobbyfarmPermission(resourcePlural, rbac.VerbList))
	if err != nil || !authrResponse.Success {
		util.ReturnHTTPMessage(w, r, 403, "forbidden", "no access to list scheduledevents")
		return
	}

	vars := mux.Vars(r)

	id := vars["id"]
	if id == "" {
		util.ReturnHTTPMessage(w, r, 400, "badrequest", "no ID passed in")
		return
	}

	otacList, err := s.acClient.ListOtac(r.Context(), &generalpb.ListOptions{
		LabelSelector: fmt.Sprintf("%s=%s", hflabels.ScheduledEventLabel, id),
	})
	if err != nil {
		glog.Error(hferrors.GetErrorMessage(err))
		util.ReturnHTTPMessage(w, r, 500, "internalerror", "error retreiving OTACs")
		return
	}

	var otacs []PreparedOTAC
	for _, otac := range otacList.GetOtacs() {
		otacs = append(otacs, PreparedOTAC{
			Id:                otac.GetId(),
			User:              otac.GetUser(),
			RedeemedTimestamp: otac.GetRedeemedTimestamp(),
			MaxDuration:       otac.GetMaxDuration(),
		})
	}

	encoded, err := json.Marshal(otacs)
	if err != nil {
		glog.Errorf("error marshalling prepared otacs: %v", err)
		util.ReturnHTTPMessage(w, r, 500, "internalerror", "error listing otacs")
		return
	}

	util.ReturnHTTPContent(w, r, 200, "success", encoded)

	glog.V(4).Infof("listed OTACs for SE %s", id)
}

func (s ScheduledEventServer) DeleteOTACFunc(w http.ResponseWriter, r *http.Request) {
	user, err := rbac.AuthenticateRequest(r, s.authnClient)
	if err != nil {
		util.ReturnHTTPMessage(w, r, 401, "unauthorized", "authentication failed")
		return
	}

	impersonatedUserId := user.GetId()
	authrResponse, err := rbac.AuthorizeSimple(r, s.authrClient, impersonatedUserId, rbac.HobbyfarmPermission(resourcePlural, rbac.VerbUpdate))
	if err != nil || !authrResponse.Success {
		util.ReturnHTTPMessage(w, r, 403, "forbidden", "no access to update scheduledevents")
		return
	}

	vars := mux.Vars(r)

	otac := vars["otac"]
	if otac == "" {
		util.ReturnHTTPMessage(w, r, 400, "badrequest", "no ID passed in")
		return
	}

	_, err = s.acClient.DeleteOtac(r.Context(), &generalpb.ResourceId{Id: otac})
	if err != nil {
		glog.Error(hferrors.GetErrorMessage(err))
		util.ReturnHTTPMessage(w, r, 500, "internalerror", "error deleting OTACs")
		return
	}

	util.ReturnHTTPMessage(w, r, 200, "success", "deleted OTAC")
}

func (s ScheduledEventServer) GenerateOTACsFunc(w http.ResponseWriter, r *http.Request) {
	user, err := rbac.AuthenticateRequest(r, s.authnClient)
	if err != nil {
		util.ReturnHTTPMessage(w, r, 401, "unauthorized", "authentication failed")
		return
	}

	impersonatedUserId := user.GetId()
	authrResponse, err := rbac.AuthorizeSimple(r, s.authrClient, impersonatedUserId, rbac.HobbyfarmPermission(resourcePlural, rbac.VerbUpdate))
	if err != nil || !authrResponse.Success {
		util.ReturnHTTPMessage(w, r, 403, "forbidden", "no access to update scheduledevents")
		return
	}

	vars := mux.Vars(r)

	id := vars["id"]
	if id == "" {
		util.ReturnHTTPMessage(w, r, 400, "badrequest", "no ID passed in")
		return
	}

	countFormValue := vars["count"]
	if countFormValue == "" {
		util.ReturnHTTPMessage(w, r, 400, "badrequest", "no count passed in")
		return
	}
	count, err := strconv.Atoi(countFormValue)
	if err != nil {
		glog.Error(err)
		util.ReturnHTTPMessage(w, r, 404, "badrequest", "invalid count given")
		return
	}

	maxDurationValue := r.PostFormValue("max_duration")

	scheduledEvent, err := s.internalScheduledEventServer.GetScheduledEvent(r.Context(), &generalpb.GetRequest{Id: id})
	if err != nil {
		glog.Error(hferrors.GetErrorMessage(err))
		if hferrors.IsGrpcNotFound(err) {
			errorMsg := fmt.Sprintf("error generating OTACs: scheduledEvent %s not found", id)
			util.ReturnHTTPMessage(w, r, http.StatusNotFound, "not found", errorMsg)
			return
		}
		errorMsg := fmt.Sprintf("error generating OTACs: error while retrieving scheduledEvent %s", id)
		util.ReturnHTTPMessage(w, r, http.StatusInternalServerError, "internalerror", errorMsg)
		return
	}

	var otacs []PreparedOTAC

	for i := 0; i < count; i++ {
		// Generate an access code that can not be guessed
		otac, err := s.acClient.CreateOtac(r.Context(), &accesscodepb.CreateOtacRequest{
			SeName:      scheduledEvent.GetId(),
			SeUid:       scheduledEvent.GetUid(),
			MaxDuration: maxDurationValue,
		})
		if err != nil {
			glog.Errorf("error creating one time access code %s", hferrors.GetErrorMessage(err))
			continue
		}
		otacs = append(otacs, PreparedOTAC{
			Id:                otac.GetId(),
			User:              otac.GetUser(),
			RedeemedTimestamp: otac.GetRedeemedTimestamp(),
			MaxDuration:       otac.GetMaxDuration(),
		})
	}

	encoded, err := json.Marshal(otacs)
	if err != nil {
		glog.Errorf("error marshalling prepared otacs: %v", err)
		util.ReturnHTTPMessage(w, r, 500, "internalerror", "error listing generated otacs")
		return
	}

	util.ReturnHTTPContent(w, r, 200, "success", encoded)

	glog.V(4).Infof("generated %d new OTACs for SE %s", count, id)
}

func (s ScheduledEventServer) finishSessions(ctx context.Context, se *scheduledeventpb.ScheduledEvent) error {
	// get a list of sessions for the user
	sessionList, err := s.sessionClient.ListSession(ctx, &generalpb.ListOptions{
		LabelSelector: fmt.Sprintf("%s=%s", hflabels.AccessCodeLabel, se.GetAccessCode()),
	})
	if err != nil {
		glog.Errorf("error updating session %s", hferrors.GetErrorMessage(err))
		return fmt.Errorf("error attempting to update: failed to list sessions")
	}

	now := time.Now().Format(time.UnixDate)

	for _, session := range sessionList.GetSessions() {
		_, err = s.sessionClient.UpdateSessionStatus(ctx, &sessionpb.UpdateSessionStatusRequest{
			Id:             session.GetId(),
			Active:         wrapperspb.Bool(false),
			Finished:       wrapperspb.Bool(false),
			ExpirationTime: now,
		})
		if err != nil {
			glog.Errorf("error updating session %s", hferrors.GetErrorMessage(err))
			return fmt.Errorf("error attempting to update")
		}
	}
	return nil
}
