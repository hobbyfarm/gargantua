package settings

import (
	"context"
	v1 "github.com/hobbyfarm/gargantua/pkg/apis/hobbyfarm.io/v1"
	hfClientset "github.com/hobbyfarm/gargantua/pkg/client/clientset/versioned"
	"github.com/hobbyfarm/gargantua/pkg/labels"
	"github.com/hobbyfarm/gargantua/pkg/property"
	"github.com/hobbyfarm/gargantua/pkg/util"
	"k8s.io/apimachinery/pkg/api/errors"
	v12 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sync"
)

func InstallResources(ctx context.Context, client *hfClientset.Clientset, wg *sync.WaitGroup) error {
	defer wg.Done()

	for _, scope := range scopes() {
		_, err := client.HobbyfarmV1().Scopes(util.GetReleaseNamespace()).Get(ctx, scope.Name, v12.GetOptions{})
		if errors.IsNotFound(err) {
			if _, err := client.HobbyfarmV1().
				Scopes(util.GetReleaseNamespace()).Create(ctx, scope, v12.CreateOptions{}); err != nil {
				return err
			}
			continue
		}

		if err != nil {
			return err
		}
	}

	for _, setting := range settings() {
		_, err := client.HobbyfarmV1().Settings(util.GetReleaseNamespace()).Get(ctx, setting.Name, v12.GetOptions{})
		if errors.IsNotFound(err) {
			if _, err := client.HobbyfarmV1().
				Settings(util.GetReleaseNamespace()).Create(ctx, setting, v12.CreateOptions{}); err != nil {
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

func scopes() []*v1.Scope {
	return []*v1.Scope{
		{
			ObjectMeta: v12.ObjectMeta{
				Name:      "public",
				Namespace: util.GetReleaseNamespace(),
			},
			DisplayName: "Public",
		},
		{
			ObjectMeta: v12.ObjectMeta{
				Name:      "admin-ui",
				Namespace: util.GetReleaseNamespace(),
			},
			DisplayName: "Admin UI",
		},
		{
			ObjectMeta: v12.ObjectMeta{
				Name:      "user-ui",
				Namespace: util.GetReleaseNamespace(),
			},
			DisplayName: "User UI",
		},
	}
}

func settings() []*v1.Setting {
	return []*v1.Setting{
		{
			ObjectMeta: v12.ObjectMeta{
				Name:      "motd-admin-ui",
				Namespace: util.GetReleaseNamespace(),
				Labels: map[string]string{
					labels.SettingScope: "admin-ui",
				},
			},
			Value: "",
			Property: property.Property{
				DataType:    property.DataTypeString,
				ValueType:   property.ValueTypeScalar,
				DisplayName: "Admin UI MOTD",
			},
		},
		{
			ObjectMeta: v12.ObjectMeta{
				Name:      "motd-ui",
				Namespace: util.GetReleaseNamespace(),
				Labels: map[string]string{
					labels.SettingScope: "public",
				},
			},
			Value: "",
			Property: property.Property{
				DataType:    property.DataTypeString,
				ValueType:   property.ValueTypeScalar,
				DisplayName: "User UI MOTD",
			},
		},
		{
			ObjectMeta: v12.ObjectMeta{
				Name:      "registration-disabled",
				Namespace: util.GetReleaseNamespace(),
				Labels: map[string]string{
					labels.SettingScope: "public",
				},
			},
			Value: "false",
			Property: property.Property{
				DataType:    property.DataTypeBoolean,
				ValueType:   property.ValueTypeScalar,
				DisplayName: "Registration Disabled",
			},
		},
	}
}
