package scenarioservice

import (
	"context"

	generalpb "github.com/hobbyfarm/gargantua/v3/protos/general"
	scenariopb "github.com/hobbyfarm/gargantua/v3/protos/scenario"

	"github.com/golang/glog"
	hfv1 "github.com/hobbyfarm/gargantua/v3/pkg/apis/hobbyfarm.io/v1"
	hfClientset "github.com/hobbyfarm/gargantua/v3/pkg/client/clientset/versioned"
	hfClientsetv1 "github.com/hobbyfarm/gargantua/v3/pkg/client/clientset/versioned/typed/hobbyfarm.io/v1"
	hfInformers "github.com/hobbyfarm/gargantua/v3/pkg/client/informers/externalversions"
	listersv1 "github.com/hobbyfarm/gargantua/v3/pkg/client/listers/hobbyfarm.io/v1"
	hferrors "github.com/hobbyfarm/gargantua/v3/pkg/errors"
	"github.com/hobbyfarm/gargantua/v3/pkg/labels"
	"github.com/hobbyfarm/gargantua/v3/pkg/util"
	"google.golang.org/grpc/codes"
	"google.golang.org/protobuf/types/known/emptypb"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/retry"
)

type GrpcScenarioServer struct {
	scenariopb.UnimplementedScenarioSvcServer
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

func (s *GrpcScenarioServer) CreateScenario(ctx context.Context, req *scenariopb.CreateScenarioRequest) (*emptypb.Empty, error) {
	name := req.GetName()
	description := req.GetDescription()
	rawSteps := req.GetRawSteps()
	rawCategories := req.GetRawCategories()
	rawTags := req.GetRawTags()
	rawVirtualMachines := req.GetRawVms()
	rawVmTasks := req.GetRawVmTasks()
	keepaliveDuration := req.GetKeepaliveDuration()
	pauseDuration := req.GetPauseDuration()
	pausable := req.GetPausable()

	requiredStringParams := map[string]string{
		"name":        name,
		"description": description,
	}
	for param, value := range requiredStringParams {
		if value == "" {
			return &emptypb.Empty{}, hferrors.GrpcNotSpecifiedError(req, param)
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
		steps, err := util.GenericUnmarshal[[]hfv1.ScenarioStep](rawSteps, "raw_steps")
		if err != nil {
			return &emptypb.Empty{}, hferrors.GrpcParsingError(req, "raw_steps")
		}
		scenario.Spec.Steps = steps
	}
	if rawCategories != "" {
		categories, err := util.GenericUnmarshal[[]string](rawCategories, "raw_categories")
		if err != nil {
			return &emptypb.Empty{}, hferrors.GrpcParsingError(req, "raw_categories")
		}
		updatedLabels := labels.UpdateCategoryLabels(scenario.ObjectMeta.Labels, []string{}, categories)
		scenario.ObjectMeta.Labels = updatedLabels
		scenario.Spec.Categories = categories
	}
	if rawTags != "" {
		tags, err := util.GenericUnmarshal[[]string](rawTags, "raw_tags")
		if err != nil {
			return &emptypb.Empty{}, hferrors.GrpcParsingError(req, "raw_tags")
		}
		scenario.Spec.Tags = tags
	}
	if rawVirtualMachines != "" {
		vms, err := util.GenericUnmarshal[[]map[string]string](rawVirtualMachines, "raw_vms")
		if err != nil {
			return &emptypb.Empty{}, hferrors.GrpcParsingError(req, "raw_vms")
		}
		scenario.Spec.VirtualMachines = vms
	}
	if rawVmTasks != "" {
		vmTasks, err := util.GenericUnmarshal[[]hfv1.VirtualMachineTasks](rawSteps, "raw_vm_tasks")
		if err != nil {
			return &emptypb.Empty{}, hferrors.GrpcParsingError(req, "raw_vm_tasks")
		}
		scenario.Spec.Tasks = vmTasks
	}
	err := util.VerifyTaskContent(scenario.Spec.Tasks, req)
	if err != nil {
		return &emptypb.Empty{}, err
	}

	_, err = s.scenarioClient.Create(ctx, scenario, metav1.CreateOptions{})
	if err != nil {
		return &emptypb.Empty{}, hferrors.GrpcError(
			codes.Internal,
			err.Error(),
			req,
		)
	}
	return &emptypb.Empty{}, nil
}

func (s *GrpcScenarioServer) GetScenario(ctx context.Context, req *generalpb.GetRequest) (*scenariopb.Scenario, error) {
	scenario, err := util.GenericHfGetter(ctx, req, s.scenarioClient, s.scenarioLister.Scenarios(util.GetReleaseNamespace()), "scenario", s.scenarioSynced())
	if err != nil {
		return &scenariopb.Scenario{}, err
	}

	scenarioSteps := []*scenariopb.ScenarioStep{}
	for _, step := range scenario.Spec.Steps {
		scenarioSteps = append(scenarioSteps, &scenariopb.ScenarioStep{Title: step.Title, Content: step.Content})
	}

	vms := []*generalpb.StringMap{}
	for _, vm := range scenario.Spec.VirtualMachines {
		vms = append(vms, &generalpb.StringMap{Value: vm})
	}

	vmTasks := []*scenariopb.VirtualMachineTasks{}
	for _, vmtask := range scenario.Spec.Tasks {
		tasks := []*scenariopb.Task{}
		for _, task := range vmtask.Tasks {
			tasks = append(tasks, &scenariopb.Task{
				Name:                task.Name,
				Description:         task.Description,
				Command:             task.Command,
				ExpectedOutputValue: task.ExpectedOutputValue,
				ExpectedReturnCode:  int32(task.ExpectedReturnCode),
				ReturnType:          task.ReturnType,
			})
		}
		vmTasks = append(vmTasks, &scenariopb.VirtualMachineTasks{
			VmId:  vmtask.VMName,
			Tasks: tasks,
		})
	}

	return &scenariopb.Scenario{
		Id:                scenario.Name,
		Uid:               string(scenario.UID),
		Name:              scenario.Spec.Name,
		Description:       scenario.Spec.Description,
		Steps:             scenarioSteps,
		Categories:        scenario.Spec.Categories,
		Tags:              scenario.Spec.Tags,
		Vms:               vms,
		KeepaliveDuration: scenario.Spec.KeepAliveDuration,
		PauseDuration:     scenario.Spec.PauseDuration,
		Pausable:          scenario.Spec.Pauseable,
		VmTasks:           vmTasks,
		Labels:            scenario.Labels,
	}, nil
}

func (s *GrpcScenarioServer) UpdateScenario(ctx context.Context, req *scenariopb.UpdateScenarioRequest) (*emptypb.Empty, error) {
	id := req.GetId()
	if len(id) == 0 {
		return &emptypb.Empty{}, hferrors.GrpcIdNotSpecifiedError(req)
	}
	name := req.GetName()
	description := req.GetDescription()
	rawSteps := req.GetRawSteps()
	rawCategories := req.GetRawCategories()
	rawTags := req.GetRawTags()
	rawVirtualMachines := req.GetRawVms()
	rawVmTasks := req.GetRawVmTasks()
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
			steps, err := util.GenericUnmarshal[[]hfv1.ScenarioStep](rawSteps, "raw_steps")
			if err != nil {
				return hferrors.GrpcParsingError(req, "raw_steps")
			}
			scenario.Spec.Steps = steps
		}
		if rawCategories != "" {
			newCategories, err := util.GenericUnmarshal[[]string](rawCategories, "raw_categories")
			if err != nil {
				return hferrors.GrpcParsingError(req, "raw_categories")
			}
			oldCategories := scenario.Spec.Categories
			updatedLabels := labels.UpdateCategoryLabels(scenario.ObjectMeta.Labels, oldCategories, newCategories)
			scenario.Spec.Categories = newCategories
			scenario.ObjectMeta.Labels = updatedLabels
		}
		if rawTags != "" {
			tags, err := util.GenericUnmarshal[[]string](rawTags, "raw_tags")
			if err != nil {
				return hferrors.GrpcParsingError(req, "raw_tags")
			}
			scenario.Spec.Tags = tags
		}
		if rawVirtualMachines != "" {
			vms, err := util.GenericUnmarshal[[]map[string]string](rawVirtualMachines, "raw_vms")
			if err != nil {
				return hferrors.GrpcParsingError(req, "raw_vms")
			}
			scenario.Spec.VirtualMachines = vms
		}
		if rawVmTasks != "" {
			vmTasks, err := util.GenericUnmarshal[[]hfv1.VirtualMachineTasks](rawSteps, "raw_vm_tasks")
			if err != nil {
				return hferrors.GrpcParsingError(req, "raw_vm_tasks")
			}
			scenario.Spec.Tasks = vmTasks
		}
		err = util.VerifyTaskContent(scenario.Spec.Tasks, req)
		if err != nil {
			return err
		}

		_, updateErr := s.scenarioClient.Update(ctx, scenario, metav1.UpdateOptions{})
		return updateErr
	})

	if retryErr != nil {
		return &emptypb.Empty{}, hferrors.GrpcError(
			codes.Internal,
			"error attempting to update",
			req,
		)
	}

	return &emptypb.Empty{}, nil
}

func (s *GrpcScenarioServer) DeleteScenario(ctx context.Context, req *generalpb.ResourceId) (*emptypb.Empty, error) {
	return util.DeleteHfResource(ctx, req, s.scenarioClient, "scenario")
}

func (s *GrpcScenarioServer) DeleteCollectionScenario(ctx context.Context, listOptions *generalpb.ListOptions) (*emptypb.Empty, error) {
	return util.DeleteHfCollection(ctx, listOptions, s.scenarioClient, "scenarios")
}

func (s *GrpcScenarioServer) ListScenario(ctx context.Context, listOptions *generalpb.ListOptions) (*scenariopb.ListScenariosResponse, error) {
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
		return &scenariopb.ListScenariosResponse{}, err
	}

	preparedScenarios := []*scenariopb.Scenario{}

	for _, scenario := range scenarios {

		scenarioSteps := []*scenariopb.ScenarioStep{}
		for _, step := range scenario.Spec.Steps {
			scenarioSteps = append(scenarioSteps, &scenariopb.ScenarioStep{Title: step.Title, Content: step.Content})
		}

		vms := []*generalpb.StringMap{}
		for _, vm := range scenario.Spec.VirtualMachines {
			vms = append(vms, &generalpb.StringMap{Value: vm})
		}

		vmTasks := []*scenariopb.VirtualMachineTasks{}
		for _, vmtask := range scenario.Spec.Tasks {
			tasks := []*scenariopb.Task{}
			for _, task := range vmtask.Tasks {
				tasks = append(tasks, &scenariopb.Task{
					Name:                task.Name,
					Description:         task.Description,
					Command:             task.Command,
					ExpectedOutputValue: task.ExpectedOutputValue,
					ExpectedReturnCode:  int32(task.ExpectedReturnCode),
					ReturnType:          task.ReturnType,
				})
			}
			vmTasks = append(vmTasks, &scenariopb.VirtualMachineTasks{
				VmId:  vmtask.VMName,
				Tasks: tasks,
			})
		}

		preparedScenarios = append(preparedScenarios, &scenariopb.Scenario{
			Id:                scenario.Name,
			Uid:               string(scenario.UID),
			Name:              scenario.Spec.Name,
			Description:       scenario.Spec.Description,
			Steps:             scenarioSteps,
			Categories:        scenario.Spec.Categories,
			Tags:              scenario.Spec.Tags,
			Vms:               vms,
			KeepaliveDuration: scenario.Spec.KeepAliveDuration,
			PauseDuration:     scenario.Spec.PauseDuration,
			Pausable:          scenario.Spec.Pauseable,
			VmTasks:           vmTasks,
			Labels:            scenario.Labels,
		})
	}

	return &scenariopb.ListScenariosResponse{Scenarios: preparedScenarios}, nil
}
