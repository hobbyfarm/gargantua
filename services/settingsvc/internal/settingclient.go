package settingservice

import (
	"context"
	"fmt"
	"time"

	"github.com/golang/glog"
	v1 "github.com/hobbyfarm/gargantua/v3/pkg/apis/hobbyfarm.io/v1"
	hfClientset "github.com/hobbyfarm/gargantua/v3/pkg/client/clientset/versioned"
	"github.com/hobbyfarm/gargantua/v3/pkg/client/informers/externalversions"
	settingUtil "github.com/hobbyfarm/gargantua/v3/pkg/setting"
	"github.com/hobbyfarm/gargantua/v3/pkg/util"
	v12 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var (
	settings = map[string]*v1.Setting{}
)

type SettingsHandlers struct {
}

func (s SettingsHandlers) OnAdd(obj any, isInInitialList bool) {
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
		s := set
		settings[set.Name] = &s
	}

	informer.Hobbyfarm().V1().Settings().Informer().AddEventHandlerWithResyncPeriod(SettingsHandlers{}, 30*time.Minute)

	return nil
}

func GetSetting(name settingUtil.SettingName) (*v1.Setting, error) {
	var set, ok = settings[string(name)]
	if !ok {
		glog.Errorf("error retrieving setting %s", name)
		return nil, fmt.Errorf("error retrieving setting %s", name)
	}

	return set, nil
}
