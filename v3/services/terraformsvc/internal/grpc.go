package terraformsvc

import (
	"context"
	"fmt"
	"math/rand"
	"strings"

	"github.com/hobbyfarm/gargantua/v3/protos/general"
	terraformpb "github.com/hobbyfarm/gargantua/v3/protos/terraform"

	"github.com/golang/glog"
	"github.com/golang/protobuf/ptypes/empty"
	tfv1 "github.com/hobbyfarm/gargantua/v3/pkg/apis/terraformcontroller.cattle.io/v1"
	hfClientset "github.com/hobbyfarm/gargantua/v3/pkg/client/clientset/versioned"
	tfClientsetv1 "github.com/hobbyfarm/gargantua/v3/pkg/client/clientset/versioned/typed/terraformcontroller.cattle.io/v1"
	hfInformers "github.com/hobbyfarm/gargantua/v3/pkg/client/informers/externalversions"
	listersv1 "github.com/hobbyfarm/gargantua/v3/pkg/client/listers/terraformcontroller.cattle.io/v1"
	hferrors "github.com/hobbyfarm/gargantua/v3/pkg/errors"
	"github.com/hobbyfarm/gargantua/v3/pkg/util"
	"google.golang.org/grpc/codes"
	"google.golang.org/protobuf/types/known/timestamppb"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/cache"
)

type GrpcTerraformServer struct {
	terraformpb.UnimplementedTerraformSvcServer
	stateClient     tfClientsetv1.StateInterface
	stateLister     listersv1.StateLister
	stateSynced     cache.InformerSynced
	executionClient tfClientsetv1.ExecutionInterface
	executionLister listersv1.ExecutionLister
	executionSynced cache.InformerSynced
}

func NewGrpcTerraformServer(hfClientSet hfClientset.Interface, hfInformerFactory hfInformers.SharedInformerFactory) *GrpcTerraformServer {
	return &GrpcTerraformServer{
		stateClient:     hfClientSet.TerraformcontrollerV1().States(util.GetReleaseNamespace()),
		stateLister:     hfInformerFactory.Terraformcontroller().V1().States().Lister(),
		stateSynced:     hfInformerFactory.Terraformcontroller().V1().States().Informer().HasSynced,
		executionClient: hfClientSet.TerraformcontrollerV1().Executions(util.GetReleaseNamespace()),
		executionLister: hfInformerFactory.Terraformcontroller().V1().Executions().Lister(),
		executionSynced: hfInformerFactory.Terraformcontroller().V1().Executions().Informer().HasSynced,
	}
}

func (s *GrpcTerraformServer) CreateState(ctx context.Context, req *terraformpb.CreateStateRequest) (*empty.Empty, error) {
	vmId := req.GetVmId()
	img := req.GetImage()
	variables := req.GetVariables()
	moduleName := req.GetModuleName()
	data := req.GetData()
	autoConfirm := req.GetAutoConfirm()
	destroyOnDelete := req.GetDestroyOnDelete()
	version := req.GetVersion()

	requiredStringParams := map[string]string{
		"vmId":       vmId,
		"image":      img,
		"moduleName": moduleName,
	}
	for param, value := range requiredStringParams {
		if value == "" {
			return &empty.Empty{}, hferrors.GrpcNotSpecifiedError(req, param)
		}
	}
	if variables == nil || len(variables.GetConfigNames()) == 0 {
		return &empty.Empty{}, hferrors.GrpcError(
			codes.InvalidArgument,
			"invalid value \"%v\" for property %s",
			req,
			req.GetVariables(),
			"variables",
		)
	}

	random := fmt.Sprintf("%08x", rand.Uint32())
	id := strings.Join([]string{vmId + "-tfs", random}, "-")

	tfs := &tfv1.State{
		ObjectMeta: metav1.ObjectMeta{
			Name: id,
		},
		Spec: tfv1.StateSpec{
			Variables: tfv1.Variables{
				EnvConfigName:  variables.GetEnvConfigNames(),
				EnvSecretNames: variables.GetEnvSecretNames(),
				ConfigNames:    variables.GetConfigNames(),
				SecretNames:    variables.GetSecretNames(),
			},
			Image:           img,
			AutoConfirm:     autoConfirm,
			DestroyOnDelete: destroyOnDelete,
			ModuleName:      moduleName,
			Data:            data,
			Version:         version,
		},
	}

	_, err := s.stateClient.Create(ctx, tfs, metav1.CreateOptions{})
	if err != nil {
		return &empty.Empty{}, hferrors.GrpcError(
			codes.Internal,
			err.Error(),
			req,
		)
	}
	return &empty.Empty{}, nil
}

func (s *GrpcTerraformServer) GetState(ctx context.Context, req *general.GetRequest) (*terraformpb.State, error) {
	state, err := util.GenericHfGetter(ctx, req, s.stateClient, s.stateLister.States(util.GetReleaseNamespace()), "state", s.stateSynced())
	if err != nil {
		return &terraformpb.State{}, err
	}

	tfConditions := []*terraformpb.Condition{}

	for _, condition := range state.Status.Conditions {
		tfCondition := &terraformpb.Condition{
			Type:               condition.Type,
			LastUpdateTime:     condition.LastUpdateTime,
			LastTransitionTime: condition.LastTransitionTime,
			Reason:             condition.Reason,
			Message:            condition.Message,
		}
		tfConditions = append(tfConditions, tfCondition)
	}

	status := &terraformpb.StateStatus{
		Conditions:        tfConditions,
		LastRunHash:       state.Status.LastRunHash,
		ExecutionName:     state.Status.ExecutionName,
		ExecutionPlanName: state.Status.ExecutionName,
	}

	var creationTimeStamp *timestamppb.Timestamp
	if !state.CreationTimestamp.IsZero() {
		creationTimeStamp = timestamppb.New(state.CreationTimestamp.Time)
	}

	return &terraformpb.State{
		Id:    state.Name,
		Image: state.Spec.Image,
		Variables: &terraformpb.Variables{
			EnvConfigNames: state.Spec.Variables.EnvConfigName,
			EnvSecretNames: state.Spec.Variables.EnvSecretNames,
			ConfigNames:    state.Spec.Variables.ConfigNames,
			SecretNames:    state.Spec.Variables.SecretNames,
		},
		ModuleName:        state.Spec.ModuleName,
		Data:              state.Spec.Data,
		AutoConfirm:       state.Spec.AutoConfirm,
		DestroyOnDelete:   state.Spec.DestroyOnDelete,
		Version:           state.Spec.Version,
		Status:            status,
		CreationTimestamp: creationTimeStamp,
	}, nil
}

func (s *GrpcTerraformServer) DeleteState(ctx context.Context, req *general.ResourceId) (*empty.Empty, error) {
	return util.DeleteHfResource(ctx, req, s.stateClient, "state")
}

func (s *GrpcTerraformServer) DeleteCollectionState(ctx context.Context, listOptions *general.ListOptions) (*empty.Empty, error) {
	return util.DeleteHfCollection(ctx, listOptions, s.stateClient, "state")
}

func (s *GrpcTerraformServer) ListState(ctx context.Context, listOptions *general.ListOptions) (*terraformpb.ListStateResponse, error) {
	doLoadFromCache := listOptions.GetLoadFromCache()
	var states []tfv1.State
	var err error
	if !doLoadFromCache {
		var stateList *tfv1.StateList
		stateList, err = util.ListByHfClient(ctx, listOptions, s.stateClient, "states")
		if err == nil {
			states = stateList.Items
		}
	} else {
		states, err = util.ListByCache(listOptions, s.stateLister, "states", s.stateSynced())
	}
	if err != nil {
		glog.Error(err)
		return &terraformpb.ListStateResponse{}, err
	}

	preparedStates := []*terraformpb.State{}

	for _, state := range states {
		tfConditions := []*terraformpb.Condition{}

		for _, condition := range state.Status.Conditions {
			tfCondition := &terraformpb.Condition{
				Type:               condition.Type,
				LastUpdateTime:     condition.LastUpdateTime,
				LastTransitionTime: condition.LastTransitionTime,
				Reason:             condition.Reason,
				Message:            condition.Message,
			}
			tfConditions = append(tfConditions, tfCondition)
		}

		status := &terraformpb.StateStatus{
			Conditions:        tfConditions,
			LastRunHash:       state.Status.LastRunHash,
			ExecutionName:     state.Status.ExecutionName,
			ExecutionPlanName: state.Status.ExecutionName,
		}

		var creationTimeStamp *timestamppb.Timestamp
		if !state.CreationTimestamp.IsZero() {
			creationTimeStamp = timestamppb.New(state.CreationTimestamp.Time)
		}

		preparedStates = append(preparedStates, &terraformpb.State{
			Id:    state.Name,
			Image: state.Spec.Image,
			Variables: &terraformpb.Variables{
				EnvConfigNames: state.Spec.Variables.EnvConfigName,
				EnvSecretNames: state.Spec.Variables.EnvSecretNames,
				ConfigNames:    state.Spec.Variables.ConfigNames,
				SecretNames:    state.Spec.Variables.SecretNames,
			},
			ModuleName:        state.Spec.ModuleName,
			Data:              state.Spec.Data,
			AutoConfirm:       state.Spec.AutoConfirm,
			DestroyOnDelete:   state.Spec.DestroyOnDelete,
			Version:           state.Spec.Version,
			Status:            status,
			CreationTimestamp: creationTimeStamp,
		})
	}

	return &terraformpb.ListStateResponse{States: preparedStates}, nil
}

func (s *GrpcTerraformServer) GetExecution(ctx context.Context, req *general.GetRequest) (*terraformpb.Execution, error) {
	execution, err := util.GenericHfGetter(ctx, req, s.executionClient, s.executionLister.Executions(util.GetReleaseNamespace()), "execution", s.executionSynced())
	if err != nil {
		return &terraformpb.Execution{}, err
	}

	tfConditions := []*terraformpb.Condition{}

	for _, condition := range execution.Status.Conditions {
		tfCondition := &terraformpb.Condition{
			Type:               condition.Type,
			LastUpdateTime:     condition.LastUpdateTime,
			LastTransitionTime: condition.LastTransitionTime,
			Reason:             condition.Reason,
			Message:            condition.Message,
		}
		tfConditions = append(tfConditions, tfCondition)
	}

	status := &terraformpb.ExecutionStatus{
		Conditions:    tfConditions,
		JobName:       execution.Status.JobName,
		JobLogs:       execution.Status.JobLogs,
		PlanOutput:    execution.Status.PlanOutput,
		PlanConfirmed: execution.Status.PlanConfirmed,
		ApplyOutput:   execution.Status.ApplyOutput,
		Outputs:       execution.Status.Outputs,
	}

	content := &terraformpb.ModuleContent{
		Content: execution.Spec.Content.Content,
		Git: &terraformpb.GitLocation{
			Url:             execution.Spec.Content.Git.URL,
			Branch:          execution.Spec.Content.Git.Branch,
			Tag:             execution.Spec.Content.Git.Tag,
			Commit:          execution.Spec.Content.Git.Commit,
			SecretName:      execution.Spec.Content.Git.SecretName,
			IntervalSeconds: int64(execution.Spec.Content.Git.IntervalSeconds),
		},
	}

	var creationTimeStamp *timestamppb.Timestamp
	if !execution.CreationTimestamp.IsZero() {
		creationTimeStamp = timestamppb.New(execution.CreationTimestamp.Time)
	}

	return &terraformpb.Execution{
		Id:                execution.Name,
		AutoConfirm:       execution.Spec.AutoConfirm,
		Content:           content,
		ContentHash:       execution.Spec.ContentHash,
		RunHash:           execution.Spec.RunHash,
		Data:              execution.Spec.Data,
		ExecutionName:     execution.Spec.ExecutionName,
		ExecutionVersion:  execution.Spec.ExecutionVersion,
		SecretName:        execution.Spec.SecretName,
		Status:            status,
		CreationTimestamp: creationTimeStamp,
	}, nil
}

func (s *GrpcTerraformServer) ListExecution(ctx context.Context, listOptions *general.ListOptions) (*terraformpb.ListExecutionResponse, error) {
	doLoadFromCache := listOptions.GetLoadFromCache()
	var executions []tfv1.Execution
	var err error
	if !doLoadFromCache {
		var executionList *tfv1.ExecutionList
		executionList, err = util.ListByHfClient(ctx, listOptions, s.executionClient, "executions")
		if err == nil {
			executions = executionList.Items
		}
	} else {
		executions, err = util.ListByCache(listOptions, s.executionLister, "executions", s.executionSynced())
	}
	if err != nil {
		glog.Error(err)
		return &terraformpb.ListExecutionResponse{}, err
	}

	preparedExecutions := []*terraformpb.Execution{}

	for _, execution := range executions {
		tfConditions := []*terraformpb.Condition{}

		for _, condition := range execution.Status.Conditions {
			tfCondition := &terraformpb.Condition{
				Type:               condition.Type,
				LastUpdateTime:     condition.LastUpdateTime,
				LastTransitionTime: condition.LastTransitionTime,
				Reason:             condition.Reason,
				Message:            condition.Message,
			}
			tfConditions = append(tfConditions, tfCondition)
		}

		status := &terraformpb.ExecutionStatus{
			Conditions:    tfConditions,
			JobName:       execution.Status.JobName,
			JobLogs:       execution.Status.JobLogs,
			PlanOutput:    execution.Status.PlanOutput,
			PlanConfirmed: execution.Status.PlanConfirmed,
			ApplyOutput:   execution.Status.ApplyOutput,
			Outputs:       execution.Status.Outputs,
		}

		content := &terraformpb.ModuleContent{
			Content: execution.Spec.Content.Content,
			Git: &terraformpb.GitLocation{
				Url:             execution.Spec.Content.Git.URL,
				Branch:          execution.Spec.Content.Git.Branch,
				Tag:             execution.Spec.Content.Git.Tag,
				Commit:          execution.Spec.Content.Git.Commit,
				SecretName:      execution.Spec.Content.Git.SecretName,
				IntervalSeconds: int64(execution.Spec.Content.Git.IntervalSeconds),
			},
		}

		var creationTimeStamp *timestamppb.Timestamp
		if !execution.CreationTimestamp.IsZero() {
			creationTimeStamp = timestamppb.New(execution.CreationTimestamp.Time)
		}

		preparedExecutions = append(preparedExecutions, &terraformpb.Execution{
			Id:                execution.Name,
			AutoConfirm:       execution.Spec.AutoConfirm,
			Content:           content,
			ContentHash:       execution.Spec.ContentHash,
			RunHash:           execution.Spec.RunHash,
			Data:              execution.Spec.Data,
			ExecutionName:     execution.Spec.ExecutionName,
			ExecutionVersion:  execution.Spec.ExecutionVersion,
			SecretName:        execution.Spec.SecretName,
			Status:            status,
			CreationTimestamp: creationTimeStamp,
		})
	}

	return &terraformpb.ListExecutionResponse{Executions: preparedExecutions}, nil
}
