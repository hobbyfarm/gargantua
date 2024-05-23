package settingservice

import (
	"context"
	"sync"

	"github.com/golang/glog"
	"github.com/hobbyfarm/gargantua/v3/pkg/labels"
	settingUtil "github.com/hobbyfarm/gargantua/v3/pkg/setting"
	"github.com/hobbyfarm/gargantua/v3/pkg/util"
	generalpb "github.com/hobbyfarm/gargantua/v3/protos/general"
	settingpb "github.com/hobbyfarm/gargantua/v3/protos/setting"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func Preinstall(ctx context.Context, internalSettingServer *GrpcSettingServer) {
	var wg sync.WaitGroup

	wg.Add(1)

	go func() {
		err := installResources(ctx, internalSettingServer, &wg)
		if err != nil {
			glog.Errorf("error installing resources: %s", err.Error())
		}
	}()

	wg.Wait()
}

func installResources(ctx context.Context, internalSettingServer *GrpcSettingServer, wg *sync.WaitGroup) error {
	defer wg.Done()

	for _, scope := range scopes() {
		_, err := internalSettingServer.GetScope(ctx, &generalpb.GetRequest{Id: scope.GetName()})
		if s, ok := status.FromError(err); ok && s.Code() == codes.NotFound {
			if _, err := internalSettingServer.CreateScope(ctx, scope); err != nil {
				return err
			}
			continue
		}
		if err != nil {
			return err
		}
	}

	for _, setting := range predefinedSettings() {
		_, err := internalSettingServer.GetSetting(ctx, &generalpb.GetRequest{Id: setting.GetName()})
		if s, ok := status.FromError(err); ok && s.Code() == codes.NotFound {
			if _, err := internalSettingServer.CreateSetting(ctx, setting); err != nil {
				return err
			}
			continue
		}
		if err != nil {
			return err
		}
	}
	return nil
}

func scopes() []*settingpb.CreateScopeRequest {
	return []*settingpb.CreateScopeRequest{
		{
			Name:        "public",
			Namespace:   util.GetReleaseNamespace(),
			DisplayName: "Public",
		},
		{
			Name:        "admin-ui",
			Namespace:   util.GetReleaseNamespace(),
			DisplayName: "Admin UI",
		},
		{
			Name:        "user-ui",
			Namespace:   util.GetReleaseNamespace(),
			DisplayName: "User UI",
		},
		{
			Name:        "gargantua",
			Namespace:   util.GetReleaseNamespace(),
			DisplayName: "Gargantua",
		},
	}
}

func predefinedSettings() []*settingpb.CreateSettingRequest {
	return []*settingpb.CreateSettingRequest{
		{
			Name:      string(settingUtil.SettingAdminUIMOTD),
			Namespace: util.GetReleaseNamespace(),
			Labels: map[string]string{
				labels.SettingScope: "admin-ui",
			},
			Value: "",
			Property: &settingpb.Property{
				DataType:    settingpb.DataType_DATA_TYPE_STRING,
				ValueType:   settingpb.ValueType_VALUE_TYPE_SCALAR,
				DisplayName: "Admin UI MOTD",
			},
		},
		{
			Name:      string(settingUtil.SettingUIMOTD),
			Namespace: util.GetReleaseNamespace(),
			Labels: map[string]string{
				labels.SettingScope: "public",
			},
			Value: "",
			Property: &settingpb.Property{
				DataType:    settingpb.DataType_DATA_TYPE_STRING,
				ValueType:   settingpb.ValueType_VALUE_TYPE_SCALAR,
				DisplayName: "User UI MOTD",
			},
		},
		{
			Name:      string(settingUtil.SettingRegistrationDisabled),
			Namespace: util.GetReleaseNamespace(),
			Labels: map[string]string{
				labels.SettingScope: "public",
			},
			Value: "false",
			Property: &settingpb.Property{
				DataType:    settingpb.DataType_DATA_TYPE_BOOLEAN,
				ValueType:   settingpb.ValueType_VALUE_TYPE_SCALAR,
				DisplayName: "Registration Disabled",
			},
		},
		{

			Name:      string(settingUtil.ScheduledEventRetentionTime),
			Namespace: util.GetReleaseNamespace(),
			Labels: map[string]string{
				labels.SettingScope: "gargantua",
			},
			Value: "24",
			Property: &settingpb.Property{
				DataType:    settingpb.DataType_DATA_TYPE_INTEGER,
				ValueType:   settingpb.ValueType_VALUE_TYPE_SCALAR,
				DisplayName: "ScheduledEvent retention time (h)",
			},
		},
		{
			Name:      string(settingUtil.SettingRegistrationPrivacyPolicyRequired),
			Namespace: util.GetReleaseNamespace(),
			Labels: map[string]string{
				labels.SettingScope:  "public",
				labels.SettingGroup:  "privacy-policy",
				labels.SettingWeight: "3",
			},
			Value: "false",
			Property: &settingpb.Property{
				DataType:    settingpb.DataType_DATA_TYPE_BOOLEAN,
				ValueType:   settingpb.ValueType_VALUE_TYPE_SCALAR,
				DisplayName: "Require Privacy Policy acception",
			},
		},
		{
			Name:      string(settingUtil.SettingRegistrationPrivacyPolicyLink),
			Namespace: util.GetReleaseNamespace(),
			Labels: map[string]string{
				labels.SettingScope:  "public",
				labels.SettingGroup:  "privacy-policy",
				labels.SettingWeight: "2",
			},
			Value: "",
			Property: &settingpb.Property{
				DataType:    settingpb.DataType_DATA_TYPE_STRING,
				ValueType:   settingpb.ValueType_VALUE_TYPE_SCALAR,
				DisplayName: "URL to Privacy Policy Agreement",
			},
		},
		{
			Name:      string(settingUtil.SettingRegistrationPrivacyPolicyLinkName),
			Namespace: util.GetReleaseNamespace(),
			Labels: map[string]string{
				labels.SettingScope:  "public",
				labels.SettingGroup:  "privacy-policy",
				labels.SettingWeight: "1",
			},
			Value: "",
			Property: &settingpb.Property{
				DataType:    settingpb.DataType_DATA_TYPE_STRING,
				ValueType:   settingpb.ValueType_VALUE_TYPE_SCALAR,
				DisplayName: "Privacy Policy URL Display Name",
			},
		},
		{
			Name:      string(settingUtil.ImprintLink),
			Namespace: util.GetReleaseNamespace(),
			Labels: map[string]string{
				labels.SettingScope:  "public",
				labels.SettingGroup:  "imprint",
				labels.SettingWeight: "1",
			},
			Value: "",
			Property: &settingpb.Property{
				DataType:    settingpb.DataType_DATA_TYPE_STRING,
				ValueType:   settingpb.ValueType_VALUE_TYPE_SCALAR,
				DisplayName: "URL to Imprint",
			},
		},
		{
			Name:      string(settingUtil.ImprintLinkName),
			Namespace: util.GetReleaseNamespace(),
			Labels: map[string]string{
				labels.SettingScope:  "public",
				labels.SettingGroup:  "imprint",
				labels.SettingWeight: "2",
			},
			Value: "",
			Property: &settingpb.Property{
				DataType:    settingpb.DataType_DATA_TYPE_STRING,
				ValueType:   settingpb.ValueType_VALUE_TYPE_SCALAR,
				DisplayName: "Imprint URL Display Name",
			},
		},
		{
			Name:      string(settingUtil.AboutModalButtons),
			Namespace: util.GetReleaseNamespace(),
			Labels: map[string]string{
				labels.SettingScope: "user-ui",
				labels.SettingGroup: "about-modal",
			},
			Value: "{\"HobbyFarm Project\":\"https://github.com/hobbyfarm\"}",
			Property: &settingpb.Property{
				DataType:    settingpb.DataType_DATA_TYPE_STRING,
				ValueType:   settingpb.ValueType_VALUE_TYPE_MAP,
				DisplayName: "About Modal Buttons (Title -> URL)",
			},
		},
		{
			Name:      string(settingUtil.StrictAccessCodeValidation),
			Namespace: util.GetReleaseNamespace(),
			Labels: map[string]string{
				labels.SettingScope: "gargantua",
			},
			Value: "false",
			Property: &settingpb.Property{
				DataType:    settingpb.DataType_DATA_TYPE_BOOLEAN,
				ValueType:   settingpb.ValueType_VALUE_TYPE_SCALAR,
				DisplayName: "Strict AccessCode Validation",
			},
		},
		{
			Name:      string(settingUtil.UserTokenExpiration),
			Namespace: util.GetReleaseNamespace(),
			Labels: map[string]string{
				labels.SettingScope: "gargantua",
			},
			Value: "24",
			Property: &settingpb.Property{
				DataType:    settingpb.DataType_DATA_TYPE_INTEGER,
				ValueType:   settingpb.ValueType_VALUE_TYPE_SCALAR,
				DisplayName: "User Token Expiration (hours)",
			},
		},
	}
}
