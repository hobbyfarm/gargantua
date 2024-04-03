package vmservice

import (
	"context"
	"fmt"
	"math/rand"

	"github.com/hobbyfarm/gargantua/v3/protos/general"
	vmProto "github.com/hobbyfarm/gargantua/v3/protos/vm"

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
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/retry"
)

type GrpcVMServer struct {
	vmProto.UnimplementedVMSvcServer
	vmClient hfClientsetv1.VirtualMachineInterface
	vmLister listersv1.VirtualMachineLister
	vmSynced cache.InformerSynced
}

func NewGrpcVMServer(hfClientSet hfClientset.Interface, hfInformerFactory hfInformers.SharedInformerFactory) *GrpcVMServer {
	return &GrpcVMServer{
		vmClient: hfClientSet.HobbyfarmV1().VirtualMachines(util.GetReleaseNamespace()),
		vmLister: hfInformerFactory.Hobbyfarm().V1().VirtualMachines().Lister(),
		vmSynced: hfInformerFactory.Hobbyfarm().V1().VirtualMachines().Informer().HasSynced,
	}
}

func (s *GrpcVMServer) CreateVM(ctx context.Context, req *vmProto.CreateVMRequest) (*empty.Empty, error) {
	var id string
	var ownerReferenceId string
	var ownerReferenceUid types.UID
	var ownerReferenceKind string

	// id := req.GetId()
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
		return &empty.Empty{}, hferrors.GrpcError(codes.InvalidArgument, "no ID and UID for owner reference provided", req)
	}

	// vm set takes precedence over vm claim
	if vmSetOwner {
		ownerReferenceId = vmSetId
		ownerReferenceUid = types.UID(vmSetUid)
		ownerReferenceKind = "VirtualMachineSet"
		id = util.GetVMSetBaseName(ownerReferenceId)
	} else {
		ownerReferenceId = vmClaimId
		ownerReferenceUid = types.UID(vmClaimUid)
		ownerReferenceKind = "VirtualMachineClaim"
		id = fmt.Sprintf("%s-%08x", ownerReferenceId, rand.Uint32())
	}

	requiredStringParams := map[string]string{
		"id":           id,
		"vmTemplateId": vmTemplateId,
		"protocol":     protocol,
	}
	for param, value := range requiredStringParams {
		if value == "" {
			return &empty.Empty{}, hferrors.GrpcNotSpecifiedError(req, param)
		}
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
			SshUsername:              sshUserName,
			SecretName:               secretName,
			Protocol:                 protocol,
			VirtualMachineClaimId:    vmClaimId,
			UserId:                   user,
			Provision:                provision,
			VirtualMachineSetId:      vmSetId,
		},
	}

	_, err := s.vmClient.Create(ctx, vm, metav1.CreateOptions{})
	if err != nil {
		return &empty.Empty{}, hferrors.GrpcError(
			codes.Internal,
			err.Error(),
			req,
		)
	}
	return &empty.Empty{}, nil
}

func (s *GrpcVMServer) GetVM(ctx context.Context, req *general.GetRequest) (*vmProto.VM, error) {
	vm, err := util.GenericHfGetter(ctx, req, s.vmClient, s.vmLister.VirtualMachines(util.GetReleaseNamespace()), "virtual machine", s.vmSynced())
	if err != nil {
		return &vmProto.VM{}, err
	}

	status := &vmProto.VMStatus{
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

	return &vmProto.VM{
		Id:           vm.Name,
		VmTemplateId: vm.Spec.VirtualMachineTemplateId,
		SshUsername:  vm.Spec.SshUsername,
		Protocol:     vm.Spec.Protocol,
		SecretName:   vm.Spec.SecretName,
		VmClaimId:    vm.Spec.VirtualMachineClaimId,
		User:         vm.Spec.UserId,
		Provision:    vm.Spec.Provision,
		VmSetId:      vm.Spec.VirtualMachineSetId,
		Labels:       vm.Labels,
		Finalizers:   vm.Finalizers,
		Status:       status,
	}, nil
}

func (s *GrpcVMServer) UpdateVM(ctx context.Context, req *vmProto.UpdateVMRequest) (*empty.Empty, error) {
	id := req.GetId()
	if len(id) == 0 {
		return &empty.Empty{}, hferrors.GrpcIdNotSpecifiedError(req)
	}

	bound := req.GetBound()
	vmClaimId := req.GetVmClaimId()
	user := req.GetUser()

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

		_, updateErr := s.vmClient.Update(ctx, vm, metav1.UpdateOptions{})
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

func (s *GrpcVMServer) UpdateVMStatus(ctx context.Context, req *vmProto.UpdateVMStatusRequest) (*empty.Empty, error) {
	id := req.GetId()
	if len(id) == 0 {
		return &empty.Empty{}, hferrors.GrpcIdNotSpecifiedError(req)
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
		return &empty.Empty{}, hferrors.GrpcError(
			codes.Internal,
			"error attempting to update vm status: %v",
			req,
			retryErr,
		)
	}
	return &empty.Empty{}, nil
}

func (s *GrpcVMServer) DeleteVM(ctx context.Context, req *general.ResourceId) (*empty.Empty, error) {
	return util.DeleteHfResource(ctx, req, s.vmClient, "virtual machine")
}

func (s *GrpcVMServer) DeleteCollectionVM(ctx context.Context, listOptions *general.ListOptions) (*empty.Empty, error) {
	return util.DeleteHfCollection(ctx, listOptions, s.vmClient, "virtual machines")
}

func (s *GrpcVMServer) ListVM(ctx context.Context, listOptions *general.ListOptions) (*vmProto.ListVMsResponse, error) {
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
		return &vmProto.ListVMsResponse{}, err
	}

	preparedVms := []*vmProto.VM{}

	for _, vm := range vms {
		status := &vmProto.VMStatus{
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

		preparedVms = append(preparedVms, &vmProto.VM{
			Id:           vm.Name,
			VmTemplateId: vm.Spec.VirtualMachineTemplateId,
			SshUsername:  vm.Spec.SshUsername,
			Protocol:     vm.Spec.Protocol,
			SecretName:   vm.Spec.SecretName,
			VmClaimId:    vm.Spec.VirtualMachineClaimId,
			User:         vm.Spec.UserId,
			Provision:    vm.Spec.Provision,
			VmSetId:      vm.Spec.VirtualMachineSetId,
			Labels:       vm.Labels,
			Finalizers:   vm.Finalizers,
			Status:       status,
		})
	}

	return &vmProto.ListVMsResponse{Vms: preparedVms}, nil
}
