package scenarioservice

import (
	"context"

	"github.com/hobbyfarm/gargantua/v3/protos/general"
	scenarioProto "github.com/hobbyfarm/gargantua/v3/protos/scenario"

	"github.com/golang/glog"
	"github.com/golang/protobuf/ptypes/empty"
	hfv1 "github.com/hobbyfarm/gargantua/v3/pkg/apis/hobbyfarm.io/v1"
	hfClientset "github.com/hobbyfarm/gargantua/v3/pkg/client/clientset/versioned"
	hfClientsetv1 "github.com/hobbyfarm/gargantua/v3/pkg/client/clientset/versioned/typed/hobbyfarm.io/v1"
	hfInformers "github.com/hobbyfarm/gargantua/v3/pkg/client/informers/externalversions"
	listersv1 "github.com/hobbyfarm/gargantua/v3/pkg/client/listers/hobbyfarm.io/v1"
	hferrors "github.com/hobbyfarm/gargantua/v3/pkg/errors"
	"github.com/hobbyfarm/gargantua/v3/pkg/labels"
	"github.com/hobbyfarm/gargantua/v3/pkg/util"
	"google.golang.org/grpc/codes"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/retry"
)

type GrpcScenarioServer struct {
	scenarioProto.UnimplementedScenarioSvcServer
	scenarioClient hfClientsetv1.ScenarioInterface
	scenarioLister listersv1.ScenarioLister
	scenarioSynced cache.InformerSynced
}

func NewGrpcScenarioServer(hfClientSet hfClientset.Interface, hfInformerFactory hfInformers.SharedInformerFactory) *GrpcScenarioServer {
	return &GrpcScenarioServer{
		scenarioClient: hfClientSet.HobbyfarmV1().Scenarios(util.GetReleaseNamespace()),
		scenarioLister: hfInformerFactory.Hobbyfarm().V1().Scenarios().Lister(),
		scenarioSynced: hfInformerFactory.Hobbyfarm().V1().Scenarios().Informer().HasSynced,
	}
}

func (s *GrpcScenarioServer) CreateScenario(ctx context.Context, req *scenarioProto.CreateScenarioRequest) (*empty.Empty, error) {
	name := req.GetName()
	description := req.GetDescription()
	rawSteps := req.GetRawSteps()
	rawCategories := req.GetRawCategories()
	rawTags := req.GetRawTags()
	rawVirtualMachines := req.GetRawVms()
	keepaliveDuration := req.GetKeepaliveDuration()
	pauseDuration := req.GetPauseDuration()
	pausable := req.GetPausable()

	requiredStringParams := map[string]string{
		"name":        name,
		"description": description,
	}
	for param, value := range requiredStringParams {
		if value == "" {
			return &empty.Empty{}, hferrors.GrpcNotSpecifiedError(req, param)
		}
	}

	id := util.GenerateResourceName("s", name, 10)

	scenario := &hfv1.Scenario{
		ObjectMeta: metav1.ObjectMeta{
			Name:   id,
			Labels: make(map[string]string),
		},
		Spec: hfv1.ScenarioSpec{
			Name:              name,
			Description:       description,
			KeepAliveDuration: keepaliveDuration,
			PauseDuration:     pauseDuration,
			Pauseable:         pausable,
		},
	}

	if rawSteps != "" {
		steps, err := util.GenericUnmarshal[[]hfv1.ScenarioStep](rawSteps, "rawSteps")
		if err != nil {
			return &empty.Empty{}, hferrors.GrpcParsingError(req, "rawSteps")
		}
		scenario.Spec.Steps = steps
	}
	if rawCategories != "" {
		categories, err := util.GenericUnmarshal[[]string](rawCategories, "rawCategories")
		if err != nil {
			return &empty.Empty{}, hferrors.GrpcParsingError(req, "rawCategories")
		}
		updatedLabels := labels.UpdateCategoryLabels(scenario.ObjectMeta.Labels, []string{}, categories)
		scenario.ObjectMeta.Labels = updatedLabels
		scenario.Spec.Categories = categories
	}
	if rawTags != "" {
		tags, err := util.GenericUnmarshal[[]string](rawTags, "rawTags")
		if err != nil {
			return &empty.Empty{}, hferrors.GrpcParsingError(req, "rawTags")
		}
		scenario.Spec.Tags = tags
	}
	if rawVirtualMachines != "" {
		vms, err := util.GenericUnmarshal[[]map[string]string](rawVirtualMachines, "rawVirtualMachines")
		if err != nil {
			return &empty.Empty{}, hferrors.GrpcParsingError(req, "rawVirtualMachines")
		}
		scenario.Spec.VirtualMachines = vms
	}

	_, err := s.scenarioClient.Create(ctx, scenario, metav1.CreateOptions{})
	if err != nil {
		return &empty.Empty{}, hferrors.GrpcError(
			codes.Internal,
			err.Error(),
			req,
		)
	}
	return &empty.Empty{}, nil
}

func (s *GrpcScenarioServer) GetScenario(ctx context.Context, req *general.GetRequest) (*scenarioProto.Scenario, error) {
	scenario, err := util.GenericHfGetter(ctx, req, s.scenarioClient, s.scenarioLister.Scenarios(util.GetReleaseNamespace()), "scenario", s.scenarioSynced())
	if err != nil {
		return &scenarioProto.Scenario{}, err
	}

	scenarioSteps := []*scenarioProto.ScenarioStep{}
	for _, step := range scenario.Spec.Steps {
		scenarioSteps = append(scenarioSteps, &scenarioProto.ScenarioStep{Title: step.Title, Content: step.Content})
	}

	vms := []*general.StringMap{}
	for _, vm := range scenario.Spec.VirtualMachines {
		vms = append(vms, &general.StringMap{Value: vm})
	}

	return &scenarioProto.Scenario{
		Id:                scenario.Name,
		Name:              scenario.Spec.Name,
		Description:       scenario.Spec.Description,
		Steps:             scenarioSteps,
		Categories:        scenario.Spec.Categories,
		Tags:              scenario.Spec.Tags,
		Vms:               vms,
		KeepaliveDuration: scenario.Spec.KeepAliveDuration,
		PauseDuration:     scenario.Spec.PauseDuration,
		Pausable:          scenario.Spec.Pauseable,
		Labels:            scenario.Labels,
	}, nil
}

func (s *GrpcScenarioServer) UpdateScenario(ctx context.Context, req *scenarioProto.UpdateScenarioRequest) (*empty.Empty, error) {
	id := req.GetId()
	if len(id) == 0 {
		return &empty.Empty{}, hferrors.GrpcIdNotSpecifiedError(req)
	}
	name := req.GetName()
	description := req.GetDescription()
	rawSteps := req.GetRawSteps()
	rawCategories := req.GetRawCategories()
	rawTags := req.GetRawTags()
	rawVirtualMachines := req.GetRawVms()
	keepaliveDuration := req.GetKeepaliveDuration()
	pauseDuration := req.GetPauseDuration()
	pausable := req.GetPausable()

	retryErr := retry.RetryOnConflict(retry.DefaultRetry, func() error {
		scenario, err := s.scenarioClient.Get(ctx, id, metav1.GetOptions{})
		if err != nil {
			glog.Error(err)
			return hferrors.GrpcError(
				codes.Internal,
				"error while retrieving scenario %s",
				req,
				req.GetId(),
			)
		}
		if name != "" {
			scenario.Spec.Name = name
		}
		if description != "" {
			scenario.Spec.Description = description
		}
		if keepaliveDuration != nil {
			scenario.Spec.KeepAliveDuration = keepaliveDuration.GetValue()
		}
		if pauseDuration != nil {
			scenario.Spec.PauseDuration = pauseDuration.GetValue()
		}
		if pausable != nil {
			scenario.Spec.Pauseable = pausable.GetValue()
		}
		if rawSteps != "" {
			steps, err := util.GenericUnmarshal[[]hfv1.ScenarioStep](rawSteps, "rawSteps")
			if err != nil {
				return hferrors.GrpcParsingError(req, "rawSteps")
			}
			scenario.Spec.Steps = steps
		}
		if rawCategories != "" {
			newCategories, err := util.GenericUnmarshal[[]string](rawCategories, "rawCategories")
			if err != nil {
				return hferrors.GrpcParsingError(req, "rawCategories")
			}
			oldCategories := scenario.Spec.Categories
			updatedLabels := labels.UpdateCategoryLabels(scenario.ObjectMeta.Labels, oldCategories, newCategories)
			scenario.Spec.Categories = newCategories
			scenario.ObjectMeta.Labels = updatedLabels
		}
		if rawTags != "" {
			tags, err := util.GenericUnmarshal[[]string](rawTags, "rawTags")
			if err != nil {
				return hferrors.GrpcParsingError(req, "rawTags")
			}
			scenario.Spec.Tags = tags
		}
		if rawVirtualMachines != "" {
			vms, err := util.GenericUnmarshal[[]map[string]string](rawVirtualMachines, "rawVirtualMachines")
			if err != nil {
				return hferrors.GrpcParsingError(req, "rawVirtualMachines")
			}
			scenario.Spec.VirtualMachines = vms
		}

		_, updateErr := s.scenarioClient.Update(ctx, scenario, metav1.UpdateOptions{})
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

func (s *GrpcScenarioServer) DeleteScenario(ctx context.Context, req *general.ResourceId) (*empty.Empty, error) {
	return util.DeleteHfResource(ctx, req, s.scenarioClient, "scenario")
}

func (s *GrpcScenarioServer) DeleteCollectionScenario(ctx context.Context, listOptions *general.ListOptions) (*empty.Empty, error) {
	return util.DeleteHfCollection(ctx, listOptions, s.scenarioClient, "scenarios")
}

func (s *GrpcScenarioServer) ListScenario(ctx context.Context, listOptions *general.ListOptions) (*scenarioProto.ListScenariosResponse, error) {
	doLoadFromCache := listOptions.GetLoadFromCache()
	var scenarios []hfv1.Scenario
	var err error
	if !doLoadFromCache {
		var scenarioList *hfv1.ScenarioList
		scenarioList, err = util.ListByHfClient(ctx, listOptions, s.scenarioClient, "scenarios")
		if err == nil {
			scenarios = scenarioList.Items
		}
	} else {
		scenarios, err = util.ListByCache(listOptions, s.scenarioLister, "scenarios", s.scenarioSynced())
	}
	if err != nil {
		glog.Error(err)
		return &scenarioProto.ListScenariosResponse{}, err
	}

	preparedScenarios := []*scenarioProto.Scenario{}

	for _, scenario := range scenarios {

		scenarioSteps := []*scenarioProto.ScenarioStep{}
		for _, step := range scenario.Spec.Steps {
			scenarioSteps = append(scenarioSteps, &scenarioProto.ScenarioStep{Title: step.Title, Content: step.Content})
		}

		vms := []*general.StringMap{}
		for _, vm := range scenario.Spec.VirtualMachines {
			vms = append(vms, &general.StringMap{Value: vm})
		}

		preparedScenarios = append(preparedScenarios, &scenarioProto.Scenario{
			Id:                scenario.Name,
			Name:              scenario.Spec.Name,
			Description:       scenario.Spec.Description,
			Steps:             scenarioSteps,
			Categories:        scenario.Spec.Categories,
			Tags:              scenario.Spec.Tags,
			Vms:               vms,
			KeepaliveDuration: scenario.Spec.KeepAliveDuration,
			PauseDuration:     scenario.Spec.PauseDuration,
			Pausable:          scenario.Spec.Pauseable,
			Labels:            scenario.Labels,
		})
	}

	return &scenarioProto.ListScenariosResponse{Scenarios: preparedScenarios}, nil
}
