package settingservice

import (
	"context"
	"strconv"

	"github.com/golang/glog"
	hfv1 "github.com/hobbyfarm/gargantua/v3/pkg/apis/hobbyfarm.io/v1"
	hfClientset "github.com/hobbyfarm/gargantua/v3/pkg/client/clientset/versioned"
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
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type GrpcSettingServer struct {
	settingProto.UnimplementedSettingSvcServer
	hfClientSet hfClientset.Interface
	ctx         context.Context
}

func NewGrpcSettingServer(hfClientSet hfClientset.Interface, ctx context.Context) *GrpcSettingServer {
	return &GrpcSettingServer{
		hfClientSet: hfClientSet,
		ctx:         ctx,
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
	_, err := s.hfClientSet.HobbyfarmV1().Scopes(util.GetReleaseNamespace()).Create(ctx, hfScope, metav1.CreateOptions{})
	if err != nil {
		return &empty.Empty{}, hferrors.GrpcError(
			codes.Internal,
			err.Error(),
			creq,
		)
	}
	return &empty.Empty{}, nil
}

func (s *GrpcSettingServer) GetScope(ctx context.Context, id *general.ResourceId) (*settingProto.Scope, error) {
	scope, err := s.hfClientSet.HobbyfarmV1().Scopes(util.GetReleaseNamespace()).Get(s.ctx, id.GetId(), metav1.GetOptions{})
	if errors.IsNotFound(err) {
		glog.Errorf("scope %s not found", id.GetId())
		return &settingProto.Scope{}, hferrors.GrpcError(
			codes.NotFound,
			"scope %s not found",
			id,
			id.GetId(),
		)
	}
	return &settingProto.Scope{Name: scope.Name, DisplayName: scope.DisplayName}, nil
}

func (s *GrpcSettingServer) ListScopes(ctx context.Context, listOptions *general.ListOptions) (*settingProto.Scopes, error) {
	scopes, err := s.hfClientSet.HobbyfarmV1().Scopes(util.GetReleaseNamespace()).List(s.ctx, metav1.ListOptions{
		LabelSelector: listOptions.GetLabelSelector(),
	})
	if err != nil {
		glog.Errorf("error while listing scopes: %s", err.Error())
		return &settingProto.Scopes{}, hferrors.GrpcError(
			codes.Internal,
			"error listing scopes",
			listOptions,
		)
	}

	var preparedScopes = make([]*settingProto.Scope, len(scopes.Items))

	for i, s := range scopes.Items {
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
	_, err := s.hfClientSet.HobbyfarmV1().Settings(util.GetReleaseNamespace()).Create(ctx, hfSetting, metav1.CreateOptions{})
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

func (s *GrpcSettingServer) ListSettings(ctx context.Context, lreq *general.ListOptions) (*settingProto.ListSettingsResponse, error) {
	kSettings, err := s.hfClientSet.HobbyfarmV1().Settings(util.GetReleaseNamespace()).List(s.ctx, metav1.ListOptions{
		LabelSelector: lreq.GetLabelSelector(),
	})
	if err != nil {
		glog.Errorf("error listing settings: %s", err.Error())
		return &settingProto.ListSettingsResponse{}, hferrors.GrpcError(
			codes.Internal,
			"error listing settings",
			lreq,
		)
	}

	var settings []*settingProto.PreparedListSetting
	for _, ks := range kSettings.Items {
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
		settings = append(settings, prepListSetting)
	}
	return &settingProto.ListSettingsResponse{Settings: settings}, nil

}

func (s *GrpcSettingServer) GetSetting(ctx context.Context, id *general.ResourceId) (*settingProto.Setting, error) {
	kSetting, err := s.hfClientSet.HobbyfarmV1().Settings(util.GetReleaseNamespace()).Get(s.ctx, id.GetId(), metav1.GetOptions{})
	if errors.IsNotFound(err) {
		glog.Errorf("setting %s not found", id.GetId())
		return &settingProto.Setting{}, hferrors.GrpcError(
			codes.NotFound,
			"setting %s not found",
			id,
			id.GetId(),
		)
	}
	if err != nil {
		glog.Errorf("error getting setting from database: %s", err.Error())
		return &settingProto.Setting{}, hferrors.GrpcError(
			codes.Internal,
			"error retrieving setting",
			id,
		)
	}

	// check if the user has permissions to do this action
	scope, ok := kSetting.Labels[labels.SettingScope]
	if !ok {
		glog.Errorf("setting %s does not have scope label", kSetting.Name)
		return &settingProto.Setting{}, hferrors.GrpcError(
			codes.Internal,
			"error retrieving scope label",
			id,
		)
	}

	prepSetting := &settingProto.Setting{Name: kSetting.Name, Property: &settingProto.Property{
		DataType:    settingUtil.DataTypeMappingToProto[kSetting.Property.DataType],
		ValueType:   settingUtil.ValueTypeMappingToProto[kSetting.Property.ValueType],
		DisplayName: kSetting.DisplayName,
		Required:    kSetting.Required,
		Enum:        kSetting.Enum,
		UniqueItems: kSetting.UniqueItems,
	}, Value: kSetting.Value, Scope: scope}

	if kSetting.Maximum != nil {
		prepSetting.Property.Maximum = wrapperspb.Double(*kSetting.Property.Maximum)
	}
	if kSetting.Minimum != nil {
		prepSetting.Property.Minimum = wrapperspb.Double(*kSetting.Property.Minimum)
	}
	if kSetting.MaxLength != nil {
		prepSetting.Property.MaxLength = wrapperspb.Int64(*kSetting.Property.MaxLength)
	}
	if kSetting.MinLength != nil {
		prepSetting.Property.MinLength = wrapperspb.Int64(*kSetting.Property.MinLength)
	}
	if kSetting.Format != nil {
		prepSetting.Property.Format = wrapperspb.String(*kSetting.Property.Format)
	}
	if kSetting.Pattern != nil {
		prepSetting.Property.Pattern = wrapperspb.String(*kSetting.Property.Pattern)
	}
	if kSetting.Default != nil {
		prepSetting.Property.Default = wrapperspb.String(*kSetting.Property.Default)
	}

	return prepSetting, nil
}

func (s *GrpcSettingServer) UpdateSetting(ctx context.Context, setting *settingProto.Setting) (*empty.Empty, error) {
	kSetting, err := s.hfClientSet.HobbyfarmV1().Settings(util.GetReleaseNamespace()).Get(s.ctx, setting.GetName(), metav1.GetOptions{})
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

	_, err = s.hfClientSet.HobbyfarmV1().Settings(util.GetReleaseNamespace()).Update(s.ctx, kSetting, metav1.UpdateOptions{})
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
