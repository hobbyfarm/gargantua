package vmtemplateservice

import (
	"context"
	"crypto/sha256"
	"encoding/base32"
	"github.com/hobbyfarm/gargantua/v3/pkg/labels"
	"strings"

	generalpb "github.com/hobbyfarm/gargantua/v3/protos/general"
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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/retry"
)

type GrpcVMTemplateServer struct {
	vmtemplatepb.UnimplementedVMTemplateSvcServer
	vmTemplateClient hfClientsetv1.VirtualMachineTemplateInterface
	vmTemplateLister listersv1.VirtualMachineTemplateLister
	vmTemplateSynced cache.InformerSynced
}

func NewGrpcVMTemplateServer(hfClientSet hfClientset.Interface, hfInformerFactory hfInformers.SharedInformerFactory) *GrpcVMTemplateServer {
	return &GrpcVMTemplateServer{
		vmTemplateClient: hfClientSet.HobbyfarmV1().VirtualMachineTemplates(util.GetReleaseNamespace()),
		vmTemplateLister: hfInformerFactory.Hobbyfarm().V1().VirtualMachineTemplates().Lister(),
		vmTemplateSynced: hfInformerFactory.Hobbyfarm().V1().VirtualMachineTemplates().Informer().HasSynced,
	}
}

func (s *GrpcVMTemplateServer) CreateVMTemplate(ctx context.Context, req *vmtemplatepb.CreateVMTemplateRequest) (*generalpb.ResourceId, error) {
	name := req.GetName()
	image := req.GetImage()
	configMapRaw := req.GetConfigMapRaw()
	costBasePrice := req.GetCostBasePrice()
	costTimeUnit := req.GetCostTimeUnit()

	requiredStringParams := map[string]string{
		"name":  name,
		"image": image,
	}
	for param, value := range requiredStringParams {
		if value == "" {
			return &generalpb.ResourceId{}, hferrors.GrpcNotSpecifiedError(req, param)
		}
	}

	hasher := sha256.New()
	hasher.Write([]byte(name))
	sha := base32.StdEncoding.WithPadding(-1).EncodeToString(hasher.Sum(nil))[:10]

	vmTemplate := &hfv1.VirtualMachineTemplate{
		ObjectMeta: metav1.ObjectMeta{
			Name: "vmt-" + strings.ToLower(sha),
		},
		Spec: hfv1.VirtualMachineTemplateSpec{
			Name:  name,
			Image: image,
		},
	}

	if configMapRaw != "" {
		configMap, err := util.GenericUnmarshal[map[string]string](configMapRaw, "config_map")
		if err != nil {
			return &generalpb.ResourceId{}, hferrors.GrpcParsingError(req, "config_map")
		}
		vmTemplate.Spec.ConfigMap = configMap
	}

	if costBasePrice != "" && costTimeUnit != "" {
		vmTemplate.ObjectMeta.Labels = map[string]string{
			labels.CostBasePrice: req.GetCostBasePrice(),
			labels.CostTimeUnit:  req.GetCostTimeUnit(),
		}
	}

	_, err := s.vmTemplateClient.Create(ctx, vmTemplate, metav1.CreateOptions{})
	if err != nil {
		return &generalpb.ResourceId{}, hferrors.GrpcError(
			codes.Internal,
			err.Error(),
			req,
		)
	}
	return &generalpb.ResourceId{Id: vmTemplate.Name}, nil
}

func (s *GrpcVMTemplateServer) GetVMTemplate(ctx context.Context, req *generalpb.GetRequest) (*vmtemplatepb.VMTemplate, error) {
	vmTemplate, err := util.GenericHfGetter(ctx, req, s.vmTemplateClient, s.vmTemplateLister.VirtualMachineTemplates(util.GetReleaseNamespace()), "virtual machine template", s.vmTemplateSynced())
	if err != nil {
		return &vmtemplatepb.VMTemplate{}, err
	}

	return &vmtemplatepb.VMTemplate{
		Id:            vmTemplate.Name,
		Uid:           string(vmTemplate.UID),
		Name:          vmTemplate.Spec.Name,
		Image:         vmTemplate.Spec.Image,
		ConfigMap:     vmTemplate.Spec.ConfigMap,
		CostBasePrice: util.RefOrNil(vmTemplate.ObjectMeta.Labels[labels.CostBasePrice]),
		CostTimeUnit:  util.RefOrNil(vmTemplate.ObjectMeta.Labels[labels.CostTimeUnit]),
	}, nil
}

func (s *GrpcVMTemplateServer) UpdateVMTemplate(ctx context.Context, req *vmtemplatepb.UpdateVMTemplateRequest) (*emptypb.Empty, error) {
	id := req.GetId()
	if len(id) == 0 {
		return &emptypb.Empty{}, hferrors.GrpcIdNotSpecifiedError(req)
	}

	name := req.GetName()
	image := req.GetImage()
	configMapRaw := req.GetConfigMapRaw()
	costBasePrice := req.GetCostBasePrice()
	costTimeUnit := req.GetCostTimeUnit()

	retryErr := retry.RetryOnConflict(retry.DefaultRetry, func() error {
		vmTemplate, err := s.vmTemplateClient.Get(ctx, id, metav1.GetOptions{})
		if err != nil {
			glog.Error(err)
			return hferrors.GrpcError(
				codes.Internal,
				"error while retrieving virtual machine template %s",
				req,
				req.GetId(),
			)
		}

		if name != "" {
			vmTemplate.Spec.Name = name
		}

		if image != "" {
			vmTemplate.Spec.Image = image
		}

		if configMapRaw != "" {
			configMap, err := util.GenericUnmarshal[map[string]string](configMapRaw, "config_map")
			if err != nil {
				return err
			}
			vmTemplate.Spec.ConfigMap = configMap
		}

		if costBasePrice == "" && costTimeUnit == "" {
			if vmTemplate.ObjectMeta.Labels != nil {
				delete(vmTemplate.ObjectMeta.Labels, labels.CostBasePrice)
				delete(vmTemplate.ObjectMeta.Labels, labels.CostTimeUnit)
			}
		}

		if costBasePrice != "" && costTimeUnit != "" {
			if vmTemplate.ObjectMeta.Labels == nil {
				vmTemplate.ObjectMeta.Labels = make(map[string]string)
			}
			vmTemplate.ObjectMeta.Labels[labels.CostBasePrice] = costBasePrice
			vmTemplate.ObjectMeta.Labels[labels.CostTimeUnit] = costTimeUnit
		}

		_, updateErr := s.vmTemplateClient.Update(ctx, vmTemplate, metav1.UpdateOptions{})
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

func (s *GrpcVMTemplateServer) DeleteVMTemplate(ctx context.Context, req *generalpb.ResourceId) (*emptypb.Empty, error) {
	return util.DeleteHfResource(ctx, req, s.vmTemplateClient, "virtual machine template")
}

func (s *GrpcVMTemplateServer) DeleteCollectionVMTemplate(ctx context.Context, listOptions *generalpb.ListOptions) (*emptypb.Empty, error) {
	return util.DeleteHfCollection(ctx, listOptions, s.vmTemplateClient, "virtual machine templates")
}

func (s *GrpcVMTemplateServer) ListVMTemplate(ctx context.Context, listOptions *generalpb.ListOptions) (*vmtemplatepb.ListVMTemplatesResponse, error) {
	doLoadFromCache := listOptions.GetLoadFromCache()
	var vmTemplates []hfv1.VirtualMachineTemplate
	var err error
	if !doLoadFromCache {
		var vmTemplateList *hfv1.VirtualMachineTemplateList
		vmTemplateList, err = util.ListByHfClient(ctx, listOptions, s.vmTemplateClient, "virtual machine templates")
		if err == nil {
			vmTemplates = vmTemplateList.Items
		}
	} else {
		vmTemplates, err = util.ListByCache(listOptions, s.vmTemplateLister, "virtual machine templates", s.vmTemplateSynced())
	}
	if err != nil {
		glog.Error(err)
		return &vmtemplatepb.ListVMTemplatesResponse{}, err
	}

	preparedVmTemplates := []*vmtemplatepb.VMTemplate{}

	for _, vmTemplate := range vmTemplates {
		preparedVmTemplates = append(preparedVmTemplates, &vmtemplatepb.VMTemplate{
			Id:            vmTemplate.Name,
			Uid:           string(vmTemplate.UID),
			Name:          vmTemplate.Spec.Name,
			Image:         vmTemplate.Spec.Image,
			ConfigMap:     vmTemplate.Spec.ConfigMap,
			CostBasePrice: util.RefOrNil(vmTemplate.ObjectMeta.Labels[labels.CostBasePrice]),
			CostTimeUnit:  util.RefOrNil(vmTemplate.ObjectMeta.Labels[labels.CostTimeUnit]),
		})
	}

	return &vmtemplatepb.ListVMTemplatesResponse{Vmtemplates: preparedVmTemplates}, nil
}
