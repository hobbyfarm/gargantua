package vmservice

import (
	"context"
	"fmt"
	"strings"

	environmentpb "github.com/hobbyfarm/gargantua/v3/protos/environment"
	generalpb "github.com/hobbyfarm/gargantua/v3/protos/general"
	vmpb "github.com/hobbyfarm/gargantua/v3/protos/vm"
	vmtemplatepb "github.com/hobbyfarm/gargantua/v3/protos/vmtemplate"

	"github.com/golang/glog"
	hfv1 "github.com/hobbyfarm/gargantua/v3/pkg/apis/hobbyfarm.io/v1"
	hfClientset "github.com/hobbyfarm/gargantua/v3/pkg/client/clientset/versioned"
	hfClientsetv1 "github.com/hobbyfarm/gargantua/v3/pkg/client/clientset/versioned/typed/hobbyfarm.io/v1"
	hfInformers "github.com/hobbyfarm/gargantua/v3/pkg/client/informers/externalversions"
	listersv1 "github.com/hobbyfarm/gargantua/v3/pkg/client/listers/hobbyfarm.io/v1"
	hferrors "github.com/hobbyfarm/gargantua/v3/pkg/errors"
	"github.com/hobbyfarm/gargantua/v3/pkg/util"
	"google.golang.org/grpc/codes"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	v1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/retry"
)

type GrpcVMServer struct {
	vmpb.UnimplementedVMSvcServer
	secretClient      v1.SecretInterface
	configMapClient   v1.ConfigMapInterface
	vmClient          hfClientsetv1.VirtualMachineInterface
	vmLister          listersv1.VirtualMachineLister
	vmSynced          cache.InformerSynced
	environmentClient environmentpb.EnvironmentSvcClient
	vmTemplateClient  vmtemplatepb.VMTemplateSvcClient
}

func NewGrpcVMServer(hfClientSet hfClientset.Interface, hfInformerFactory hfInformers.SharedInformerFactory, kubeClient *kubernetes.Clientset, environmentClient environmentpb.EnvironmentSvcClient, vmTemplateClient vmtemplatepb.VMTemplateSvcClient) *GrpcVMServer {
	return &GrpcVMServer{
		secretClient:      kubeClient.CoreV1().Secrets(util.GetReleaseNamespace()),
		configMapClient:   kubeClient.CoreV1().ConfigMaps(util.GetReleaseNamespace()),
		vmClient:          hfClientSet.HobbyfarmV1().VirtualMachines(util.GetReleaseNamespace()),
		vmLister:          hfInformerFactory.Hobbyfarm().V1().VirtualMachines().Lister(),
		vmSynced:          hfInformerFactory.Hobbyfarm().V1().VirtualMachines().Informer().HasSynced,
		environmentClient: environmentClient,
		vmTemplateClient:  vmTemplateClient,
	}
}

func (s *GrpcVMServer) CreateVM(ctx context.Context, req *vmpb.CreateVMRequest) (*emptypb.Empty, error) {
	var ownerReferenceId string
	var ownerReferenceUid types.UID
	var ownerReferenceKind string

	id := req.GetId()
	environmentId := req.GetEnvironmentId()
	vmTemplateId := req.GetVmTemplateId()
	sshUserName := req.GetSshUsername()
	protocol := req.GetProtocol()
	secretName := req.GetSecretName()
	vmClaimId := req.GetVmClaimId()
	vmClaimUid := req.GetVmClaimUid()
	user := req.GetUser()
	provision := req.GetProvision()
	vmSetId := req.GetVmSetId()
	vmSetUid := req.GetVmSetUid()
	labels := req.GetLabels()
	finalizers := req.GetFinalizers()

	vmSetOwner := vmSetId != "" && vmSetUid != ""
	vmClaimOwner := vmClaimId != "" && vmClaimUid != ""
	// either vmClaimId AND vmClaimUid or vmSetId AND vmSetUid need to be provided for the owner reference
	// if that's not the case, return an error
	if !vmSetOwner && !vmClaimOwner {
		return &emptypb.Empty{}, hferrors.GrpcError(codes.InvalidArgument, "no ID and UID for owner reference provided", req)
	}

	// vm set takes precedence over vm claim
	if vmSetOwner {
		ownerReferenceId = vmSetId
		ownerReferenceUid = types.UID(vmSetUid)
		ownerReferenceKind = "VirtualMachineSet"
	} else {
		ownerReferenceId = vmClaimId
		ownerReferenceUid = types.UID(vmClaimUid)
		ownerReferenceKind = "VirtualMachineClaim"
	}

	requiredStringParams := map[string]string{
		"id":           id,
		"vmTemplateId": vmTemplateId,
		"protocol":     protocol,
	}
	for param, value := range requiredStringParams {
		if value == "" {
			return &emptypb.Empty{}, hferrors.GrpcNotSpecifiedError(req, param)
		}
	}

	env, err := s.environmentClient.GetEnvironment(ctx, &generalpb.GetRequest{Id: environmentId, LoadFromCache: true})
	if err != nil {
		glog.Errorf("error getting env %v", err)
		return &emptypb.Empty{}, err
	}

	_, exists := env.GetTemplateMapping()[vmTemplateId]
	if !exists {
		glog.Errorf("error pulling environment template info %v", err)
		return &emptypb.Empty{}, fmt.Errorf("error during creation: environment %s does not support vmt %s", env.GetId(), vmTemplateId)
	}

	vm := &hfv1.VirtualMachine{
		ObjectMeta: metav1.ObjectMeta{
			Name: id,
			OwnerReferences: []metav1.OwnerReference{
				{
					APIVersion: "hobbyfarm.io/v1",
					Kind:       ownerReferenceKind,
					Name:       ownerReferenceId,
					UID:        ownerReferenceUid,
				},
			},
			Labels:     labels,
			Finalizers: finalizers,
		},
		Spec: hfv1.VirtualMachineSpec{
			VirtualMachineTemplateId: vmTemplateId,
			SecretName:               secretName,
			Protocol:                 protocol,
			VirtualMachineClaimId:    vmClaimId,
			UserId:                   user,
			Provision:                provision,
			VirtualMachineSetId:      vmSetId,
		},
	}

	if sshUserName != "" {
		vm.Spec.SshUsername = sshUserName
	}

	_, err = s.vmClient.Create(ctx, vm, metav1.CreateOptions{})
	if err != nil {
		return &emptypb.Empty{}, hferrors.GrpcError(
			codes.Internal,
			err.Error(),
			req,
		)
	}
	return &emptypb.Empty{}, nil
}

func (s *GrpcVMServer) GetVM(ctx context.Context, req *generalpb.GetRequest) (*vmpb.VM, error) {
	vm, err := util.GenericHfGetter(ctx, req, s.vmClient, s.vmLister.VirtualMachines(util.GetReleaseNamespace()), "virtual machine", s.vmSynced())
	if err != nil {
		return &vmpb.VM{}, err
	}

	status := &vmpb.VMStatus{
		Status:        string(vm.Status.Status),
		Allocated:     vm.Status.Allocated,
		Tainted:       vm.Status.Tainted,
		PublicIp:      vm.Status.PublicIP,
		PrivateIp:     vm.Status.PrivateIP,
		EnvironmentId: vm.Status.EnvironmentId,
		Hostname:      vm.Status.Hostname,
		Tfstate:       vm.Status.TFState,
		WsEndpoint:    vm.Status.WsEndpoint,
	}

	var deletionTimeStamp *timestamppb.Timestamp
	if !vm.DeletionTimestamp.IsZero() {
		deletionTimeStamp = timestamppb.New(vm.DeletionTimestamp.Time)
	}

	return &vmpb.VM{
		Id:                vm.Name,
		Uid:               string(vm.UID),
		VmTemplateId:      vm.Spec.VirtualMachineTemplateId,
		SshUsername:       vm.Spec.SshUsername,
		Protocol:          vm.Spec.Protocol,
		SecretName:        vm.Spec.SecretName,
		VmClaimId:         vm.Spec.VirtualMachineClaimId,
		User:              vm.Spec.UserId,
		Provision:         vm.Spec.Provision,
		VmSetId:           vm.Spec.VirtualMachineSetId,
		Labels:            vm.Labels,
		Finalizers:        vm.Finalizers,
		Status:            status,
		Annotations:       vm.Annotations,
		DeletionTimestamp: deletionTimeStamp,
	}, nil
}

func (s *GrpcVMServer) GetVMConfig(ctx context.Context, req *vmpb.GetVMConfigRequest) (*vmpb.VMConfig, error) {
	vm, err := s.GetVM(ctx, &generalpb.GetRequest{Id: req.GetId()})
	if err != nil {
		glog.Error(err)
		return nil, hferrors.GrpcError(
			codes.Internal,
			"error while retrieving virtual machine %s",
			req,
			req.GetId(),
		)
	}

	vmt, err := s.vmTemplateClient.GetVMTemplate(ctx, &generalpb.GetRequest{Id: vm.GetVmTemplateId(), LoadFromCache: true})
	if err != nil {
		glog.Errorf("error getting vmt %v", err)
		return &vmpb.VMConfig{}, err
	}
	env, err := s.environmentClient.GetEnvironment(ctx, &generalpb.GetRequest{Id: vm.GetStatus().GetEnvironmentId(), LoadFromCache: true})
	if err != nil {
		glog.Errorf("error getting env %v", err)
		return &vmpb.VMConfig{}, err
	}

	_, exists := env.GetTemplateMapping()[vmt.GetId()]
	if !exists {
		glog.Errorf("error pulling environment template info %v", err)
		return &vmpb.VMConfig{}, fmt.Errorf("error during RFP: environment %s does not support vmt %s", env.GetId(), vmt.GetId())
	}

	config := util.GetVMConfig(env, vmt)

	for k, v := range config {
		if req.WithSecrets.GetValue() && strings.HasPrefix(v, "secret:") {
			// it was requested that secret handles will be replaced
			// secret:<name>:<field> will be replaced with the corresponding value of the linked secret.
			parts := strings.Split(v, ":")
			if len(parts) < 3 {
				continue
			}
			secretName := parts[1]
			secretField := parts[2]

			secret, err := s.secretClient.Get(ctx, secretName, metav1.GetOptions{})
			if err != nil {
				glog.Error(err)
				continue
			}

			config[k] = string(secret.Data[secretField])
		} else if strings.HasPrefix(v, "configmap:") {
			// replace configmap values
			// configmap:<name>:<field> will be replaced with the corresponding value of the linked secret.
			parts := strings.Split(v, ":")
			if len(parts) < 3 {
				continue
			}
			configMapName := parts[1]
			configMapField := parts[2]

			cm, err := s.configMapClient.Get(ctx, configMapName, metav1.GetOptions{})
			if err != nil {
				glog.Error(err)
				continue
			}

			config[k] = string(cm.Data[configMapField])
		}
	}

	return &vmpb.VMConfig{Config: config}, nil
}

func (s *GrpcVMServer) UpdateVM(ctx context.Context, req *vmpb.UpdateVMRequest) (*emptypb.Empty, error) {
	id := req.GetId()
	if len(id) == 0 {
		return &emptypb.Empty{}, hferrors.GrpcIdNotSpecifiedError(req)
	}

	bound := req.GetBound()
	vmClaimId := req.GetVmClaimId()
	user := req.GetUser()
	secretName := req.GetSecretName()
	finalizers := req.GetFinalizers()

	retryErr := retry.RetryOnConflict(retry.DefaultRetry, func() error {
		vm, err := s.vmClient.Get(ctx, id, metav1.GetOptions{})
		if err != nil {
			glog.Error(err)
			return hferrors.GrpcError(
				codes.Internal,
				"error while retrieving virtual machine %s",
				req,
				req.GetId(),
			)
		}

		if bound != "" {
			vm.Labels["bound"] = bound
		}

		if vmClaimId != nil {
			vm.Spec.VirtualMachineClaimId = vmClaimId.GetValue()
		}

		if user != nil {
			vm.Spec.UserId = user.GetValue()
		}

		if secretName != "" {
			vm.Spec.SecretName = secretName
		}

		if finalizers != nil {
			vm.SetFinalizers(finalizers.GetValues())
		}

		_, updateErr := s.vmClient.Update(ctx, vm, metav1.UpdateOptions{})
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

func (s *GrpcVMServer) UpdateVMStatus(ctx context.Context, req *vmpb.UpdateVMStatusRequest) (*emptypb.Empty, error) {
	id := req.GetId()
	if len(id) == 0 {
		return &emptypb.Empty{}, hferrors.GrpcIdNotSpecifiedError(req)
	}

	status := req.GetStatus()
	allocated := req.GetAllocated()
	tainted := req.GetTainted()
	publicIp := req.GetPublicIp()
	privateIp := req.GetPrivateIp()
	hostname := req.GetHostname()
	environmentId := req.GetEnvironmentId()
	tfState := req.GetTfstate()
	wsEndpoint := req.GetWsEndpoint()

	retryErr := retry.RetryOnConflict(retry.DefaultRetry, func() error {
		vm, err := s.vmClient.Get(ctx, id, metav1.GetOptions{})
		if err != nil {
			glog.Error(err)
			return hferrors.GrpcError(
				codes.Internal,
				"error while retrieving virtual machine %s",
				req,
				req.GetId(),
			)
		}

		if status != "" {
			vm.Status.Status = hfv1.VmStatus(status)
		}

		if allocated != nil {
			vm.Status.Allocated = allocated.GetValue()
		}

		if tainted != nil {
			vm.Status.Tainted = tainted.GetValue()
		}

		if publicIp != nil {
			vm.Status.PublicIP = publicIp.GetValue()
		}

		if privateIp != nil {
			vm.Status.PrivateIP = privateIp.GetValue()
		}

		if hostname != nil {
			vm.Status.Hostname = hostname.GetValue()
		}

		if environmentId != "" {
			vm.Status.EnvironmentId = environmentId
		}

		if tfState != "" {
			vm.Status.TFState = tfState
		}

		if wsEndpoint != "" {
			vm.Status.WsEndpoint = wsEndpoint
		}

		_, updateErr := s.vmClient.UpdateStatus(ctx, vm, metav1.UpdateOptions{})
		if updateErr != nil {
			return updateErr
		}
		// @TODO: verify result like in util.go
		glog.V(4).Infof("updated result for vm")
		return nil
	})
	if retryErr != nil {
		return &emptypb.Empty{}, hferrors.GrpcError(
			codes.Internal,
			"error attempting to update vm status: %v",
			req,
			retryErr,
		)
	}
	return &emptypb.Empty{}, nil
}

func (s *GrpcVMServer) DeleteVM(ctx context.Context, req *generalpb.ResourceId) (*emptypb.Empty, error) {
	return util.DeleteHfResource(ctx, req, s.vmClient, "virtual machine")
}

func (s *GrpcVMServer) DeleteCollectionVM(ctx context.Context, listOptions *generalpb.ListOptions) (*emptypb.Empty, error) {
	return util.DeleteHfCollection(ctx, listOptions, s.vmClient, "virtual machines")
}

func (s *GrpcVMServer) ListVM(ctx context.Context, listOptions *generalpb.ListOptions) (*vmpb.ListVMsResponse, error) {
	doLoadFromCache := listOptions.GetLoadFromCache()
	var vms []hfv1.VirtualMachine
	var err error
	if !doLoadFromCache {
		var vmList *hfv1.VirtualMachineList
		vmList, err = util.ListByHfClient(ctx, listOptions, s.vmClient, "virtual machines")
		if err == nil {
			vms = vmList.Items
		}
	} else {
		vms, err = util.ListByCache(listOptions, s.vmLister, "virtual machines", s.vmSynced())
	}
	if err != nil {
		glog.Error(err)
		return &vmpb.ListVMsResponse{}, err
	}

	preparedVms := []*vmpb.VM{}

	for _, vm := range vms {
		status := &vmpb.VMStatus{
			Status:        string(vm.Status.Status),
			Allocated:     vm.Status.Allocated,
			Tainted:       vm.Status.Tainted,
			PublicIp:      vm.Status.PublicIP,
			PrivateIp:     vm.Status.PrivateIP,
			Hostname:      vm.Status.Hostname,
			EnvironmentId: vm.Status.EnvironmentId,
			Tfstate:       vm.Status.TFState,
			WsEndpoint:    vm.Status.WsEndpoint,
		}

		var deletionTimeStamp *timestamppb.Timestamp
		if !vm.DeletionTimestamp.IsZero() {
			deletionTimeStamp = timestamppb.New(vm.DeletionTimestamp.Time)
		}

		preparedVms = append(preparedVms, &vmpb.VM{
			Id:                vm.Name,
			Uid:               string(vm.UID),
			VmTemplateId:      vm.Spec.VirtualMachineTemplateId,
			SshUsername:       vm.Spec.SshUsername,
			Protocol:          vm.Spec.Protocol,
			SecretName:        vm.Spec.SecretName,
			VmClaimId:         vm.Spec.VirtualMachineClaimId,
			User:              vm.Spec.UserId,
			Provision:         vm.Spec.Provision,
			VmSetId:           vm.Spec.VirtualMachineSetId,
			Labels:            vm.Labels,
			Finalizers:        vm.Finalizers,
			Status:            status,
			Annotations:       vm.Annotations,
			DeletionTimestamp: deletionTimeStamp,
		})
	}

	return &vmpb.ListVMsResponse{Vms: preparedVms}, nil
}
