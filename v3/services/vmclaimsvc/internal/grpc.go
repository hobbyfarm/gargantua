package vmclaimservice

import (
	"context"
	"fmt"

	hfInformers "github.com/hobbyfarm/gargantua/v3/pkg/client/informers/externalversions"
	generalpb "github.com/hobbyfarm/gargantua/v3/protos/general"
	vmclaimpb "github.com/hobbyfarm/gargantua/v3/protos/vmclaim"

	"github.com/golang/glog"
	hfv1 "github.com/hobbyfarm/gargantua/v3/pkg/apis/hobbyfarm.io/v1"
	hfClientset "github.com/hobbyfarm/gargantua/v3/pkg/client/clientset/versioned"
	hfClientsetv1 "github.com/hobbyfarm/gargantua/v3/pkg/client/clientset/versioned/typed/hobbyfarm.io/v1"
	listersv1 "github.com/hobbyfarm/gargantua/v3/pkg/client/listers/hobbyfarm.io/v1"
	hferrors "github.com/hobbyfarm/gargantua/v3/pkg/errors"
	hflabels "github.com/hobbyfarm/gargantua/v3/pkg/labels"
	"github.com/hobbyfarm/gargantua/v3/pkg/util"
	"google.golang.org/grpc/codes"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/retry"
	"k8s.io/client-go/util/workqueue"
)

type GrpcVMClaimServer struct {
	vmclaimpb.UnimplementedVMClaimSvcServer
	vmClaimClient    hfClientsetv1.VirtualMachineClaimInterface
	vmClaimLister    listersv1.VirtualMachineClaimLister
	vmClaimSynced    cache.InformerSynced
	vmClaimWorkqueue workqueue.DelayingInterface
}

func NewGrpcVMClaimServer(
	hfClientSet hfClientset.Interface,
	hfInformerFactory hfInformers.SharedInformerFactory,
	workqueue workqueue.DelayingInterface,
) *GrpcVMClaimServer {
	return &GrpcVMClaimServer{
		vmClaimClient:    hfClientSet.HobbyfarmV1().VirtualMachineClaims(util.GetReleaseNamespace()),
		vmClaimLister:    hfInformerFactory.Hobbyfarm().V1().VirtualMachineClaims().Lister(),
		vmClaimSynced:    hfInformerFactory.Hobbyfarm().V1().VirtualMachineClaims().Informer().HasSynced,
		vmClaimWorkqueue: workqueue,
	}
}

func (s *GrpcVMClaimServer) CreateVMClaim(ctx context.Context, req *vmclaimpb.CreateVMClaimRequest) (*emptypb.Empty, error) {
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

	_, err := s.vmClaimClient.Create(ctx, vmClaim, metav1.CreateOptions{})
	if err != nil {
		return &emptypb.Empty{}, hferrors.GrpcError(
			codes.Internal,
			err.Error(),
			req,
		)
	}
	return &emptypb.Empty{}, nil
}

func (s *GrpcVMClaimServer) GetVMClaim(ctx context.Context, req *generalpb.GetRequest) (*vmclaimpb.VMClaim, error) {
	vmc, err := util.GenericHfGetter(ctx, req, s.vmClaimClient, s.vmClaimLister.VirtualMachineClaims(util.GetReleaseNamespace()), "virtual machine claim", s.vmClaimSynced())
	if err != nil {
		return &vmclaimpb.VMClaim{}, err
	}

	vmClaimVMs := make(map[string]*vmclaimpb.VMClaimVM)

	for key, vm := range vmc.Spec.VirtualMachines {
		vmClaimVM := &vmclaimpb.VMClaimVM{
			Template:         vm.Template,
			VirtualMachineId: vm.VirtualMachineId,
		}
		vmClaimVMs[key] = vmClaimVM
	}

	status := &vmclaimpb.VMClaimStatus{
		BindMode:           vmc.Status.BindMode,
		StaticBindAttempts: uint32(vmc.Status.StaticBindAttempts),
		Bound:              vmc.Status.Bound,
		Ready:              vmc.Status.Ready,
		Tainted:            vmc.Status.Ready,
	}

	var deletionTimeStamp *timestamppb.Timestamp
	if !vmc.DeletionTimestamp.IsZero() {
		deletionTimeStamp = timestamppb.New(vmc.DeletionTimestamp.Time)
	}

	return &vmclaimpb.VMClaim{
		Id:                  vmc.Name,
		Uid:                 string(vmc.UID),
		UserId:              vmc.Spec.UserId,
		RestrictedBind:      vmc.Spec.RestrictedBind,
		RestrictedBindValue: vmc.Spec.RestrictedBindValue,
		Vms:                 vmClaimVMs,
		DynamicCapable:      vmc.Spec.DynamicCapable,
		BaseName:            vmc.Spec.BaseName,
		Labels:              vmc.Labels,
		Status:              status,
		DeletionTimestamp:   deletionTimeStamp,
	}, nil
}

func (s *GrpcVMClaimServer) UpdateVMClaim(ctx context.Context, req *vmclaimpb.UpdateVMClaimRequest) (*emptypb.Empty, error) {
	id := req.GetId()
	if len(id) == 0 {
		return &emptypb.Empty{}, hferrors.GrpcIdNotSpecifiedError(req)
	}

	vmset := req.GetVmset()
	restrictedBind := req.GetRestrictedBind()

	retryErr := retry.RetryOnConflict(retry.DefaultRetry, func() error {
		vmc, err := s.vmClaimClient.Get(ctx, id, metav1.GetOptions{})
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
			vmc.Spec.RestrictedBindValue = vmc.Labels[hflabels.ScheduledEventLabel]
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

		_, updateErr := s.vmClaimClient.Update(ctx, vmc, metav1.UpdateOptions{})
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

func (s *GrpcVMClaimServer) UpdateVMClaimStatus(ctx context.Context, req *vmclaimpb.UpdateVMClaimStatusRequest) (*emptypb.Empty, error) {
	id := req.GetId()
	if len(id) == 0 {
		return &emptypb.Empty{}, hferrors.GrpcIdNotSpecifiedError(req)
	}
	bindMode := req.GetBindMode()
	staticBindAttempts := req.GetStaticBindAttempts()
	bound := req.GetBound()
	ready := req.GetReady()
	tainted := req.GetTainted()

	retryErr := retry.RetryOnConflict(retry.DefaultRetry, func() error {
		vmc, err := s.vmClaimClient.Get(ctx, id, metav1.GetOptions{})
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

		_, updateErr := s.vmClaimClient.UpdateStatus(ctx, vmc, metav1.UpdateOptions{})
		if updateErr != nil {
			return updateErr
		}
		// @TODO: verify result like in util.go
		glog.V(4).Infof("updated result for vmc")
		return nil
	})
	if retryErr != nil {
		return &emptypb.Empty{}, hferrors.GrpcError(
			codes.Internal,
			"error attempting to update vmc status: %v",
			req,
			retryErr,
		)
	}
	return &emptypb.Empty{}, nil
}

func (s *GrpcVMClaimServer) DeleteVMClaim(ctx context.Context, req *generalpb.ResourceId) (*emptypb.Empty, error) {
	return util.DeleteHfResource(ctx, req, s.vmClaimClient, "virtual machine claim")
}

func (s *GrpcVMClaimServer) DeleteCollectionVMClaim(ctx context.Context, listOptions *generalpb.ListOptions) (*emptypb.Empty, error) {
	return util.DeleteHfCollection(ctx, listOptions, s.vmClaimClient, "virtual machine claims")
}

func (s *GrpcVMClaimServer) ListVMClaim(ctx context.Context, listOptions *generalpb.ListOptions) (*vmclaimpb.ListVMClaimsResponse, error) {
	doLoadFromCache := listOptions.GetLoadFromCache()
	var vmClaims []hfv1.VirtualMachineClaim
	var err error
	if !doLoadFromCache {
		var vmClaimList *hfv1.VirtualMachineClaimList
		vmClaimList, err = util.ListByHfClient(ctx, listOptions, s.vmClaimClient, "virtual machine claims")
		if err == nil {
			vmClaims = vmClaimList.Items
		}
	} else {
		vmClaims, err = util.ListByCache(listOptions, s.vmClaimLister, "virtual machine claims", s.vmClaimSynced())
	}
	if err != nil {
		glog.Error(err)
		return &vmclaimpb.ListVMClaimsResponse{}, err
	}

	preparedVmcs := []*vmclaimpb.VMClaim{}

	for _, vmc := range vmClaims {
		vmClaimVMs := make(map[string]*vmclaimpb.VMClaimVM)
		for key, vm := range vmc.Spec.VirtualMachines {
			vmClaimVM := &vmclaimpb.VMClaimVM{
				Template:         vm.Template,
				VirtualMachineId: vm.VirtualMachineId,
			}
			vmClaimVMs[key] = vmClaimVM
		}

		status := &vmclaimpb.VMClaimStatus{
			BindMode:           vmc.Status.BindMode,
			StaticBindAttempts: uint32(vmc.Status.StaticBindAttempts),
			Bound:              vmc.Status.Bound,
			Ready:              vmc.Status.Ready,
			Tainted:            vmc.Status.Ready,
		}

		var deletionTimeStamp *timestamppb.Timestamp
		if !vmc.DeletionTimestamp.IsZero() {
			deletionTimeStamp = timestamppb.New(vmc.DeletionTimestamp.Time)
		}

		preparedVmcs = append(preparedVmcs, &vmclaimpb.VMClaim{
			Id:                  vmc.Name,
			Uid:                 string(vmc.UID),
			UserId:              vmc.Spec.UserId,
			RestrictedBind:      vmc.Spec.RestrictedBind,
			RestrictedBindValue: vmc.Spec.RestrictedBindValue,
			Vms:                 vmClaimVMs,
			DynamicCapable:      vmc.Spec.DynamicCapable,
			BaseName:            vmc.Spec.BaseName,
			Labels:              vmc.Labels,
			Status:              status,
			DeletionTimestamp:   deletionTimeStamp,
		})
	}

	return &vmclaimpb.ListVMClaimsResponse{Vmclaims: preparedVmcs}, nil
}

func (s *GrpcVMClaimServer) AddToWorkqueue(ctx context.Context, req *generalpb.ResourceId) (*emptypb.Empty, error) {
	return util.AddToWorkqueue(s.vmClaimWorkqueue, req)
}
