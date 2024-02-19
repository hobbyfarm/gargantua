package vmsetservice

import (
	"context"

	"github.com/hobbyfarm/gargantua/v3/protos/general"
	vmSetProto "github.com/hobbyfarm/gargantua/v3/protos/vmset"

	"github.com/golang/glog"
	"github.com/golang/protobuf/ptypes/empty"
	hfv1 "github.com/hobbyfarm/gargantua/v3/pkg/apis/hobbyfarm.io/v1"
	hfClientset "github.com/hobbyfarm/gargantua/v3/pkg/client/clientset/versioned"
	"github.com/hobbyfarm/gargantua/v3/pkg/errors"
	"github.com/hobbyfarm/gargantua/v3/pkg/util"
	"google.golang.org/grpc/codes"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/util/retry"
)

type GrpcVMSetServer struct {
	vmSetProto.UnimplementedVMSetSvcServer
	hfClientSet hfClientset.Interface
}

func NewGrpcVMSetServer(hfClientSet hfClientset.Interface) *GrpcVMSetServer {
	return &GrpcVMSetServer{
		hfClientSet: hfClientSet,
	}
}

func (s *GrpcVMSetServer) CreateVMSet(ctx context.Context, req *vmSetProto.CreateVMSetRequest) (*empty.Empty, error) {
	id := req.GetId()
	count := req.GetCount()
	environment := req.GetEnvironment()
	vmTemplate := req.GetVmTemplate()
	baseName := req.GetBaseName()
	restrictedBind := req.GetRestrictedBind()
	restrictedBindValue := req.GetRestrictedBindValue()
	seName := req.GetSeName()
	seUid := req.GetSeUid()
	labels := req.GetLabels()

	vms := &hfv1.VirtualMachineSet{
		ObjectMeta: metav1.ObjectMeta{
			Name: id,
			OwnerReferences: []metav1.OwnerReference{
				{
					APIVersion: "hobbyfarm.io/v1",
					Kind:       "ScheduledEvent",
					Name:       seName,
					UID:        types.UID(seUid),
				},
			},
			Labels: labels,
		},
		Spec: hfv1.VirtualMachineSetSpec{
			Count:          count,
			Environment:    environment,
			RestrictedBind: restrictedBind,
			VMTemplate:     vmTemplate,
			BaseName:       baseName,
		},
	}

	if restrictedBind {
		vms.Spec.RestrictedBindValue = restrictedBindValue
	}

	_, err := s.hfClientSet.HobbyfarmV1().VirtualMachineSets(util.GetReleaseNamespace()).Create(ctx, vms, v1.CreateOptions{})
	if err != nil {
		return &empty.Empty{}, errors.GrpcError(
			codes.Internal,
			err.Error(),
			req,
		)
	}
	return &empty.Empty{}, nil
}

func (s *GrpcVMSetServer) GetVMSet(ctx context.Context, id *general.ResourceId) (*vmSetProto.VMSet, error) {
	if len(id.GetId()) == 0 {
		return &vmSetProto.VMSet{}, errors.GrpcError(
			codes.InvalidArgument,
			"no id passed in",
			id,
		)
	}
	vms, err := s.hfClientSet.HobbyfarmV1().VirtualMachineSets(util.GetReleaseNamespace()).Get(ctx, id.GetId(), v1.GetOptions{})
	if err != nil {
		glog.V(2).Infof("error while retrieving virtual machine set: %v", err)
		return &vmSetProto.VMSet{}, errors.GrpcError(
			codes.Internal,
			"error while retrieving virtual machine set by id: %s with error: %v",
			id,
			id.GetId(),
			err,
		)
	}

	vmSetVMs := []*vmSetProto.VMProvision{}

	for _, vm := range vms.Status.Machines {
		vmSetVM := &vmSetProto.VMProvision{
			VmName:   vm.VirtualMachineName,
			TfcState: vm.TFControllerState,
			TfcCm:    vm.TFControllerCM,
		}
		vmSetVMs = append(vmSetVMs, vmSetVM)
	}

	status := &vmSetProto.VMSetStatus{
		Machines:    vmSetVMs,
		Available:   uint32(vms.Status.AvailableCount),
		Provisioned: vms.Status.ProvisionedCount,
	}

	return &vmSetProto.VMSet{
		Id:                  vms.Name,
		Count:               vms.Spec.Count,
		Environment:         vms.Spec.Environment,
		VmTemplate:          vms.Spec.VMTemplate,
		BaseName:            vms.Spec.BaseName,
		RestrictedBind:      vms.Spec.RestrictedBind,
		RestrictedBindValue: vms.Spec.RestrictedBindValue,
		Labels:              vms.Labels,
		Status:              status,
	}, nil
}

func (s *GrpcVMSetServer) UpdateVMSet(ctx context.Context, req *vmSetProto.UpdateVMSetRequest) (*empty.Empty, error) {
	id := req.GetId()
	if len(id) == 0 {
		return &empty.Empty{}, errors.GrpcError(
			codes.InvalidArgument,
			"no id passed in",
			req,
		)
	}

	count := req.GetCount()
	environment := req.GetEnvironment()
	vmTemplate := req.GetVmTemplate()
	restrictedBind := req.GetRestrictedBind()

	retryErr := retry.RetryOnConflict(retry.DefaultRetry, func() error {
		vms, err := s.hfClientSet.HobbyfarmV1().VirtualMachineSets(util.GetReleaseNamespace()).Get(ctx, id, v1.GetOptions{})
		if err != nil {
			glog.Error(err)
			return errors.GrpcError(
				codes.Internal,
				"error while retrieving virtual machine set %s",
				req,
				req.GetId(),
			)
		}

		if count != nil {
			vms.Spec.Count = count.Value
		}

		if environment != "" {
			vms.Spec.Environment = environment
		}

		if vmTemplate != "" {
			vms.Spec.VMTemplate = vmTemplate
		}

		if restrictedBind != nil {
			vms.Spec.RestrictedBind = restrictedBind.Value
			vms.Spec.RestrictedBindValue = vms.Labels[util.ScheduledEventLabel]
		}
		// if restricted bind is disabled, make sure that restricted bind value is also empty
		if !vms.Spec.RestrictedBind {
			vms.Spec.RestrictedBindValue = ""
		}

		_, updateErr := s.hfClientSet.HobbyfarmV1().VirtualMachineSets(util.GetReleaseNamespace()).Update(ctx, vms, v1.UpdateOptions{})
		return updateErr
	})

	if retryErr != nil {
		return &empty.Empty{}, errors.GrpcError(
			codes.Internal,
			"error attempting to update",
			req,
		)
	}

	return &empty.Empty{}, nil
}

func (s *GrpcVMSetServer) UpdateVMSetStatus(ctx context.Context, req *vmSetProto.UpdateVMSetStatusRequest) (*empty.Empty, error) {
	id := req.GetId()
	if len(id) == 0 {
		return &empty.Empty{}, errors.GrpcError(
			codes.InvalidArgument,
			"no id passed in",
			req,
		)
	}
	machines := req.GetMachines()
	available := req.GetAvailable()
	provisioned := req.GetProvisioned()

	retryErr := retry.RetryOnConflict(retry.DefaultRetry, func() error {
		vms, err := s.hfClientSet.HobbyfarmV1().VirtualMachineSets(util.GetReleaseNamespace()).Get(ctx, id, metav1.GetOptions{})
		if err != nil {
			glog.Error(err)
			return errors.GrpcError(
				codes.Internal,
				"error while retrieving virtual machine set %s",
				req,
				req.GetId(),
			)
		}

		if available != nil {
			vms.Status.AvailableCount = int(available.Value)
		}

		if provisioned != nil {
			vms.Status.ProvisionedCount = provisioned.Value
		}

		if len(machines) > 0 {
			vmSetVMs := []hfv1.VirtualMachineProvision{}
			for key, vm := range machines {
				vmSetVM := hfv1.VirtualMachineProvision{
					VirtualMachineName: vm.VmName,
					TFControllerState:  vm.TfcState,
					TFControllerCM:     vm.TfcCm,
				}
				vmSetVMs[key] = vmSetVM
				vmSetVMs = append(vmSetVMs, vmSetVM)
			}
			vms.Status.Machines = vmSetVMs
		}

		_, updateErr := s.hfClientSet.HobbyfarmV1().VirtualMachineSets(util.GetReleaseNamespace()).UpdateStatus(ctx, vms, metav1.UpdateOptions{})
		if updateErr != nil {
			return updateErr
		}
		// @TODO: verify result like in util.go
		glog.V(4).Infof("updated result for vms")
		return nil
	})
	if retryErr != nil {
		return &empty.Empty{}, errors.GrpcError(
			codes.Internal,
			"error attempting to update vms status: %v",
			req,
			retryErr,
		)
	}
	return &empty.Empty{}, nil
}

func (s *GrpcVMSetServer) DeleteVMSet(ctx context.Context, req *general.ResourceId) (*empty.Empty, error) {
	id := req.GetId()
	if len(id) == 0 {
		return &empty.Empty{}, errors.GrpcError(
			codes.InvalidArgument,
			"no ID passed in",
			req,
		)
	}

	err := s.hfClientSet.HobbyfarmV1().VirtualMachineSets(util.GetReleaseNamespace()).Delete(ctx, id, v1.DeleteOptions{})

	if err != nil {
		glog.Errorf("error deleting virtual machine set %s: %v", id, err)
		return &empty.Empty{}, errors.GrpcError(
			codes.Internal,
			"error deleting virtual machine set %s",
			req,
			id,
		)
	}

	return &empty.Empty{}, nil
}

func (s *GrpcVMSetServer) DeleteCollectionVMSet(ctx context.Context, listOptions *general.ListOptions) (*empty.Empty, error) {
	err := s.hfClientSet.HobbyfarmV1().VirtualMachineSets(util.GetReleaseNamespace()).DeleteCollection(ctx, v1.DeleteOptions{}, v1.ListOptions{
		LabelSelector: listOptions.GetLabelSelector(),
	})
	if err != nil {
		return &empty.Empty{}, errors.GrpcError(
			codes.Internal,
			"error deleting virtual machine sets",
			listOptions,
		)
	}
	return &empty.Empty{}, nil
}

func (s *GrpcVMSetServer) ListVMSet(ctx context.Context, listOptions *general.ListOptions) (*vmSetProto.ListVMSetsResponse, error) {
	vmsList, err := s.hfClientSet.HobbyfarmV1().VirtualMachineSets(util.GetReleaseNamespace()).List(ctx, v1.ListOptions{
		LabelSelector: listOptions.GetLabelSelector(),
	})
	if err != nil {
		glog.Error(err)
		return &vmSetProto.ListVMSetsResponse{}, errors.GrpcError(
			codes.Internal,
			"error retreiving virtual machine sets",
			listOptions,
		)
	}

	preparedVmSets := []*vmSetProto.VMSet{}

	for _, vms := range vmsList.Items {
		vmSetVMs := []*vmSetProto.VMProvision{}
		for key, vm := range vms.Status.Machines {
			vmSetVM := &vmSetProto.VMProvision{
				VmName:   vm.VirtualMachineName,
				TfcState: vm.TFControllerState,
				TfcCm:    vm.TFControllerCM,
			}
			vmSetVMs[key] = vmSetVM
			vmSetVMs = append(vmSetVMs, vmSetVM)
		}

		status := &vmSetProto.VMSetStatus{
			Machines:    vmSetVMs,
			Available:   uint32(vms.Status.AvailableCount),
			Provisioned: vms.Status.ProvisionedCount,
		}

		preparedVmSets = append(preparedVmSets, &vmSetProto.VMSet{
			Id:                  vms.Name,
			Count:               vms.Spec.Count,
			Environment:         vms.Spec.Environment,
			VmTemplate:          vms.Spec.VMTemplate,
			BaseName:            vms.Spec.BaseName,
			RestrictedBind:      vms.Spec.RestrictedBind,
			RestrictedBindValue: vms.Spec.RestrictedBindValue,
			Status:              status,
			Labels:              vms.Labels,
		})
	}

	return &vmSetProto.ListVMSetsResponse{Vmsets: preparedVmSets}, nil
}
