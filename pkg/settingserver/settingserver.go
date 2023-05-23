package settingserver

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/golang/glog"
	"github.com/gorilla/mux"
	"github.com/hobbyfarm/gargantua/pkg/authclient"
	hfClientset "github.com/hobbyfarm/gargantua/pkg/client/clientset/versioned"
	labels "github.com/hobbyfarm/gargantua/pkg/labels"
	"github.com/hobbyfarm/gargantua/pkg/property"
	"github.com/hobbyfarm/gargantua/pkg/rbacclient"
	"github.com/hobbyfarm/gargantua/pkg/util"
	"io"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"net/http"
	"strconv"
)

const (
	resourcePlural = "settings"
)

type SettingServer struct {
	ctx         context.Context
	auth        *authclient.AuthClient
	hfClientSet hfClientset.Interface
}

type PreparedSetting struct {
	Name string `json:"name"`
	property.Property
	Value  string `json:"value"`
	Scope  string `json:"scope"`
	Weight int    `json:"weight"`
}

func NewSettingServer(clientset hfClientset.Interface, authClient *authclient.AuthClient, ctx context.Context) (*SettingServer, error) {
	setting := SettingServer{}

	setting.ctx = ctx
	setting.hfClientSet = clientset
	setting.auth = authClient

	return &setting, nil
}

func (s SettingServer) SetupRoutes(r *mux.Router) {
	r.HandleFunc("/setting/list/{scope}", s.ListFunc)
	r.HandleFunc("/setting/update/{setting_id}", s.UpdateFunc)
}

func (s SettingServer) ListFunc(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)

	scope, ok := vars["scope"]
	if !ok {
		util.ReturnHTTPMessage(w, r, 404, "notfound", "scope not found")
		return
	}

	resource := resourcePlural + "/" + scope
	_, err := s.auth.AuthGrant(rbacclient.RbacRequest().HobbyfarmPermission(resource, rbacclient.VerbList), w, r)
	if err != nil {
		util.ReturnHTTPMessage(w, r, 403, "forbidden", "no access to list settings")
		return
	}

	kSettings, err := s.hfClientSet.HobbyfarmV1().Settings(util.GetReleaseNamespace()).List(s.ctx, metav1.ListOptions{
		LabelSelector: fmt.Sprintf("%s=%s", "hobbyfarm.io/setting-scope", scope),
	})
	if err != nil {
		glog.Errorf("error listing settings: %s", err.Error())
		util.ReturnHTTPMessage(w, r, 500, "internalerror", "error listing settings")
		return
	}

	var settings []PreparedSetting
	for _, ks := range kSettings.Items {
		scope, _ := ks.Labels[labels.SettingScope]
		weight, _ := ks.Labels[labels.SettingWeight]
		iweight, _ := strconv.Atoi(weight)

		settings = append(settings, PreparedSetting{Name: ks.Name, Property: ks.Property, Value: ks.Value, Scope: scope, Weight: iweight})
	}

	encodedSettings, err := json.Marshal(settings)
	if err != nil {
		glog.Errorf("error marshalling prepared settings: %s", err.Error())
		util.ReturnHTTPMessage(w, r, 500, "internalerror", "error listing settings")
		return
	}

	util.ReturnHTTPContent(w, r, 200, "success", encodedSettings)

	glog.V(8).Infof("listed settings")
}

func (s SettingServer) UpdateFunc(w http.ResponseWriter, r *http.Request) {
	_, err := s.auth.AuthGrant(rbacclient.RbacRequest().HobbyfarmPermission(resourcePlural, rbacclient.VerbUpdate), w, r)
	if err != nil {
		util.ReturnHTTPMessage(w, r, 403, "forbidden", "no access to update settings")
		return
	}

	data, err := io.ReadAll(r.Body)
	if err != nil {
		glog.Errorf("error reading request body: %s", err.Error())
		util.ReturnHTTPMessage(w, r, 500, "internalerror", "error handling request body")
		return
	}

	setting := PreparedSetting{}
	err = json.Unmarshal(data, &setting)
	if err != nil {
		glog.Errorf("error unmarshalling json: %s", err.Error())
		util.ReturnHTTPMessage(w, r, 500, "internalerror", "error unmarshalling json body")
		return
	}

	// given the setting name, fetch it and update it
	kSetting, err := s.hfClientSet.HobbyfarmV1().Settings(util.GetReleaseNamespace()).Get(s.ctx, setting.Name, metav1.GetOptions{})
	if err != nil {
		glog.Errorf("error getting setting: %s", err.Error())
		util.ReturnHTTPMessage(w, r, 500, "internalerror", "error getting setting from database")
		return
	}

	kSetting.Value = setting.Value

	_, err = s.hfClientSet.HobbyfarmV1().Settings(util.GetReleaseNamespace()).Update(s.ctx, kSetting, metav1.UpdateOptions{})
	if err != nil {
		glog.Errorf("error updating setting: %s", err.Error())
		util.ReturnHTTPMessage(w, r, 500, "internalerror", "error updating setting")
		return
	}

	util.ReturnHTTPMessage(w, r, 200, "updated", "")
	glog.V(8).Infof("updated setting %s", setting.Name)
}
