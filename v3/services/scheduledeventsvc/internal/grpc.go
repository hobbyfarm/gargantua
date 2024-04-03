package eventservice

import (
	"context"
	"strconv"

	"github.com/hobbyfarm/gargantua/v3/protos/general"
	eventProto "github.com/hobbyfarm/gargantua/v3/protos/scheduledevent"

	"github.com/golang/glog"
	"github.com/golang/protobuf/ptypes/empty"
	hfv1 "github.com/hobbyfarm/gargantua/v3/pkg/apis/hobbyfarm.io/v1"
	hfClientset "github.com/hobbyfarm/gargantua/v3/pkg/client/clientset/versioned"
	hfClientsetv1 "github.com/hobbyfarm/gargantua/v3/pkg/client/clientset/versioned/typed/hobbyfarm.io/v1"
	hfInformers "github.com/hobbyfarm/gargantua/v3/pkg/client/informers/externalversions"
	listersv1 "github.com/hobbyfarm/gargantua/v3/pkg/client/listers/hobbyfarm.io/v1"
	hferrors "github.com/hobbyfarm/gargantua/v3/pkg/errors"
	"github.com/hobbyfarm/gargantua/v3/pkg/util"
	"google.golang.org/grpc/codes"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/retry"
)

type GrpcScheduledEventServer struct {
	eventProto.UnimplementedScheduledEventSvcServer
	eventClient hfClientsetv1.ScheduledEventInterface
	eventLister listersv1.ScheduledEventLister
	eventSynced cache.InformerSynced
}

func NewGrpcScheduledEventServer(hfClientSet hfClientset.Interface, hfInformerFactory hfInformers.SharedInformerFactory) *GrpcScheduledEventServer {
	return &GrpcScheduledEventServer{
		eventClient: hfClientSet.HobbyfarmV1().ScheduledEvents(util.GetReleaseNamespace()),
		eventLister: hfInformerFactory.Hobbyfarm().V1().ScheduledEvents().Lister(),
		eventSynced: hfInformerFactory.Hobbyfarm().V1().ScheduledEvents().Informer().HasSynced,
	}
}

func (s *GrpcScheduledEventServer) CreateScheduledEvent(ctx context.Context, req *eventProto.CreateScheduledEventRequest) (*empty.Empty, error) {
	name := req.GetName()
	description := req.GetDescription()
	creator := req.GetCreator()
	startTime := req.GetStartTime()
	endTime := req.GetEndTime()
	onDemandRaw := req.GetOnDemandRaw()
	printableRaw := req.GetPrintableRaw()
	restrictedBind := req.GetRestrictedBind()
	reqVmsRaw := req.GetRequiredVmsRaw()
	accessCode := req.GetAccessCode()
	scenariosRaw := req.GetScenariosRaw()
	coursesRaw := req.GetCoursesRaw()
	labels := req.GetLabels()

	requiredStringParams := map[string]string{
		"name":           name,
		"description":    description,
		"creator":        creator,
		"startTime":      startTime,
		"endTime":        endTime,
		"onDemandRaw":    onDemandRaw,
		"printableRaw":   printableRaw,
		"requiredVmsRaw": reqVmsRaw,
		"accessCode":     accessCode,
	}
	for param, value := range requiredStringParams {
		if value == "" {
			return &empty.Empty{}, hferrors.GrpcNotSpecifiedError(req, param)
		}
	}

	if scenariosRaw == "" && coursesRaw == "" {
		return &empty.Empty{}, hferrors.GrpcError(codes.InvalidArgument, "no courses or scenarios provided", req)
	}

	onDemand, err := strconv.ParseBool(onDemandRaw)
	if err != nil {
		return &empty.Empty{}, hferrors.GrpcBadRequestError(req, "on_demand_raw", onDemandRaw)
	}
	printable, err := strconv.ParseBool(printableRaw)
	if err != nil {
		return &empty.Empty{}, hferrors.GrpcBadRequestError(req, "printable_raw", printableRaw)
	}

	requiredVms, err := util.GenericUnmarshal[map[string]map[string]int](reqVmsRaw, "required_vms_raw")
	if err != nil {
		return &empty.Empty{}, hferrors.GrpcParsingError(req, "required_vms_raw")
	}

	random := util.RandStringRunes(16)
	id := util.GenerateResourceName("se", random, 10)

	event := &hfv1.ScheduledEvent{
		ObjectMeta: metav1.ObjectMeta{
			Name:   id,
			Labels: labels,
		},
		Spec: hfv1.ScheduledEventSpec{
			Creator:                 creator,
			Name:                    name,
			Description:             description,
			StartTime:               startTime,
			EndTime:                 endTime,
			OnDemand:                onDemand,
			RequiredVirtualMachines: requiredVms,
			AccessCode:              accessCode,
			RestrictedBind:          restrictedBind,
			Printable:               printable,
		},
	}

	if restrictedBind {
		event.Spec.RestrictedBindValue = event.Name
	}

	if coursesRaw != "" {
		courses, err := util.GenericUnmarshal[[]string](coursesRaw, "courses_raw")
		if err != nil {
			return &empty.Empty{}, hferrors.GrpcParsingError(req, "courses_raw")
		}
		event.Spec.Courses = courses
	}
	if scenariosRaw != "" {
		scenarios, err := util.GenericUnmarshal[[]string](scenariosRaw, "scenarios_raw")
		if err != nil {
			return &empty.Empty{}, hferrors.GrpcParsingError(req, "scenarios_raw")
		}
		event.Spec.Scenarios = scenarios
	}

	_, err = s.eventClient.Create(ctx, event, metav1.CreateOptions{})
	if err != nil {
		return &empty.Empty{}, hferrors.GrpcError(
			codes.Internal,
			err.Error(),
			req,
		)
	}
	return &empty.Empty{}, nil
}

func (s *GrpcScheduledEventServer) GetScheduledEvent(ctx context.Context, req *general.GetRequest) (*eventProto.ScheduledEvent, error) {
	event, err := util.GenericHfGetter(ctx, req, s.eventClient, s.eventLister.ScheduledEvents(util.GetReleaseNamespace()), "scheduled event", s.eventSynced())
	if err != nil {
		return &eventProto.ScheduledEvent{}, err
	}

	status := &eventProto.ScheduledEventStatus{
		Vmsets:      event.Status.VirtualMachineSets,
		Active:      event.Status.Active,
		Provisioned: event.Status.Provisioned,
		Ready:       event.Status.Ready,
		Finished:    event.Status.Finished,
	}

	requiredVms := make(map[string]*eventProto.VMTemplateCountMap)
	for environment, vmTemplateCountMap := range event.Spec.RequiredVirtualMachines {
		requiredVms[environment] = &eventProto.VMTemplateCountMap{VmTemplateCounts: util.ConvertMap[int, uint32](vmTemplateCountMap)}
	}

	return &eventProto.ScheduledEvent{
		Id:                  event.Name,
		Uid:                 string(event.UID),
		Name:                event.Spec.Name,
		Description:         event.Spec.Description,
		Creator:             event.Spec.Creator,
		StartTime:           event.Spec.StartTime,
		EndTime:             event.Spec.EndTime,
		OnDemand:            event.Spec.OnDemand,
		Printable:           event.Spec.Printable,
		RestrictedBind:      event.Spec.RestrictedBind,
		RestrictedBindValue: event.Spec.RestrictedBindValue,
		RequiredVms:         requiredVms,
		AccessCode:          event.Spec.AccessCode,
		Scenarios:           event.Spec.Scenarios,
		Courses:             event.Spec.Courses,
		Labels:              event.Labels,
		Status:              status,
	}, nil
}

func (s *GrpcScheduledEventServer) UpdateScheduledEvent(ctx context.Context, req *eventProto.UpdateScheduledEventRequest) (*empty.Empty, error) {
	id := req.GetId()
	if len(id) == 0 {
		return &empty.Empty{}, hferrors.GrpcIdNotSpecifiedError(req)
	}
	name := req.GetName()
	description := req.GetDescription()
	startTime := req.GetStartTime()
	endTime := req.GetEndTime()
	onDemandRaw := req.GetOnDemandRaw()
	printableRaw := req.GetPrintableRaw()
	restrictedBind := req.GetRestrictedBind()
	reqVmsRaw := req.GetRequiredVmsRaw()
	accessCode := req.GetAccessCode()
	scenariosRaw := req.GetScenariosRaw()
	coursesRaw := req.GetCoursesRaw()

	retryErr := retry.RetryOnConflict(retry.DefaultRetry, func() error {
		event, err := s.eventClient.Get(ctx, id, metav1.GetOptions{})
		if err != nil {
			glog.Error(err)
			return hferrors.GrpcError(
				codes.Internal,
				"error while retrieving scheduled event %s",
				req,
				req.GetId(),
			)
		}
		if name != "" {
			event.Spec.Name = name
		}
		if description != "" {
			event.Spec.Description = description
		}
		if startTime != "" {
			event.Spec.StartTime = startTime
		}
		if endTime != "" {
			event.Spec.EndTime = endTime
		}
		if onDemandRaw != "" {
			onDemand, err := strconv.ParseBool(onDemandRaw)
			if err != nil {
				return hferrors.GrpcBadRequestError(req, "on_demand_raw", onDemandRaw)
			}
			event.Spec.OnDemand = onDemand
		}
		if printableRaw != "" {
			printable, err := strconv.ParseBool(printableRaw)
			if err != nil {
				return hferrors.GrpcBadRequestError(req, "printable_raw", printableRaw)
			}
			event.Spec.Printable = printable
		}
		if restrictedBind != nil {
			event.Spec.RestrictedBind = restrictedBind.Value

			if event.Spec.RestrictedBind {
				event.Spec.RestrictedBindValue = event.Name
			} else {
				event.Spec.RestrictedBindValue = ""
			}
		}
		if reqVmsRaw != "" {
			requiredVms, err := util.GenericUnmarshal[map[string]map[string]int](reqVmsRaw, "required_vms_raw")
			if err != nil {
				return hferrors.GrpcParsingError(req, "required_vms_raw")
			}
			event.Spec.RequiredVirtualMachines = requiredVms
		}
		if accessCode != "" {
			event.Spec.AccessCode = accessCode
		}
		if scenariosRaw != "" {
			scenarios, err := util.GenericUnmarshal[[]string](scenariosRaw, "scenarios_raw")
			if err != nil {
				hferrors.GrpcParsingError(req, "scenarios_raw")
			}
			event.Spec.Scenarios = scenarios
		}
		if coursesRaw != "" {
			courses, err := util.GenericUnmarshal[[]string](coursesRaw, "courses_raw")
			if err != nil {
				hferrors.GrpcParsingError(req, "courses_raw")
			}
			event.Spec.Courses = courses
		}

		_, updateErr := s.eventClient.Update(ctx, event, metav1.UpdateOptions{})
		return updateErr
	})

	if retryErr != nil {
		return &empty.Empty{}, hferrors.GrpcError(
			codes.Internal,
			"error attempting to update",
			req,
		)
	}

	return &empty.Empty{}, nil
}

func (s *GrpcScheduledEventServer) UpdateScheduledEventStatus(ctx context.Context, req *eventProto.UpdateScheduledEventStatusRequest) (*empty.Empty, error) {
	id := req.GetId()
	if len(id) == 0 {
		return &empty.Empty{}, hferrors.GrpcIdNotSpecifiedError(req)
	}

	vmSets := req.GetVmsets()
	active := req.GetActive()
	provisioned := req.GetProvisioned()
	ready := req.GetReady()
	finished := req.GetFinished()

	retryErr := retry.RetryOnConflict(retry.DefaultRetry, func() error {
		event, err := s.eventClient.Get(ctx, id, metav1.GetOptions{})
		if err != nil {
			glog.Error(err)
			return hferrors.GrpcError(
				codes.Internal,
				"error while retrieving scheduled event %s",
				req,
				req.GetId(),
			)
		}

		if vmSets != nil {
			event.Status.VirtualMachineSets = vmSets.GetValue()
		}
		if active != nil {
			event.Status.Active = active.GetValue()
		}
		if provisioned != nil {
			event.Status.Provisioned = provisioned.GetValue()
		}
		if ready != nil {
			event.Status.Ready = ready.GetValue()
		}
		if finished != nil {
			event.Status.Finished = finished.GetValue()
		}

		_, updateErr := s.eventClient.UpdateStatus(ctx, event, metav1.UpdateOptions{})
		if updateErr != nil {
			return updateErr
		}
		// @TODO: verify result like in util.go
		glog.V(4).Infof("updated result for scheduled event")
		return nil
	})
	if retryErr != nil {
		return &empty.Empty{}, hferrors.GrpcError(
			codes.Internal,
			"error attempting to update scheduled event status: %v",
			req,
			retryErr,
		)
	}
	return &empty.Empty{}, nil
}

func (s *GrpcScheduledEventServer) DeleteScheduledEvent(ctx context.Context, req *general.ResourceId) (*empty.Empty, error) {
	return util.DeleteHfResource(ctx, req, s.eventClient, "scheduled event")
}

func (s *GrpcScheduledEventServer) DeleteCollectionScheduledEvent(ctx context.Context, listOptions *general.ListOptions) (*empty.Empty, error) {
	return util.DeleteHfCollection(ctx, listOptions, s.eventClient, "scheduled event")
}

func (s *GrpcScheduledEventServer) ListScheduledEvent(ctx context.Context, listOptions *general.ListOptions) (*eventProto.ListScheduledEventsResponse, error) {
	doLoadFromCache := listOptions.GetLoadFromCache()
	var events []hfv1.ScheduledEvent
	var err error
	if !doLoadFromCache {
		var eventList *hfv1.ScheduledEventList
		eventList, err = util.ListByHfClient(ctx, listOptions, s.eventClient, "scheduled events")
		if err == nil {
			events = eventList.Items
		}
	} else {
		events, err = util.ListByCache(listOptions, s.eventLister, "scheduled events", s.eventSynced())
	}
	if err != nil {
		glog.Error(err)
		return &eventProto.ListScheduledEventsResponse{}, err
	}

	preparedEvents := []*eventProto.ScheduledEvent{}

	for _, event := range events {
		status := &eventProto.ScheduledEventStatus{
			Vmsets:      event.Status.VirtualMachineSets,
			Active:      event.Status.Active,
			Provisioned: event.Status.Provisioned,
			Ready:       event.Status.Ready,
			Finished:    event.Status.Finished,
		}

		requiredVms := make(map[string]*eventProto.VMTemplateCountMap)
		for environment, vmTemplateCountMap := range event.Spec.RequiredVirtualMachines {
			requiredVms[environment] = &eventProto.VMTemplateCountMap{VmTemplateCounts: util.ConvertMap[int, uint32](vmTemplateCountMap)}
		}

		preparedEvents = append(preparedEvents, &eventProto.ScheduledEvent{
			Id:                  event.Name,
			Uid:                 string(event.UID),
			Name:                event.Spec.Name,
			Description:         event.Spec.Description,
			Creator:             event.Spec.Creator,
			StartTime:           event.Spec.StartTime,
			EndTime:             event.Spec.EndTime,
			OnDemand:            event.Spec.OnDemand,
			Printable:           event.Spec.Printable,
			RestrictedBind:      event.Spec.RestrictedBind,
			RestrictedBindValue: event.Spec.RestrictedBindValue,
			RequiredVms:         requiredVms,
			AccessCode:          event.Spec.AccessCode,
			Scenarios:           event.Spec.Scenarios,
			Courses:             event.Spec.Courses,
			Labels:              event.Labels,
			Status:              status,
		})
	}

	return &eventProto.ListScheduledEventsResponse{Scheduledevents: preparedEvents}, nil
}
