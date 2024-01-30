package settingservice

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/golang/glog"
	"github.com/gorilla/mux"
	"github.com/hobbyfarm/gargantua/v3/pkg/labels"
	"github.com/hobbyfarm/gargantua/v3/pkg/property"
	"github.com/hobbyfarm/gargantua/v3/pkg/rbac"
	settingUtil "github.com/hobbyfarm/gargantua/v3/pkg/setting"
	"github.com/hobbyfarm/gargantua/v3/pkg/util"
	"github.com/hobbyfarm/gargantua/v3/protos/general"
	settingProto "github.com/hobbyfarm/gargantua/v3/protos/setting"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const (
	resourcePlural      = rbac.ResourcePluralSettings
	scopeResourcePlural = rbac.ResourcePluralScopes
)

type PreparedSetting struct {
	Name string `json:"name"`
	*settingProto.Property
	DataType  property.DataType  `json:"dataType"`
	ValueType property.ValueType `json:"valueType"`
	Value     any                `json:"value"`
	Scope     string             `json:"scope"`
	Group     string             `json:"group"`
	Weight    int64              `json:"weight"`
}

func (s SettingServer) ListFunc(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)

	scope, ok := vars["scope"]
	if !ok {
		util.ReturnHTTPMessage(w, r, 404, "notfound", "scope not found")
		return
	}

	// public scope settings are just that - public
	// so skip RBAC check for those
	if scope != "public" {
		resource := resourcePlural + "/" + scope
		user, err := rbac.AuthenticateRequest(r, s.authnClient)
		if err != nil {
			util.ReturnHTTPMessage(w, r, 401, "unauthorized", "authentication failed")
			return
		}

		impersonatedUserId := user.GetId()
		authrResponse, err := rbac.AuthorizeSimple(r, s.authrClient, impersonatedUserId, rbac.HobbyfarmPermission(resource, rbac.VerbList))
		if err != nil || !authrResponse.Success {
			util.ReturnHTTPMessage(w, r, 403, "forbidden", "no access to list settings")
			return
		}
	}

	labelSelector := fmt.Sprintf("%s=%s", labels.SettingScope, scope)
	kSettings, err := s.internalSettingServer.ListSettings(r.Context(), &general.ListOptions{LabelSelector: labelSelector})
	if err != nil {
		glog.Errorf("error listing settings: %s", err.Error())
		util.ReturnHTTPMessage(w, r, 500, "internalerror", "error listing settings")
		return
	}

	var settings []PreparedSetting
	for _, ks := range kSettings.GetSettings() {
		val, err := settingUtil.FromJSON(ks.GetProperty(), ks.GetValue())
		if err != nil {
			glog.Errorf("error encoding setting value for api: %s", err.Error())
			util.ReturnHTTPMessage(w, r, 500, "internalerror", "error encoding setting as json")
			return
		}

		stringifiedDataType := settingUtil.DataTypeMappingToHfTypes[ks.GetProperty().GetDataType()]
		stringifiedValueType := settingUtil.ValueTypeMappingToHfTypes[ks.GetProperty().GetValueType()]
		settings = append(settings, PreparedSetting{Name: ks.GetName(), Property: ks.GetProperty(),
			DataType: stringifiedDataType, ValueType: stringifiedValueType, Value: val, Scope: ks.GetScope(), Group: ks.GetGroup(), Weight: ks.GetWeight()})
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

func (s SettingServer) update(w http.ResponseWriter, r *http.Request, updatedSetting PreparedSetting) bool {
	setting, err := s.internalSettingServer.GetSetting(r.Context(), &general.ResourceId{Id: updatedSetting.Name})
	if err != nil {
		if s, ok := status.FromError(err); ok {
			if s.Code() == codes.NotFound {
				util.ReturnHTTPMessage(w, r, 404, "error", "setting not found")
				return false
			}
			util.ReturnHTTPMessage(w, r, 500, "internalerror", "error retrieving setting for update")
			return false
		}
		util.ReturnHTTPMessage(w, r, 500, "internalerror", "error retrieving setting for update")
		return false
	}
	var resource = resourcePlural + "/" + setting.GetScope()
	user, err := rbac.AuthenticateRequest(r, s.authnClient)
	if err != nil {
		util.ReturnHTTPMessage(w, r, 401, "unauthorized", "authentication failed")
		return false
	}

	impersonatedUserId := user.GetId()
	authrResponse, err := rbac.AuthorizeSimple(r, s.authrClient, impersonatedUserId, rbac.HobbyfarmPermission(resource, rbac.VerbUpdate))
	if err != nil || !authrResponse.Success {
		util.ReturnHTTPMessage(w, r, 401, "forbidden", "no access to update setting")
		return false
	}

	val, err := json.Marshal(updatedSetting.Value)

	// json marshalled strings have quotes before & after, we don't need or want that
	if setting.Property.GetDataType() == settingProto.DataType_DATA_TYPE_STRING && setting.Property.GetValueType() == settingProto.ValueType_VALUE_TYPE_SCALAR {
		val = []byte(strings.Replace(string(val), "\"", "", 2))
	}

	if err != nil {
		glog.Errorf("error marshalling and setting value: %s", err.Error())
		util.ReturnHTTPMessage(w, r, 500, "internalerror", "error updating setting")
		return false
	}

	setting.Value = string(val)

	_, err = s.internalSettingServer.UpdateSetting(r.Context(), setting)
	if err != nil {
		if s, ok := status.FromError(err); ok {
			if s.Code() == codes.InvalidArgument {
				util.ReturnHTTPMessage(w, r, 400, "error", s.Message())
				return false
			}
			util.ReturnHTTPMessage(w, r, 500, "internalerror", s.Message())
			return false
		}
		util.ReturnHTTPMessage(w, r, 500, "internalerror", "error updating setting")
		return false
	}
	return true
}

type PreparedScope struct {
	Name        string `json:"name"`
	DisplayName string `json:"displayName"`
}

func (s SettingServer) ListScopeFunc(w http.ResponseWriter, r *http.Request) {
	user, err := rbac.AuthenticateRequest(r, s.authnClient)
	if err != nil {
		util.ReturnHTTPMessage(w, r, 401, "unauthorized", "authentication failed")
		return
	}

	impersonatedUserId := user.GetId()
	authrResponse, err := rbac.AuthorizeSimple(r, s.authrClient, impersonatedUserId, rbac.HobbyfarmPermission(scopeResourcePlural, rbac.VerbList))
	if err != nil || !authrResponse.Success {
		util.ReturnHTTPMessage(w, r, 401, "forbidden", "no access to list scopes")
		return
	}

	scopes, err := s.internalSettingServer.ListScopes(r.Context(), &general.ListOptions{})
	if err != nil {
		util.ReturnHTTPMessage(w, r, http.StatusInternalServerError, "internalerror", "error listing scopes")
		glog.Errorf("error while listing scopes: %s", err.Error())
		return
	}

	encodedScopes, err := json.Marshal(scopes.Scopes)
	if err != nil {
		glog.Errorf("error marshalling prepared scopes: %s", err.Error())
		util.ReturnHTTPMessage(w, r, 500, "internalerror", "error listing scopes")
		return
	}

	util.ReturnHTTPContent(w, r, 200, "success", encodedScopes)

	glog.V(8).Info("listed scopes")
}
