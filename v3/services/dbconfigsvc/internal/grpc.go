package dbconfigservice

import (
	"context"
	"fmt"
	"math/rand"
	"os"
	"strings"

	dbconfigpb "github.com/hobbyfarm/gargantua/v3/protos/dbconfig"
	generalpb "github.com/hobbyfarm/gargantua/v3/protos/general"

	"github.com/golang/glog"
	hfv1 "github.com/hobbyfarm/gargantua/v3/pkg/apis/hobbyfarm.io/v1"
	hfClientset "github.com/hobbyfarm/gargantua/v3/pkg/client/clientset/versioned"
	hfClientsetv1 "github.com/hobbyfarm/gargantua/v3/pkg/client/clientset/versioned/typed/hobbyfarm.io/v1"
	hfInformers "github.com/hobbyfarm/gargantua/v3/pkg/client/informers/externalversions"
	listersv1 "github.com/hobbyfarm/gargantua/v3/pkg/client/listers/hobbyfarm.io/v1"
	hferrors "github.com/hobbyfarm/gargantua/v3/pkg/errors"
	hflabels "github.com/hobbyfarm/gargantua/v3/pkg/labels"
	"github.com/hobbyfarm/gargantua/v3/pkg/util"
	"google.golang.org/grpc/codes"
	"google.golang.org/protobuf/types/known/emptypb"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/retry"
)

type GrpcDynamicBindConfigurationServer struct {
	dbconfigpb.UnimplementedDynamicBindConfigSvcServer
	dbConfigClient hfClientsetv1.DynamicBindConfigurationInterface
	dbConfigLister listersv1.DynamicBindConfigurationLister
	dbConfigSynced cache.InformerSynced
}

var baseNameDynamicPrefix string

func init() {
	bndp := os.Getenv("HF_BASENAME_DYNAMIC_PREFIX")
	if bndp == "" {
		baseNameDynamicPrefix = "dynamic"
	} else {
		baseNameDynamicPrefix = bndp
	}
}

func NewGrpcDynamicBindConfigurationServer(hfClientSet hfClientset.Interface, hfInformerFactory hfInformers.SharedInformerFactory) *GrpcDynamicBindConfigurationServer {
	return &GrpcDynamicBindConfigurationServer{
		dbConfigClient: hfClientSet.HobbyfarmV1().DynamicBindConfigurations(util.GetReleaseNamespace()),
		dbConfigLister: hfInformerFactory.Hobbyfarm().V1().DynamicBindConfigurations().Lister(),
		dbConfigSynced: hfInformerFactory.Hobbyfarm().V1().DynamicBindConfigurations().Informer().HasSynced,
	}
}

func (s *GrpcDynamicBindConfigurationServer) CreateDynamicBindConfig(ctx context.Context, req *dbconfigpb.CreateDynamicBindConfigRequest) (*emptypb.Empty, error) {
	// create the dynamic bind configurations
	dbcRand := fmt.Sprintf("%s-%08x", baseNameDynamicPrefix, rand.Uint32())
	dbcName := strings.Join([]string{"se", req.GetSeName(), "dbc", dbcRand}, "-")
	seName := req.GetSeName()
	envName := req.GetEnvName()
	restrictedBind := req.GetRestrictedBind()
	restrictedBindValue := req.GetRestrictedBindValue()
	seUid := types.UID(req.GetSeUid())
	burstCapacity := req.GetBurstCountCapacity()
	dbc := &hfv1.DynamicBindConfiguration{
		ObjectMeta: metav1.ObjectMeta{
			Name: dbcName,
			OwnerReferences: []metav1.OwnerReference{
				{
					APIVersion: "hobbyfarm.io/v1",
					Kind:       "ScheduledEvent",
					Name:       seName,
					UID:        seUid,
				},
			},
			Labels: map[string]string{
				hflabels.EnvironmentLabel:    envName,
				hflabels.ScheduledEventLabel: seName,
				"restrictedbind":             fmt.Sprintf("%t", restrictedBind),
			},
		},
		Spec: hfv1.DynamicBindConfigurationSpec{
			Environment:        envName,
			BaseName:           dbcRand,
			BurstCountCapacity: util.ConvertMap[uint32, int](burstCapacity),
		},
	}

	if restrictedBind {
		dbc.Spec.RestrictedBind = restrictedBind
		dbc.Spec.RestrictedBindValue = restrictedBindValue
		dbc.ObjectMeta.Labels["restrictedbindvalue"] = restrictedBindValue
	}

	_, err := s.dbConfigClient.Create(ctx, dbc, metav1.CreateOptions{})
	if err != nil {
		return &emptypb.Empty{}, err
	}
	return &emptypb.Empty{}, nil
}

func (s *GrpcDynamicBindConfigurationServer) GetDynamicBindConfig(ctx context.Context, req *generalpb.GetRequest) (*dbconfigpb.DynamicBindConfig, error) {
	dbc, err := util.GenericHfGetter(ctx, req, s.dbConfigClient, s.dbConfigLister.DynamicBindConfigurations(util.GetReleaseNamespace()), "dynamic bind configuation", s.dbConfigSynced())
	if err != nil {
		return &dbconfigpb.DynamicBindConfig{}, err
	}

	return &dbconfigpb.DynamicBindConfig{
		Id:                  dbc.Name,
		Uid:                 string(dbc.UID),
		Environment:         dbc.Spec.Environment,
		RestrictedBind:      dbc.Spec.RestrictedBind,
		RestrictedBindValue: dbc.Spec.RestrictedBindValue,
		BurstCountCapacity:  util.ConvertMap[int, uint32](dbc.Spec.BurstCountCapacity),
		Labels:              dbc.Labels,
	}, nil
}

func (s *GrpcDynamicBindConfigurationServer) UpdateDynamicBindConfig(ctx context.Context, req *dbconfigpb.UpdateDynamicBindConfigRequest) (*emptypb.Empty, error) {
	id := req.GetId()
	if len(id) == 0 {
		return &emptypb.Empty{}, hferrors.GrpcIdNotSpecifiedError(req)
	}

	environment := req.GetEnvironment()
	restrictedBind := req.GetRestrictedBind()
	burstCountCapacity := req.GetBurstCountCapacity()

	retryErr := retry.RetryOnConflict(retry.DefaultRetry, func() error {
		dbc, err := s.dbConfigClient.Get(ctx, id, metav1.GetOptions{})
		if err != nil {
			glog.Error(err)
			return hferrors.GrpcError(
				codes.Internal,
				"error while retrieving dynamic bind configuration %s",
				req,
				req.GetId(),
			)
		}

		if environment != "" {
			dbc.Spec.Environment = environment
			dbc.ObjectMeta.Labels[hflabels.EnvironmentLabel] = environment
		}
		if restrictedBind != nil {
			dbc.Spec.RestrictedBind = restrictedBind.Value
			dbc.ObjectMeta.Labels["restrictedbind"] = fmt.Sprintf("%t", restrictedBind.Value)
		}
		// if restricted bind is disabled, make sure that restricted bind value is also empty...
		// else update restricted bind value to the id of scheduled event (if it is not already set)
		if !dbc.Spec.RestrictedBind {
			dbc.Spec.RestrictedBindValue = ""
			dbc.ObjectMeta.Labels["restrictedbindvalue"] = ""
		} else if dbc.Spec.RestrictedBindValue == "" {
			dbc.Spec.RestrictedBindValue = dbc.ObjectMeta.Labels[hflabels.ScheduledEventLabel]
			dbc.ObjectMeta.Labels["restrictedbindvalue"] = dbc.ObjectMeta.Labels[hflabels.ScheduledEventLabel]
		}

		if len(burstCountCapacity) > 0 {
			dbc.Spec.BurstCountCapacity = util.ConvertMap[uint32, int](burstCountCapacity)
		}

		_, updateErr := s.dbConfigClient.Update(ctx, dbc, metav1.UpdateOptions{})
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

func (s *GrpcDynamicBindConfigurationServer) DeleteDynamicBindConfig(ctx context.Context, req *generalpb.ResourceId) (*emptypb.Empty, error) {
	return util.DeleteHfResource(ctx, req, s.dbConfigClient, "dynamic bind configuration")
}

func (s *GrpcDynamicBindConfigurationServer) DeleteCollectionDynamicBindConfig(ctx context.Context, listOptions *generalpb.ListOptions) (*emptypb.Empty, error) {
	return util.DeleteHfCollection(ctx, listOptions, s.dbConfigClient, "dynamic bind configurations")
}

func (s *GrpcDynamicBindConfigurationServer) ListDynamicBindConfig(ctx context.Context, listOptions *generalpb.ListOptions) (*dbconfigpb.ListDynamicBindConfigsResponse, error) {
	doLoadFromCache := listOptions.GetLoadFromCache()
	var dbConfigs []hfv1.DynamicBindConfiguration
	var err error
	if !doLoadFromCache {
		var dbConfigList *hfv1.DynamicBindConfigurationList
		dbConfigList, err = util.ListByHfClient(ctx, listOptions, s.dbConfigClient, "dynamic bind configurations")
		if err == nil {
			dbConfigs = dbConfigList.Items
		}
	} else {
		dbConfigs, err = util.ListByCache(listOptions, s.dbConfigLister, "dynamic bind configurations", s.dbConfigSynced())
	}
	if err != nil {
		glog.Error(err)
		return &dbconfigpb.ListDynamicBindConfigsResponse{}, err
	}

	preparedDbcs := []*dbconfigpb.DynamicBindConfig{}

	for _, dbc := range dbConfigs {
		preparedDbcs = append(preparedDbcs, &dbconfigpb.DynamicBindConfig{
			Id:                  dbc.Name,
			Uid:                 string(dbc.UID),
			Environment:         dbc.Spec.Environment,
			BurstCountCapacity:  util.ConvertMap[int, uint32](dbc.Spec.BurstCountCapacity),
			RestrictedBind:      dbc.Spec.RestrictedBind,
			RestrictedBindValue: dbc.Spec.RestrictedBindValue,
			Labels:              dbc.Labels,
		})
	}

	return &dbconfigpb.ListDynamicBindConfigsResponse{DbConfig: preparedDbcs}, nil
}
