package vmclaimservice

import (
	"context"
	"fmt"

	"github.com/hobbyfarm/gargantua/v3/protos/general"
	vmClaimProto "github.com/hobbyfarm/gargantua/v3/protos/vmclaim"

	"github.com/golang/glog"
	"github.com/golang/protobuf/ptypes/empty"
	hfv1 "github.com/hobbyfarm/gargantua/v3/pkg/apis/hobbyfarm.io/v1"
	hfClientset "github.com/hobbyfarm/gargantua/v3/pkg/client/clientset/versioned"
	hferrors "github.com/hobbyfarm/gargantua/v3/pkg/errors"
	"github.com/hobbyfarm/gargantua/v3/pkg/util"
	"google.golang.org/grpc/codes"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/util/retry"
)

type GrpcVMClaimServer struct {
	vmClaimProto.UnimplementedVMClaimSvcServer
	hfClientSet hfClientset.Interface
}

func NewGrpcVMClaimServer(hfClientSet hfClientset.Interface) *GrpcVMClaimServer {
	return &GrpcVMClaimServer{
		hfClientSet: hfClientSet,
	}
}

func (s *GrpcVMClaimServer) CreateVMClaim(ctx context.Context, req *vmClaimProto.CreateVMClaimRequest) (*empty.Empty, error) {
	id := req.GetId()
	userName := req.GetUserName()
	vmset := req.GetVmset()
	restrictedBind := req.GetRestrictedBind()
	restrictedBindValue := req.GetRestrictedBindValue()
	dynamicCapable := req.GetDynamicCapable()
	labels := req.GetLabels()
	vmClaim := &hfv1.VirtualMachineClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name:   id,
			Labels: labels,
		},
		Spec: hfv1.VirtualMachineClaimSpec{
			UserId:          userName,
			RestrictedBind:  restrictedBind,
			BaseName:        id,
			VirtualMachines: make(map[string]hfv1.VirtualMachineClaimVM),
			DynamicCapable:  dynamicCapable,
		},
	}

	if restrictedBind {
		vmClaim.Spec.RestrictedBindValue = restrictedBindValue
	}

	for vmName, vmTemplateName := range vmset {
		vmClaim.Spec.VirtualMachines[vmName] = hfv1.VirtualMachineClaimVM{Template: vmTemplateName, VirtualMachineId: ""}
		// also label this vmc so we can query against it later
		vmClaim.ObjectMeta.Labels[fmt.Sprintf("virtualmachinetemplate.hobbyfarm.io/%s", vmTemplateName)] = "true"
	}

	_, err := s.hfClientSet.HobbyfarmV1().VirtualMachineClaims(util.GetReleaseNamespace()).Create(ctx, vmClaim, v1.CreateOptions{})
	if err != nil {
		return &empty.Empty{}, hferrors.GrpcError(
			codes.Internal,
			err.Error(),
			req,
		)
	}
	return &empty.Empty{}, nil
}

func (s *GrpcVMClaimServer) GetVMClaim(ctx context.Context, id *general.ResourceId) (*vmClaimProto.VMClaim, error) {
	if len(id.GetId()) == 0 {
		return &vmClaimProto.VMClaim{}, hferrors.GrpcError(
			codes.InvalidArgument,
			"no id passed in",
			id,
		)
	}
	vmc, err := s.hfClientSet.HobbyfarmV1().VirtualMachineClaims(util.GetReleaseNamespace()).Get(ctx, id.GetId(), v1.GetOptions{})
	if errors.IsNotFound(err) {
		return &vmClaimProto.VMClaim{}, hferrors.GrpcNotFoundError(id, "virtual machine claim")
	} else if err != nil {
		glog.V(2).Infof("error while retrieving virtual machine claim: %v", err)
		return &vmClaimProto.VMClaim{}, hferrors.GrpcError(
			codes.Internal,
			"error while retrieving virtual machine claim by id: %s with error: %v",
			id,
			id.GetId(),
			err,
		)
	}

	vmClaimVMs := make(map[string]*vmClaimProto.VMClaimVM)

	for key, vm := range vmc.Spec.VirtualMachines {
		vmClaimVM := &vmClaimProto.VMClaimVM{
			Template:         vm.Template,
			VirtualMachineId: vm.VirtualMachineId,
		}
		vmClaimVMs[key] = vmClaimVM
	}

	status := &vmClaimProto.VMClaimStatus{
		Bindmode:           vmc.Status.BindMode,
		StaticBindAttempts: uint32(vmc.Status.StaticBindAttempts),
		Bound:              vmc.Status.Bound,
		Ready:              vmc.Status.Ready,
		Tainted:            vmc.Status.Ready,
	}

	return &vmClaimProto.VMClaim{
		Id:                  vmc.Name,
		UserId:              vmc.Spec.UserId,
		RestrictedBind:      vmc.Spec.RestrictedBind,
		RestrictedBindValue: vmc.Spec.RestrictedBindValue,
		Vms:                 vmClaimVMs,
		DynamicCapable:      vmc.Spec.DynamicCapable,
		BaseName:            vmc.Spec.BaseName,
		Labels:              vmc.Labels,
		Status:              status,
	}, nil
}

func (s *GrpcVMClaimServer) UpdateVMClaim(ctx context.Context, req *vmClaimProto.UpdateVMClaimRequest) (*empty.Empty, error) {
	id := req.GetId()
	if len(id) == 0 {
		return &empty.Empty{}, hferrors.GrpcError(
			codes.InvalidArgument,
			"no id passed in",
			req,
		)
	}

	vmset := req.GetVmset()
	restrictedBind := req.GetRestrictedBind()

	retryErr := retry.RetryOnConflict(retry.DefaultRetry, func() error {
		vmc, err := s.hfClientSet.HobbyfarmV1().VirtualMachineClaims(util.GetReleaseNamespace()).Get(ctx, id, v1.GetOptions{})
		if err != nil {
			glog.Error(err)
			return hferrors.GrpcError(
				codes.Internal,
				"error while retrieving virtual machine claim %s",
				req,
				req.GetId(),
			)
		}

		if restrictedBind != nil {
			vmc.Spec.RestrictedBind = restrictedBind.Value
			vmc.Spec.RestrictedBindValue = vmc.Labels[util.ScheduledEventLabel]
		}
		// if restricted bind is disabled, make sure that restricted bind value is also empty
		if !vmc.Spec.RestrictedBind {
			vmc.Spec.RestrictedBindValue = ""
		}

		if len(vmset) > 0 {
			vmClaimVMs := make(map[string]hfv1.VirtualMachineClaimVM)
			for key, vm := range vmset {
				vmClaimVM := hfv1.VirtualMachineClaimVM{
					Template:         vm.Template,
					VirtualMachineId: vm.VirtualMachineId,
				}
				vmClaimVMs[key] = vmClaimVM
			}
			vmc.Spec.VirtualMachines = vmClaimVMs
		}

		_, updateErr := s.hfClientSet.HobbyfarmV1().VirtualMachineClaims(util.GetReleaseNamespace()).Update(ctx, vmc, v1.UpdateOptions{})
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

func (s *GrpcVMClaimServer) UpdateVMClaimStatus(ctx context.Context, req *vmClaimProto.UpdateVMClaimStatusRequest) (*empty.Empty, error) {
	id := req.GetId()
	if len(id) == 0 {
		return &empty.Empty{}, hferrors.GrpcError(
			codes.InvalidArgument,
			"no id passed in",
			req,
		)
	}
	bindMode := req.GetBindmode()
	staticBindAttempts := req.GetStaticBindAttempts()
	bound := req.GetBound()
	ready := req.GetReady()
	tainted := req.GetTainted()

	retryErr := retry.RetryOnConflict(retry.DefaultRetry, func() error {
		vmc, err := s.hfClientSet.HobbyfarmV1().VirtualMachineClaims(util.GetReleaseNamespace()).Get(ctx, id, metav1.GetOptions{})
		if err != nil {
			glog.Error(err)
			return hferrors.GrpcError(
				codes.Internal,
				"error while retrieving virtual machine claim %s",
				req,
				req.GetId(),
			)
		}

		if bindMode != "" {
			vmc.Status.BindMode = bindMode
		}

		if staticBindAttempts != nil {
			vmc.Status.StaticBindAttempts = int(staticBindAttempts.Value)
		}

		if bound != nil {
			vmc.Status.Bound = bound.Value
		}

		if ready != nil {
			vmc.Status.Ready = ready.Value
		}

		if tainted != nil {
			vmc.Status.Tainted = tainted.Value
		}

		_, updateErr := s.hfClientSet.HobbyfarmV1().VirtualMachineClaims(util.GetReleaseNamespace()).UpdateStatus(ctx, vmc, metav1.UpdateOptions{})
		if updateErr != nil {
			return updateErr
		}
		// @TODO: verify result like in util.go
		glog.V(4).Infof("updated result for vmc")
		return nil
	})
	if retryErr != nil {
		return &empty.Empty{}, hferrors.GrpcError(
			codes.Internal,
			"error attempting to update vmc status: %v",
			req,
			retryErr,
		)
	}
	return &empty.Empty{}, nil
}

func (s *GrpcVMClaimServer) DeleteVMClaim(ctx context.Context, req *general.ResourceId) (*empty.Empty, error) {
	id := req.GetId()
	if len(id) == 0 {
		return &empty.Empty{}, hferrors.GrpcError(
			codes.InvalidArgument,
			"no ID passed in",
			req,
		)
	}

	err := s.hfClientSet.HobbyfarmV1().VirtualMachineClaims(util.GetReleaseNamespace()).Delete(ctx, id, v1.DeleteOptions{})

	if err != nil {
		glog.Errorf("error deleting virtual machine claim %s: %v", id, err)
		return &empty.Empty{}, hferrors.GrpcError(
			codes.Internal,
			"error deleting virtual machine claim %s",
			req,
			id,
		)
	}

	return &empty.Empty{}, nil
}

func (s *GrpcVMClaimServer) DeleteCollectionVMClaim(ctx context.Context, listOptions *general.ListOptions) (*empty.Empty, error) {
	err := s.hfClientSet.HobbyfarmV1().VirtualMachineClaims(util.GetReleaseNamespace()).DeleteCollection(ctx, v1.DeleteOptions{}, v1.ListOptions{
		LabelSelector: listOptions.GetLabelSelector(),
	})
	if err != nil {
		return &empty.Empty{}, hferrors.GrpcError(
			codes.Internal,
			"error deleting virtual machine claims",
			listOptions,
		)
	}
	return &empty.Empty{}, nil
}

func (s *GrpcVMClaimServer) ListVMClaim(ctx context.Context, listOptions *general.ListOptions) (*vmClaimProto.ListVMClaimsResponse, error) {
	vmcList, err := s.hfClientSet.HobbyfarmV1().VirtualMachineClaims(util.GetReleaseNamespace()).List(ctx, v1.ListOptions{
		LabelSelector: listOptions.GetLabelSelector(),
	})
	if err != nil {
		glog.Error(err)
		return &vmClaimProto.ListVMClaimsResponse{}, hferrors.GrpcError(
			codes.Internal,
			"error retreiving virtual machine claims",
			listOptions,
		)
	}

	preparedVmcs := []*vmClaimProto.VMClaim{}

	for _, vmc := range vmcList.Items {
		vmClaimVMs := make(map[string]*vmClaimProto.VMClaimVM)
		for key, vm := range vmc.Spec.VirtualMachines {
			vmClaimVM := &vmClaimProto.VMClaimVM{
				Template:         vm.Template,
				VirtualMachineId: vm.VirtualMachineId,
			}
			vmClaimVMs[key] = vmClaimVM
		}

		status := &vmClaimProto.VMClaimStatus{
			Bindmode:           vmc.Status.BindMode,
			StaticBindAttempts: uint32(vmc.Status.StaticBindAttempts),
			Bound:              vmc.Status.Bound,
			Ready:              vmc.Status.Ready,
			Tainted:            vmc.Status.Ready,
		}

		preparedVmcs = append(preparedVmcs, &vmClaimProto.VMClaim{
			Id:                  vmc.Name,
			UserId:              vmc.Spec.UserId,
			RestrictedBind:      vmc.Spec.RestrictedBind,
			RestrictedBindValue: vmc.Spec.RestrictedBindValue,
			Vms:                 vmClaimVMs,
			DynamicCapable:      vmc.Spec.DynamicCapable,
			BaseName:            vmc.Spec.BaseName,
			Labels:              vmc.Labels,
			Status:              status,
		})
	}

	return &vmClaimProto.ListVMClaimsResponse{Vmclaims: preparedVmcs}, nil
}
