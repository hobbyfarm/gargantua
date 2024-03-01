package vmtemplateservice

import (
	"context"
	"crypto/sha256"
	"encoding/base32"
	"strings"

	"github.com/hobbyfarm/gargantua/v3/protos/general"
	vmTemplateProto "github.com/hobbyfarm/gargantua/v3/protos/vmtemplate"

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
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/retry"
)

type GrpcVMTemplateServer struct {
	vmTemplateProto.UnimplementedVMTemplateSvcServer
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

func (s *GrpcVMTemplateServer) CreateVMTemplate(ctx context.Context, req *vmTemplateProto.CreateVMTemplateRequest) (*empty.Empty, error) {
	name := req.GetName()
	image := req.GetImage()
	configMapRaw := req.GetConfigMapRaw()

	requiredStringParams := map[string]string{
		"name":  name,
		"image": image,
	}
	for param, value := range requiredStringParams {
		if value == "" {
			return &empty.Empty{}, hferrors.GrpcNotSpecifiedError(req, param)
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
			return &empty.Empty{}, hferrors.GrpcParsingError(req, "config_map")
		}
		vmTemplate.Spec.ConfigMap = configMap
	}

	_, err := s.vmTemplateClient.Create(ctx, vmTemplate, metav1.CreateOptions{})
	if err != nil {
		return &empty.Empty{}, hferrors.GrpcError(
			codes.Internal,
			err.Error(),
			req,
		)
	}
	return &empty.Empty{}, nil
}

func (s *GrpcVMTemplateServer) GetVMTemplate(ctx context.Context, req *general.GetRequest) (*vmTemplateProto.VMTemplate, error) {
	vmTemplate, err := util.GenericHfGetter(ctx, req, s.vmTemplateClient, s.vmTemplateLister.VirtualMachineTemplates(util.GetReleaseNamespace()), "virtual machine template", s.vmTemplateSynced())
	if err != nil {
		return &vmTemplateProto.VMTemplate{}, err
	}

	return &vmTemplateProto.VMTemplate{
		Id:        vmTemplate.Name,
		Name:      vmTemplate.Spec.Name,
		Image:     vmTemplate.Spec.Image,
		ConfigMap: vmTemplate.Spec.ConfigMap,
	}, nil
}

func (s *GrpcVMTemplateServer) UpdateVMTemplate(ctx context.Context, req *vmTemplateProto.UpdateVMTemplateRequest) (*empty.Empty, error) {
	id := req.GetId()
	if len(id) == 0 {
		return &empty.Empty{}, hferrors.GrpcIdNotSpecifiedError(req)
	}

	name := req.GetName()
	image := req.GetImage()
	configMapRaw := req.GetConfigMapRaw()

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

		_, updateErr := s.vmTemplateClient.Update(ctx, vmTemplate, metav1.UpdateOptions{})
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

func (s *GrpcVMTemplateServer) DeleteVMTemplate(ctx context.Context, req *general.ResourceId) (*empty.Empty, error) {
	return util.DeleteHfResource(ctx, req, s.vmTemplateClient, "virtual machine template")
}

func (s *GrpcVMTemplateServer) DeleteCollectionVMTemplate(ctx context.Context, listOptions *general.ListOptions) (*empty.Empty, error) {
	return util.DeleteHfCollection(ctx, listOptions, s.vmTemplateClient, "virtual machine templates")
}

func (s *GrpcVMTemplateServer) ListVMTemplate(ctx context.Context, listOptions *general.ListOptions) (*vmTemplateProto.ListVMTemplatesResponse, error) {
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
		return &vmTemplateProto.ListVMTemplatesResponse{}, err
	}

	preparedVmTemplates := []*vmTemplateProto.VMTemplate{}

	for _, vmTemplate := range vmTemplates {
		preparedVmTemplates = append(preparedVmTemplates, &vmTemplateProto.VMTemplate{
			Id:        vmTemplate.Name,
			Name:      vmTemplate.Spec.Name,
			Image:     vmTemplate.Spec.Image,
			ConfigMap: vmTemplate.Spec.ConfigMap,
		})
	}

	return &vmTemplateProto.ListVMTemplatesResponse{Vmtemplates: preparedVmTemplates}, nil
}
