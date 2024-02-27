package vmsetservice

import (
	"context"

	"github.com/hobbyfarm/gargantua/v3/protos/general"
	vmSetProto "github.com/hobbyfarm/gargantua/v3/protos/vmset"

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
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/retry"
)

type GrpcVMSetServer struct {
	vmSetProto.UnimplementedVMSetSvcServer
	vmSetClient hfClientsetv1.VirtualMachineSetInterface
	vmSetLister listersv1.VirtualMachineSetLister
	vmSetSynced cache.InformerSynced
}

func NewGrpcVMSetServer(hfClientSet hfClientset.Interface, hfInformerFactory hfInformers.SharedInformerFactory) *GrpcVMSetServer {
	return &GrpcVMSetServer{
		vmSetClient: hfClientSet.HobbyfarmV1().VirtualMachineSets(util.GetReleaseNamespace()),
		vmSetLister: hfInformerFactory.Hobbyfarm().V1().VirtualMachineSets().Lister(),
		vmSetSynced: hfInformerFactory.Hobbyfarm().V1().VirtualMachineSets().Informer().HasSynced,
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

	_, err := s.vmSetClient.Create(ctx, vms, v1.CreateOptions{})
	if err != nil {
		return &empty.Empty{}, hferrors.GrpcError(
			codes.Internal,
			err.Error(),
			req,
		)
	}
	return &empty.Empty{}, nil
}

func (s *GrpcVMSetServer) GetVMSet(ctx context.Context, req *general.GetRequest) (*vmSetProto.VMSet, error) {
	vms, err := util.GenericHfGetter(ctx, req, s.vmSetClient, s.vmSetLister.VirtualMachineSets(util.GetReleaseNamespace()), "virtual machine set", s.vmSetSynced())
	if err != nil {
		return &vmSetProto.VMSet{}, err
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
		return &empty.Empty{}, hferrors.GrpcIdNotSpecifiedError(req)
	}

	count := req.GetCount()
	environment := req.GetEnvironment()
	vmTemplate := req.GetVmTemplate()
	restrictedBind := req.GetRestrictedBind()

	retryErr := retry.RetryOnConflict(retry.DefaultRetry, func() error {
		vms, err := s.vmSetClient.Get(ctx, id, v1.GetOptions{})
		if err != nil {
			glog.Error(err)
			return hferrors.GrpcError(
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

		_, updateErr := s.vmSetClient.Update(ctx, vms, v1.UpdateOptions{})
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

func (s *GrpcVMSetServer) UpdateVMSetStatus(ctx context.Context, req *vmSetProto.UpdateVMSetStatusRequest) (*empty.Empty, error) {
	id := req.GetId()
	if len(id) == 0 {
		return &empty.Empty{}, hferrors.GrpcIdNotSpecifiedError(req)
	}
	machines := req.GetMachines()
	available := req.GetAvailable()
	provisioned := req.GetProvisioned()

	retryErr := retry.RetryOnConflict(retry.DefaultRetry, func() error {
		vms, err := s.vmSetClient.Get(ctx, id, metav1.GetOptions{})
		if err != nil {
			glog.Error(err)
			return hferrors.GrpcError(
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

		_, updateErr := s.vmSetClient.UpdateStatus(ctx, vms, metav1.UpdateOptions{})
		if updateErr != nil {
			return updateErr
		}
		// @TODO: verify result like in util.go
		glog.V(4).Infof("updated result for vms")
		return nil
	})
	if retryErr != nil {
		return &empty.Empty{}, hferrors.GrpcError(
			codes.Internal,
			"error attempting to update vms status: %v",
			req,
			retryErr,
		)
	}
	return &empty.Empty{}, nil
}

func (s *GrpcVMSetServer) DeleteVMSet(ctx context.Context, req *general.ResourceId) (*empty.Empty, error) {
	return util.DeleteHfResource(ctx, req, s.vmSetClient, "virtual machine set")
}

func (s *GrpcVMSetServer) DeleteCollectionVMSet(ctx context.Context, listOptions *general.ListOptions) (*empty.Empty, error) {
	return util.DeleteHfCollection(ctx, listOptions, s.vmSetClient, "virtual machine sets")
}

func (s *GrpcVMSetServer) ListVMSet(ctx context.Context, listOptions *general.ListOptions) (*vmSetProto.ListVMSetsResponse, error) {
	doLoadFromCache := listOptions.GetLoadFromCache()
	var vmSets []hfv1.VirtualMachineSet
	var err error
	if !doLoadFromCache {
		var vmSetList *hfv1.VirtualMachineSetList
		vmSetList, err = util.ListByHfClient(ctx, listOptions, s.vmSetClient, "virtual machine sets")
		if err == nil {
			vmSets = vmSetList.Items
		}
	} else {
		vmSets, err = util.ListByCache(listOptions, s.vmSetLister, "virtual machine sets", s.vmSetSynced())
	}
	if err != nil {
		glog.Error(err)
		return &vmSetProto.ListVMSetsResponse{}, err
	}

	preparedVmSets := []*vmSetProto.VMSet{}

	for _, vms := range vmSets {
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
