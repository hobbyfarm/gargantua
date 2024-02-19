package dbconfigservice

import (
	"context"
	"fmt"
	"math/rand"
	"os"
	"strings"

	dbConfigProto "github.com/hobbyfarm/gargantua/v3/protos/dbconfig"
	"github.com/hobbyfarm/gargantua/v3/protos/general"

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

type GrpcDynamicBindConfigurationServer struct {
	dbConfigProto.UnimplementedDynamicBindConfigSvcServer
	hfClientSet hfClientset.Interface
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

func NewGrpcDynamicBindConfigurationServer(hfClientSet hfClientset.Interface) *GrpcDynamicBindConfigurationServer {
	return &GrpcDynamicBindConfigurationServer{
		hfClientSet: hfClientSet,
	}
}

func (s *GrpcDynamicBindConfigurationServer) CreateDynamicBindConfig(ctx context.Context, req *dbConfigProto.CreateDynamicBindConfigRequest) (*empty.Empty, error) {
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
				util.EnvironmentLabel:    envName,
				util.ScheduledEventLabel: seName,
				"restrictedbind":         fmt.Sprintf("%t", restrictedBind),
			},
		},
		Spec: hfv1.DynamicBindConfigurationSpec{
			Environment:        envName,
			BaseName:           dbcRand,
			BurstCountCapacity: burstCapacity,
		},
	}

	if restrictedBind {
		dbc.Spec.RestrictedBind = restrictedBind
		dbc.Spec.RestrictedBindValue = restrictedBindValue
		dbc.ObjectMeta.Labels["restrictedbindvalue"] = restrictedBindValue
	}

	_, err := s.hfClientSet.HobbyfarmV1().DynamicBindConfigurations(util.GetReleaseNamespace()).Create(ctx, dbc, v1.CreateOptions{})
	if err != nil {
		return &empty.Empty{}, err
	}
	return &empty.Empty{}, nil
}

func (s *GrpcDynamicBindConfigurationServer) GetDynamicBindConfig(ctx context.Context, id *general.ResourceId) (*dbConfigProto.DynamicBindConfig, error) {
	if len(id.GetId()) == 0 {
		return &dbConfigProto.DynamicBindConfig{}, errors.GrpcError(
			codes.InvalidArgument,
			"no id passed in",
			id,
		)
	}
	dbc, err := s.hfClientSet.HobbyfarmV1().DynamicBindConfigurations(util.GetReleaseNamespace()).Get(ctx, id.GetId(), v1.GetOptions{})
	if err != nil {
		glog.V(2).Infof("error while retrieving dynamic bind configuration: %v", err)
		return &dbConfigProto.DynamicBindConfig{}, errors.GrpcError(
			codes.Internal,
			"error while retrieving dynamic bind configuration by id: %s with error: %v",
			id,
			id.GetId(),
			err,
		)
	}

	return &dbConfigProto.DynamicBindConfig{
		Id:                  dbc.Name,
		Environment:         dbc.Spec.Environment,
		RestrictedBind:      dbc.Spec.RestrictedBind,
		RestrictedBindValue: dbc.Spec.RestrictedBindValue,
		BurstCountCapacity:  dbc.Spec.BurstCountCapacity,
		Labels:              dbc.Labels,
	}, nil
}

func (s *GrpcDynamicBindConfigurationServer) UpdateDynamicBindConfig(ctx context.Context, req *dbConfigProto.UpdateDynamicBindConfigRequest) (*empty.Empty, error) {
	id := req.GetId()
	if len(id) == 0 {
		return &empty.Empty{}, errors.GrpcError(
			codes.InvalidArgument,
			"no id passed in",
			req,
		)
	}

	environment := req.GetEnvironment()
	restrictedBind := req.GetRestrictedBind()
	burstCountCapacity := req.GetBurstCountCapacity()

	retryErr := retry.RetryOnConflict(retry.DefaultRetry, func() error {
		dbc, err := s.hfClientSet.HobbyfarmV1().DynamicBindConfigurations(util.GetReleaseNamespace()).Get(ctx, id, v1.GetOptions{})
		if err != nil {
			glog.Error(err)
			return errors.GrpcError(
				codes.Internal,
				"error while retrieving dynamic bind configuration %s",
				req,
				req.GetId(),
			)
		}

		if environment != "" {
			dbc.Spec.Environment = environment
			dbc.ObjectMeta.Labels[util.EnvironmentLabel] = environment
		}
		if restrictedBind != nil {
			dbc.Spec.RestrictedBind = restrictedBind.Value
			dbc.ObjectMeta.Labels["restrictedbind"] = fmt.Sprintf("%t", restrictedBind)
		}
		// if restricted bind is disabled, make sure that restricted bind value is also empty...
		// else update restricted bind value to the id of scheduled event (if it is not already set)
		if !dbc.Spec.RestrictedBind {
			dbc.Spec.RestrictedBindValue = ""
			dbc.ObjectMeta.Labels["restrictedbindvalue"] = ""
		} else if dbc.Spec.RestrictedBindValue == "" {
			dbc.Spec.RestrictedBindValue = dbc.ObjectMeta.Labels[util.ScheduledEventLabel]
			dbc.ObjectMeta.Labels["restrictedbindvalue"] = dbc.ObjectMeta.Labels[util.ScheduledEventLabel]
		}

		if len(burstCountCapacity) > 0 {
			dbc.Spec.BurstCountCapacity = burstCountCapacity
		}

		_, updateErr := s.hfClientSet.HobbyfarmV1().DynamicBindConfigurations(util.GetReleaseNamespace()).Update(ctx, dbc, v1.UpdateOptions{})
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

func (s *GrpcDynamicBindConfigurationServer) DeleteDynamicBindConfig(ctx context.Context, req *general.ResourceId) (*empty.Empty, error) {
	id := req.GetId()
	if len(id) == 0 {
		return &empty.Empty{}, errors.GrpcError(
			codes.InvalidArgument,
			"no ID passed in",
			req,
		)
	}

	err := s.hfClientSet.HobbyfarmV1().DynamicBindConfigurations(util.GetReleaseNamespace()).Delete(ctx, id, v1.DeleteOptions{})

	if err != nil {
		glog.Errorf("error deleting dynamic bind configuration %s: %v", id, err)
		return &empty.Empty{}, errors.GrpcError(
			codes.Internal,
			"error deleting dynamic bind configuration %s",
			req,
			id,
		)
	}

	return &empty.Empty{}, nil
}

func (s *GrpcDynamicBindConfigurationServer) DeleteCollectionDynamicBindConfig(ctx context.Context, listOptions *general.ListOptions) (*empty.Empty, error) {
	err := s.hfClientSet.HobbyfarmV1().DynamicBindConfigurations(util.GetReleaseNamespace()).DeleteCollection(ctx, v1.DeleteOptions{}, v1.ListOptions{
		LabelSelector: listOptions.GetLabelSelector(),
	})
	if err != nil {
		return &empty.Empty{}, errors.GrpcError(
			codes.Internal,
			"error deleting dynamic bind configurations",
			listOptions,
		)
	}
	return &empty.Empty{}, nil
}

func (s *GrpcDynamicBindConfigurationServer) ListDynamicBindConfig(ctx context.Context, listOptions *general.ListOptions) (*dbConfigProto.ListDynamicBindConfigsResponse, error) {
	dbcList, err := s.hfClientSet.HobbyfarmV1().DynamicBindConfigurations(util.GetReleaseNamespace()).List(ctx, v1.ListOptions{
		LabelSelector: listOptions.GetLabelSelector(),
	})
	if err != nil {
		glog.Error(err)
		return &dbConfigProto.ListDynamicBindConfigsResponse{}, errors.GrpcError(
			codes.Internal,
			"error retreiving dynamic bind configurations",
			listOptions,
		)
	}
	preparedDbcs := []*dbConfigProto.DynamicBindConfig{}

	for _, dbc := range dbcList.Items {
		preparedDbcs = append(preparedDbcs, &dbConfigProto.DynamicBindConfig{
			Id:                  dbc.Name,
			Environment:         dbc.Spec.Environment,
			BurstCountCapacity:  dbc.Spec.BurstCountCapacity,
			RestrictedBind:      dbc.Spec.RestrictedBind,
			RestrictedBindValue: dbc.Spec.RestrictedBindValue,
			Labels:              dbc.Labels,
		})
	}

	return &dbConfigProto.ListDynamicBindConfigsResponse{DbConfig: preparedDbcs}, nil
}
