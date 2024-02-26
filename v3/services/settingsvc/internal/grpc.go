package settingservice

import (
	"context"
	"strconv"

	"github.com/golang/glog"
	hfv1 "github.com/hobbyfarm/gargantua/v3/pkg/apis/hobbyfarm.io/v1"
	hfClientset "github.com/hobbyfarm/gargantua/v3/pkg/client/clientset/versioned"
	hfClientsetv1 "github.com/hobbyfarm/gargantua/v3/pkg/client/clientset/versioned/typed/hobbyfarm.io/v1"
	hfInformers "github.com/hobbyfarm/gargantua/v3/pkg/client/informers/externalversions"
	listersv1 "github.com/hobbyfarm/gargantua/v3/pkg/client/listers/hobbyfarm.io/v1"
	hferrors "github.com/hobbyfarm/gargantua/v3/pkg/errors"
	labels "github.com/hobbyfarm/gargantua/v3/pkg/labels"
	"github.com/hobbyfarm/gargantua/v3/pkg/property"
	settingUtil "github.com/hobbyfarm/gargantua/v3/pkg/setting"
	"github.com/hobbyfarm/gargantua/v3/pkg/util"
	"github.com/hobbyfarm/gargantua/v3/protos/general"
	settingProto "github.com/hobbyfarm/gargantua/v3/protos/setting"
	"google.golang.org/grpc/codes"
	empty "google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/wrapperspb"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/cache"
)

type GrpcSettingServer struct {
	settingProto.UnimplementedSettingSvcServer
	settingClient hfClientsetv1.SettingInterface
	settingLister listersv1.SettingLister
	settingSynced cache.InformerSynced
	scopeClient   hfClientsetv1.ScopeInterface
	scopeLister   listersv1.ScopeLister
	scopeSynced   cache.InformerSynced
}

func NewGrpcSettingServer(hfClientSet hfClientset.Interface, hfInformerFactory hfInformers.SharedInformerFactory) *GrpcSettingServer {
	ns := util.GetReleaseNamespace()
	return &GrpcSettingServer{
		settingClient: hfClientSet.HobbyfarmV1().Settings(ns),
		settingLister: hfInformerFactory.Hobbyfarm().V1().Settings().Lister(),
		settingSynced: hfInformerFactory.Hobbyfarm().V1().Settings().Informer().HasSynced,
		scopeClient:   hfClientSet.HobbyfarmV1().Scopes(ns),
		scopeLister:   hfInformerFactory.Hobbyfarm().V1().Scopes().Lister(),
		scopeSynced:   hfInformerFactory.Hobbyfarm().V1().Scopes().Informer().HasSynced,
	}
}

func (s *GrpcSettingServer) CreateScope(ctx context.Context, creq *settingProto.CreateScopeRequest) (*empty.Empty, error) {
	hfScope := &hfv1.Scope{
		ObjectMeta: metav1.ObjectMeta{
			Name:      creq.GetName(),
			Namespace: creq.GetNamespace(),
		},
		DisplayName: creq.GetDisplayName(),
	}
	_, err := s.scopeClient.Create(ctx, hfScope, metav1.CreateOptions{})
	if err != nil {
		return &empty.Empty{}, hferrors.GrpcError(
			codes.Internal,
			err.Error(),
			creq,
		)
	}
	return &empty.Empty{}, nil
}

func (s *GrpcSettingServer) GetScope(ctx context.Context, req *general.GetRequest) (*settingProto.Scope, error) {
	scope, err := util.GenericHfGetter(ctx, req, s.scopeClient, s.scopeLister.Scopes(util.GetReleaseNamespace()), "scope", s.scopeSynced())
	if err != nil {
		return &settingProto.Scope{}, err
	}

	return &settingProto.Scope{Name: scope.Name, DisplayName: scope.DisplayName}, nil
}

func (s *GrpcSettingServer) ListScopes(ctx context.Context, listOptions *general.ListOptions) (*settingProto.Scopes, error) {
	doLoadFromCache := listOptions.GetLoadFromCache()
	var scopes []hfv1.Scope
	var err error
	if !doLoadFromCache {
		var scopeList *hfv1.ScopeList
		scopeList, err = util.ListByHfClient(ctx, listOptions, s.scopeClient, "scopes")
		if err == nil {
			scopes = scopeList.Items
		}
	} else {
		scopes, err = util.ListByCache(listOptions, s.scopeLister, "scopes", s.scopeSynced())
	}
	if err != nil {
		glog.Error(err)
		return &settingProto.Scopes{}, err
	}

	var preparedScopes = make([]*settingProto.Scope, len(scopes))

	for i, s := range scopes {
		preparedScopes[i] = &settingProto.Scope{
			Name:        s.Name,
			DisplayName: s.DisplayName,
		}
	}
	return &settingProto.Scopes{Scopes: preparedScopes}, nil
}

func (s *GrpcSettingServer) CreateSetting(ctx context.Context, creq *settingProto.CreateSettingRequest) (*empty.Empty, error) {
	hfSetting := &hfv1.Setting{
		ObjectMeta: metav1.ObjectMeta{
			Name:      creq.GetName(),
			Namespace: creq.GetNamespace(),
			Labels:    creq.GetLabels(),
		},
		Value: creq.GetValue(),
		Property: property.Property{
			DataType:    settingUtil.DataTypeMappingToHfTypes[creq.GetProperty().GetDataType()],
			ValueType:   settingUtil.ValueTypeMappingToHfTypes[creq.GetProperty().GetValueType()],
			DisplayName: creq.GetProperty().GetDisplayName(),
		},
	}
	_, err := s.settingClient.Create(ctx, hfSetting, metav1.CreateOptions{})
	if err != nil {
		return &empty.Empty{}, hferrors.GrpcError(
			codes.Internal,
			err.Error(),
			creq,
		)
	}
	return &empty.Empty{}, nil
}

func (s *GrpcSettingServer) GetSettingValue(ctx context.Context, sreq *general.ResourceId) (*settingProto.SettingValue, error) {
	resp := &settingProto.SettingValue{}
	setting, err := GetSetting(settingUtil.SettingName(sreq.GetId()))

	if err != nil {
		return &settingProto.SettingValue{}, hferrors.GrpcError(
			codes.InvalidArgument,
			err.Error(),
			sreq,
		)
	}

	settingValue, err := setting.FromJSON(setting.Value)

	if err != nil {
		glog.Error("could not parse JSON value")
		return &settingProto.SettingValue{}, hferrors.GrpcError(
			codes.Internal,
			"error parsing JSON value for setting %s",
			sreq,
			sreq.GetId(),
		)
	}

	switch v := settingValue.(type) {
	case bool:
		resp.Value = &settingProto.SettingValue_BoolValue{BoolValue: v}
	case string:
		resp.Value = &settingProto.SettingValue_StringValue{StringValue: v}
	case float64:
		resp.Value = &settingProto.SettingValue_Float64Value{Float64Value: v}
	case int:
		resp.Value = &settingProto.SettingValue_Int64Value{Int64Value: int64(v)}
	default:
		return &settingProto.SettingValue{}, hferrors.GrpcError(
			codes.Internal,
			"error setting %s did not match any of the following types: bool, string, float64, int",
			sreq,
			sreq.GetId(),
		)
	}
	return resp, nil
}

func (s *GrpcSettingServer) ListSettings(ctx context.Context, listOptions *general.ListOptions) (*settingProto.ListSettingsResponse, error) {
	doLoadFromCache := listOptions.GetLoadFromCache()
	var settings []hfv1.Setting
	var err error
	if !doLoadFromCache {
		var settingList *hfv1.SettingList
		settingList, err = util.ListByHfClient(ctx, listOptions, s.settingClient, "settings")
		if err == nil {
			settings = settingList.Items
		}
	} else {
		settings, err = util.ListByCache(listOptions, s.settingLister, "settings", s.settingSynced())
	}
	if err != nil {
		glog.Error(err)
		return &settingProto.ListSettingsResponse{}, err
	}

	var preparedSettings []*settingProto.PreparedListSetting
	for _, ks := range settings {
		scope := ks.Labels[labels.SettingScope]
		weight := ks.Labels[labels.SettingWeight]
		group := ks.Labels[labels.SettingGroup]
		iweight, _ := strconv.Atoi(weight)
		prepListSetting := &settingProto.PreparedListSetting{Name: ks.Name, Property: &settingProto.Property{
			DataType:    settingUtil.DataTypeMappingToProto[ks.Property.DataType],
			ValueType:   settingUtil.ValueTypeMappingToProto[ks.Property.ValueType],
			DisplayName: ks.DisplayName,
			Required:    ks.Required,
			Enum:        ks.Enum,
			UniqueItems: ks.UniqueItems,
		}, Value: ks.Value, Scope: scope, Group: group, Weight: int64(iweight)}
		if ks.Maximum != nil {
			prepListSetting.Property.Maximum = wrapperspb.Double(*ks.Property.Maximum)
		}
		if ks.Minimum != nil {
			prepListSetting.Property.Minimum = wrapperspb.Double(*ks.Property.Minimum)
		}
		if ks.MaxLength != nil {
			prepListSetting.Property.MaxLength = wrapperspb.Int64(*ks.Property.MaxLength)
		}
		if ks.MinLength != nil {
			prepListSetting.Property.MinLength = wrapperspb.Int64(*ks.Property.MinLength)
		}
		if ks.Format != nil {
			prepListSetting.Property.Format = wrapperspb.String(*ks.Property.Format)
		}
		if ks.Pattern != nil {
			prepListSetting.Property.Pattern = wrapperspb.String(*ks.Property.Pattern)
		}
		if ks.Default != nil {
			prepListSetting.Property.Default = wrapperspb.String(*ks.Property.Default)
		}
		preparedSettings = append(preparedSettings, prepListSetting)
	}
	return &settingProto.ListSettingsResponse{Settings: preparedSettings}, nil
}

func (s *GrpcSettingServer) GetSetting(ctx context.Context, req *general.GetRequest) (*settingProto.Setting, error) {
	setting, err := util.GenericHfGetter(ctx, req, s.settingClient, s.settingLister.Settings(util.GetReleaseNamespace()), "setting", s.settingSynced())
	if err != nil {
		return &settingProto.Setting{}, err
	}

	// check if the user has permissions to do this action
	scope, ok := setting.Labels[labels.SettingScope]
	if !ok {
		glog.Errorf("setting %s does not have scope label", setting.Name)
		return &settingProto.Setting{}, hferrors.GrpcError(
			codes.Internal,
			"error retrieving scope label",
			req,
		)
	}

	prepSetting := &settingProto.Setting{Name: setting.Name, Property: &settingProto.Property{
		DataType:    settingUtil.DataTypeMappingToProto[setting.Property.DataType],
		ValueType:   settingUtil.ValueTypeMappingToProto[setting.Property.ValueType],
		DisplayName: setting.DisplayName,
		Required:    setting.Required,
		Enum:        setting.Enum,
		UniqueItems: setting.UniqueItems,
	}, Value: setting.Value, Scope: scope}

	if setting.Maximum != nil {
		prepSetting.Property.Maximum = wrapperspb.Double(*setting.Property.Maximum)
	}
	if setting.Minimum != nil {
		prepSetting.Property.Minimum = wrapperspb.Double(*setting.Property.Minimum)
	}
	if setting.MaxLength != nil {
		prepSetting.Property.MaxLength = wrapperspb.Int64(*setting.Property.MaxLength)
	}
	if setting.MinLength != nil {
		prepSetting.Property.MinLength = wrapperspb.Int64(*setting.Property.MinLength)
	}
	if setting.Format != nil {
		prepSetting.Property.Format = wrapperspb.String(*setting.Property.Format)
	}
	if setting.Pattern != nil {
		prepSetting.Property.Pattern = wrapperspb.String(*setting.Property.Pattern)
	}
	if setting.Default != nil {
		prepSetting.Property.Default = wrapperspb.String(*setting.Property.Default)
	}

	return prepSetting, nil
}

func (s *GrpcSettingServer) UpdateSetting(ctx context.Context, setting *settingProto.Setting) (*empty.Empty, error) {
	kSetting, err := s.settingClient.Get(ctx, setting.GetName(), metav1.GetOptions{})
	if err != nil {
		return &empty.Empty{}, hferrors.GrpcError(
			codes.InvalidArgument,
			"error updating setting: %s",
			setting,
			err.Error(),
		)
	}

	// validate the value
	if err := kSetting.Validate(setting.GetValue()); err != nil {
		glog.Errorf("error validating value: %s", err.Error())
		return &empty.Empty{}, hferrors.GrpcError(
			codes.InvalidArgument,
			"error updating setting: %s",
			setting,
			err.Error(),
		)
	}

	kSetting.Value = setting.GetValue()

	_, err = s.settingClient.Update(ctx, kSetting, metav1.UpdateOptions{})
	if err != nil {
		glog.Errorf("error updating setting: %s", err.Error())
		return &empty.Empty{}, hferrors.GrpcError(
			codes.Internal,
			"error updating setting",
			setting,
		)
	}

	return &empty.Empty{}, nil
}
