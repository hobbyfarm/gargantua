package settingservice

import (
	"context"
	"sync"

	"github.com/golang/glog"
	"github.com/hobbyfarm/gargantua/pkg/labels"
	"github.com/hobbyfarm/gargantua/pkg/util"
	settingProto "github.com/hobbyfarm/gargantua/protos/setting"
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
		_, err := internalSettingServer.GetScope(ctx, &settingProto.Id{Name: scope.GetName()})
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
		_, err := internalSettingServer.GetSetting(ctx, &settingProto.Id{Name: setting.GetName()})
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

func scopes() []*settingProto.CreateScopeRequest {
	return []*settingProto.CreateScopeRequest{
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

func predefinedSettings() []*settingProto.CreateSettingRequest {
	return []*settingProto.CreateSettingRequest{
		{
			Name:      string(SettingAdminUIMOTD),
			Namespace: util.GetReleaseNamespace(),
			Labels: map[string]string{
				labels.SettingScope: "admin-ui",
			},
			Value: "",
			Property: &settingProto.Property{
				DataType:    settingProto.DataType_DATA_TYPE_STRING,
				ValueType:   settingProto.ValueType_VALUE_TYPE_SCALAR,
				DisplayName: "Admin UI MOTD",
			},
		},
		{
			Name:      string(SettingUIMOTD),
			Namespace: util.GetReleaseNamespace(),
			Labels: map[string]string{
				labels.SettingScope: "public",
			},
			Value: "",
			Property: &settingProto.Property{
				DataType:    settingProto.DataType_DATA_TYPE_STRING,
				ValueType:   settingProto.ValueType_VALUE_TYPE_SCALAR,
				DisplayName: "User UI MOTD",
			},
		},
		{
			Name:      string(SettingRegistrationDisabled),
			Namespace: util.GetReleaseNamespace(),
			Labels: map[string]string{
				labels.SettingScope: "public",
			},
			Value: "false",
			Property: &settingProto.Property{
				DataType:    settingProto.DataType_DATA_TYPE_BOOLEAN,
				ValueType:   settingProto.ValueType_VALUE_TYPE_SCALAR,
				DisplayName: "Registration Disabled",
			},
		},
		{

			Name:      string(ScheduledEventRetentionTime),
			Namespace: util.GetReleaseNamespace(),
			Labels: map[string]string{
				labels.SettingScope: "gargantua",
			},
			Value: "24",
			Property: &settingProto.Property{
				DataType:    settingProto.DataType_DATA_TYPE_INTEGER,
				ValueType:   settingProto.ValueType_VALUE_TYPE_SCALAR,
				DisplayName: "ScheduledEvent retention time (h)",
			},
		},
	}
}
