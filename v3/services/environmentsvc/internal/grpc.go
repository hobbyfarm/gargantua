package environmentservice

import (
	"context"
	"crypto/sha256"
	"encoding/base32"
	"strings"
	"time"

	environmentpb "github.com/hobbyfarm/gargantua/v3/protos/environment"
	generalpb "github.com/hobbyfarm/gargantua/v3/protos/general"

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

type GrpcEnvironmentServer struct {
	environmentpb.UnimplementedEnvironmentSvcServer
	environmentClient hfClientsetv1.EnvironmentInterface
	environmentLister listersv1.EnvironmentLister
	environmentSynced cache.InformerSynced
}

func NewGrpcEnvironmentServer(hfClientSet hfClientset.Interface, hfInformerFactory hfInformers.SharedInformerFactory) *GrpcEnvironmentServer {
	return &GrpcEnvironmentServer{
		environmentClient: hfClientSet.HobbyfarmV1().Environments(util.GetReleaseNamespace()),
		environmentLister: hfInformerFactory.Hobbyfarm().V1().Environments().Lister(),
		environmentSynced: hfInformerFactory.Hobbyfarm().V1().Environments().Informer().HasSynced,
	}
}

func (s *GrpcEnvironmentServer) CreateEnvironment(ctx context.Context, req *environmentpb.CreateEnvironmentRequest) (*generalpb.ResourceId, error) {
	displayName := req.GetDisplayName()
	dnsSuffix := req.GetDnssuffix() // optional
	provider := req.GetProvider()
	templateMappingRaw := req.GetTemplateMapping()
	environmentSpecificsRaw := req.GetEnvironmentSpecifics()
	ipTranslationMapRaw := req.GetIpTranslationMap()
	wsEndpoint := req.GetWsEndpoint()
	countCapacityRaw := req.GetCountCapacity()

	requiredStringParams := map[string]string{
		"displayName":          displayName,
		"provider":             provider,
		"templateMapping":      templateMappingRaw,
		"environmentSpecifics": environmentSpecificsRaw,
		"ipTranslationMap":     ipTranslationMapRaw,
		"wsEndpoint":           wsEndpoint,
		"countCapacity":        countCapacityRaw,
	}
	for param, value := range requiredStringParams {
		if value == "" {
			return &generalpb.ResourceId{}, hferrors.GrpcNotSpecifiedError(req, param)
		}
	}

	templateMapping, err := util.GenericUnmarshal[map[string]map[string]string](templateMappingRaw, "templateMapping")
	if err != nil {
		return &generalpb.ResourceId{}, hferrors.GrpcParsingError(req, "templateMapping")
	}
	countCapacity, err := util.GenericUnmarshal[map[string]int](countCapacityRaw, "countCapacity")
	if err != nil {
		return &generalpb.ResourceId{}, hferrors.GrpcParsingError(req, "countCapacity")
	}
	environmentSpecifics, err := util.GenericUnmarshal[map[string]string](environmentSpecificsRaw, "environmentSpecifics")
	if err != nil {
		return &generalpb.ResourceId{}, hferrors.GrpcParsingError(req, "environmentSpecifics")
	}
	ipTranslationMap, err := util.GenericUnmarshal[map[string]string](ipTranslationMapRaw, "ipTranslationMap")
	if err != nil {
		return &generalpb.ResourceId{}, hferrors.GrpcParsingError(req, "ipTranslationMap")
	}

	hasher := sha256.New()
	hasher.Write([]byte(time.Now().String())) // generate random name
	sha := base32.StdEncoding.WithPadding(-1).EncodeToString(hasher.Sum(nil))[:10]
	id := "env-" + strings.ToLower(sha)

	environment := &hfv1.Environment{
		ObjectMeta: metav1.ObjectMeta{
			Name: id,
		},
		Spec: hfv1.EnvironmentSpec{
			DisplayName:          displayName,
			DNSSuffix:            dnsSuffix,
			Provider:             provider,
			TemplateMapping:      templateMapping,
			EnvironmentSpecifics: environmentSpecifics,
			IPTranslationMap:     ipTranslationMap,
			WsEndpoint:           wsEndpoint,
			CountCapacity:        countCapacity,
		},
	}

	_, err = s.environmentClient.Create(ctx, environment, metav1.CreateOptions{})
	if err != nil {
		return &generalpb.ResourceId{}, hferrors.GrpcError(
			codes.Internal,
			err.Error(),
			req,
		)
	}
	return &generalpb.ResourceId{Id: id}, nil
}

func (s *GrpcEnvironmentServer) GetEnvironment(ctx context.Context, req *generalpb.GetRequest) (*environmentpb.Environment, error) {
	environment, err := util.GenericHfGetter(ctx, req, s.environmentClient, s.environmentLister.Environments(util.GetReleaseNamespace()), "environment", s.environmentSynced())
	if err != nil {
		return &environmentpb.Environment{}, err
	}

	templateMapping := make(map[string]*generalpb.StringMap)
	for templateName, keyValueMap := range environment.Spec.TemplateMapping {
		templateMapping[templateName] = &generalpb.StringMap{Value: keyValueMap}
	}

	return &environmentpb.Environment{
		Id:                   environment.Name,
		Uid:                  string(environment.UID),
		DisplayName:          environment.Spec.DisplayName,
		Dnssuffix:            environment.Spec.DNSSuffix,
		Provider:             environment.Spec.Provider,
		TemplateMapping:      templateMapping,
		EnvironmentSpecifics: environment.Spec.EnvironmentSpecifics,
		IpTranslationMap:     environment.Spec.IPTranslationMap,
		WsEndpoint:           environment.Spec.WsEndpoint,
		CountCapacity:        util.ConvertIntMap[int, uint32](environment.Spec.CountCapacity),
		Annotations:          environment.Annotations,
	}, nil
}

func (s *GrpcEnvironmentServer) UpdateEnvironment(ctx context.Context, req *environmentpb.UpdateEnvironmentRequest) (*emptypb.Empty, error) {
	id := req.GetId()
	if len(id) == 0 {
		return &emptypb.Empty{}, hferrors.GrpcIdNotSpecifiedError(req)
	}
	displayName := req.GetDisplayName()
	dnsSuffix := req.GetDnssuffix() // optional
	provider := req.GetProvider()
	templateMappingRaw := req.GetTemplateMapping()
	environmentSpecificsRaw := req.GetEnvironmentSpecifics()
	ipTranslationMapRaw := req.GetIpTranslationMap()
	wsEndpoint := req.GetWsEndpoint()
	countCapacityRaw := req.GetCountCapacity()

	retryErr := retry.RetryOnConflict(retry.DefaultRetry, func() error {
		environment, err := s.environmentClient.Get(ctx, id, metav1.GetOptions{})
		if err != nil {
			glog.Error(err)
			return hferrors.GrpcError(
				codes.Internal,
				"error while retrieving environment %s",
				req,
				req.GetId(),
			)
		}
		if displayName != "" {
			environment.Spec.DisplayName = displayName
		}
		if dnsSuffix != nil {
			environment.Spec.DNSSuffix = dnsSuffix.Value
		}
		if provider != "" {
			environment.Spec.Provider = provider
		}
		if wsEndpoint != "" {
			environment.Spec.WsEndpoint = wsEndpoint
		}
		if templateMappingRaw != "" {
			templateMapping, err := util.GenericUnmarshal[map[string]map[string]string](templateMappingRaw, "templateMapping")
			if err != nil {
				return hferrors.GrpcParsingError(req, "templateMapping")
			}
			environment.Spec.TemplateMapping = templateMapping
		}
		if environmentSpecificsRaw != "" {
			environmentSpecifics, err := util.GenericUnmarshal[map[string]string](environmentSpecificsRaw, "environmentSpecifics")
			if err != nil {
				return hferrors.GrpcParsingError(req, "environmentSpecifics")
			}
			environment.Spec.EnvironmentSpecifics = environmentSpecifics
		}
		if ipTranslationMapRaw != "" {
			ipTranslationMap, err := util.GenericUnmarshal[map[string]string](ipTranslationMapRaw, "ipTranslationMap")
			if err != nil {
				return hferrors.GrpcParsingError(req, "ipTranslationMap")
			}
			environment.Spec.IPTranslationMap = ipTranslationMap
		}
		if countCapacityRaw != "" {
			countCapacity, err := util.GenericUnmarshal[map[string]int](countCapacityRaw, "countCapacity")
			if err != nil {
				return hferrors.GrpcParsingError(req, "countCapacity")
			}
			environment.Spec.CountCapacity = countCapacity
		}

		_, updateErr := s.environmentClient.Update(ctx, environment, metav1.UpdateOptions{})
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

func (s *GrpcEnvironmentServer) DeleteEnvironment(ctx context.Context, req *generalpb.ResourceId) (*emptypb.Empty, error) {
	return util.DeleteHfResource(ctx, req, s.environmentClient, "environment")
}

func (s *GrpcEnvironmentServer) DeleteCollectionEnvironment(ctx context.Context, listOptions *generalpb.ListOptions) (*emptypb.Empty, error) {
	return util.DeleteHfCollection(ctx, listOptions, s.environmentClient, "environments")
}

func (s *GrpcEnvironmentServer) ListEnvironment(ctx context.Context, listOptions *generalpb.ListOptions) (*environmentpb.ListEnvironmentsResponse, error) {
	doLoadFromCache := listOptions.GetLoadFromCache()
	var enviroments []hfv1.Environment
	var err error
	if !doLoadFromCache {
		var enviromentList *hfv1.EnvironmentList
		enviromentList, err = util.ListByHfClient(ctx, listOptions, s.environmentClient, "environments")
		if err == nil {
			enviroments = enviromentList.Items
		}
	} else {
		enviroments, err = util.ListByCache(listOptions, s.environmentLister, "environments", s.environmentSynced())
	}
	if err != nil {
		glog.Error(err)
		return &environmentpb.ListEnvironmentsResponse{}, err
	}

	preparedEnvironments := []*environmentpb.Environment{}

	for _, environment := range enviroments {

		templateMapping := make(map[string]*generalpb.StringMap)
		for templateName, keyValueMap := range environment.Spec.TemplateMapping {
			templateMapping[templateName] = &generalpb.StringMap{Value: keyValueMap}
		}

		preparedEnvironments = append(preparedEnvironments, &environmentpb.Environment{
			Id:                   environment.Name,
			Uid:                  string(environment.UID),
			DisplayName:          environment.Spec.DisplayName,
			Dnssuffix:            environment.Spec.DNSSuffix,
			Provider:             environment.Spec.Provider,
			TemplateMapping:      templateMapping,
			EnvironmentSpecifics: environment.Spec.EnvironmentSpecifics,
			IpTranslationMap:     environment.Spec.IPTranslationMap,
			WsEndpoint:           environment.Spec.WsEndpoint,
			CountCapacity:        util.ConvertIntMap[int, uint32](environment.Spec.CountCapacity),
			Annotations:          environment.Annotations,
		})
	}

	return &environmentpb.ListEnvironmentsResponse{Environments: preparedEnvironments}, nil
}
