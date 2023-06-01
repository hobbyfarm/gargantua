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
	"k8s.io/apimachinery/pkg/api/errors"
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
	Value  any    `json:"value"`
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
	r.HandleFunc("/setting/list/{scope}", s.ListFunc).Methods(http.MethodGet)
	r.HandleFunc("/setting/update/{setting_id}", s.UpdateFunc).Methods(http.MethodPut)
	r.HandleFunc("/setting/updatecollection", s.UpdateCollection).Methods(http.MethodPut)
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

		val, err := ks.Property.FromJSON(ks.Value)
		if err != nil {
			glog.Errorf("error encoding setting value for api: %s", err.Error())
			util.ReturnHTTPMessage(w, r, 500, "internalerror", "error encoding setting as json")
			return
		}

		settings = append(settings, PreparedSetting{Name: ks.Name, Property: ks.Property, Value: val, Scope: scope, Weight: iweight})
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

	if ok := s.update(w, r, setting); !ok {
		return
	}

	util.ReturnHTTPMessage(w, r, 200, "updated", "")
	glog.V(8).Infof("updated setting %s", setting.Name)
}

func (s SettingServer) UpdateCollection(w http.ResponseWriter, r *http.Request) {
	data, err := io.ReadAll(r.Body)
	if err != nil {
		glog.Errorf("error reading request body: %s", err.Error())
		util.ReturnHTTPMessage(w, r, 500, "internalerror", "error handling request body")
		return
	}

	settings := []PreparedSetting{}
	err = json.Unmarshal(data, &settings)
	if err != nil {
		glog.Errorf("error unmarshalling json: %s", err.Error())
		util.ReturnHTTPMessage(w, r, 500, "internalerror", "error unmarshalling json body")
		return
	}

	// for each setting, fetch it and update it
	for _, set := range settings {
		if ok := s.update(w, r, set); !ok {
			return
		}
	}

	util.ReturnHTTPMessage(w, r, 200, "updated", "")
	glog.V(8).Info("updated settings")
}

func (s SettingServer) update(w http.ResponseWriter, r *http.Request, setting PreparedSetting) bool {
	kSetting, err := s.hfClientSet.HobbyfarmV1().Settings(util.GetReleaseNamespace()).Get(s.ctx, setting.Name, metav1.GetOptions{})
	if errors.IsNotFound(err) {
		util.ReturnHTTPMessage(w, r, 404, "notfound", "setting not found")
		return false
	}
	if err != nil {
		glog.Errorf("error getting setting from database: %s", err.Error())
		util.ReturnHTTPMessage(w, r, 500, "internalerror", "error updating setting")
		return false
	}

	// check if the user has permissions to do this action
	scope, ok := kSetting.Labels[labels.SettingScope]
	if !ok {
		glog.Errorf("setting %s does not have scope label", kSetting.Name)
		util.ReturnHTTPMessage(w, r, 500, "internalerror", "error updating setting")
		return false
	}

	var resource = resourcePlural + "/" + scope
	_, err = s.auth.AuthGrant(rbacclient.RbacRequest().HobbyfarmPermission(resource, rbacclient.VerbUpdate), w, r)
	if err != nil {
		util.ReturnHTTPMessage(w, r, 401, "forbidden", "no access to update setting")
		return false
	}

	kSetting = kSetting.DeepCopy()

	val, err := json.Marshal(setting.Value)
	if err != nil {
		glog.Errorf("error marshalling setting value: %s", err.Error())
		util.ReturnHTTPMessage(w, r, 500, "internalerror", "error updating setting")
		return false
	}

	// validate the value
	if err := kSetting.Validate(string(val)); err != nil {
		util.ReturnHTTPMessage(w, r, http.StatusBadRequest, "badrequest", err.Error())
		return false
	}

	kSetting.Value = string(val)

	_, err = s.hfClientSet.HobbyfarmV1().Settings(util.GetReleaseNamespace()).Update(s.ctx, kSetting, metav1.UpdateOptions{})
	if err != nil {
		glog.Errorf("error updating setting: %s", err.Error())
		util.ReturnHTTPMessage(w, r, 500, "internalerror", "error updating setting")
		return false
	}

	return true
}
