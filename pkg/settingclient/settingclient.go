package settingclient

import (
	"context"
	"github.com/golang/glog"
	v1 "github.com/hobbyfarm/gargantua/v3/pkg/apis/hobbyfarm.io/v1"
	hfClientset "github.com/hobbyfarm/gargantua/v3/pkg/client/clientset/versioned"
	"github.com/hobbyfarm/gargantua/v3/pkg/client/informers/externalversions"
	"github.com/hobbyfarm/gargantua/v3/pkg/util"
	v12 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"time"
)

var (
	settings = map[string]*v1.Setting{}
)

const (
	SettingRegistrationDisabled SettingName = "registration-disabled"
	SettingAdminUIMOTD          SettingName = "motd-admin-ui"
	SettingUIMOTD               SettingName = "motd-ui"
	ScheduledEventRetentionTime SettingName = "scheduledevent-retention-time"
)

type SettingName string

type SettingsHandlers struct {
}

func (s SettingsHandlers) OnAdd(obj any) {
	set := obj.(*v1.Setting)

	settings[set.Name] = set
}

func (s SettingsHandlers) OnUpdate(oldObj any, newObj any) {
	oldSet := oldObj.(*v1.Setting)
	newSet := newObj.(*v1.Setting)

	settings[oldSet.Name] = newSet
}

func (s SettingsHandlers) OnDelete(obj any) {
	set := obj.(*v1.Setting)

	delete(settings, set.Name)
}

func WatchSettings(ctx context.Context,
	client *hfClientset.Clientset,
	informer externalversions.SharedInformerFactory) error {

	// load settings
	settingList, err := client.HobbyfarmV1().Settings(util.GetReleaseNamespace()).List(ctx, v12.ListOptions{})
	if err != nil {
		return err
	}

	for _, set := range settingList.Items {
		settings[set.Name] = &set
	}

	informer.Hobbyfarm().V1().Settings().Informer().AddEventHandlerWithResyncPeriod(SettingsHandlers{}, 30*time.Minute)

	return nil
}

func GetSetting(name SettingName) any {
	var set, err = settings[string(name)].FromJSON(settings[string(name)].Value)
	if err != nil {
		glog.Errorf("error getting setting %s: %s", name, err.Error())
		return nil
	}

	return set
}
