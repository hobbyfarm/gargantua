package setting

import (
	"context"

	"github.com/golang/glog"
	"github.com/hobbyfarm/gargantua/services/conversionsvc/internal/validation/response"
	v12 "github.com/hobbyfarm/gargantua/pkg/apis/hobbyfarm.io/v1"
	hfClientset "github.com/hobbyfarm/gargantua/pkg/client/clientset/versioned"
	"github.com/hobbyfarm/gargantua/pkg/labels"
	"github.com/hobbyfarm/gargantua/pkg/util"
	"k8s.io/apimachinery/pkg/api/errors"
	v13 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"

	"github.com/hobbyfarm/gargantua/services/conversionsvc/internal/validation/conversion"
	"github.com/hobbyfarm/gargantua/services/conversionsvc/internal/validation/deserialize"
	v1 "k8s.io/api/admission/v1"
	"k8s.io/api/admission/v1beta1"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

type Server struct {
	hfclient *hfClientset.Clientset
}

func New(hfclient *hfClientset.Clientset) *Server {
	s := &Server{
		hfclient: hfclient,
	}

	return s
}

func (s *Server) RegisterTypes() []runtime.Object {
	return []runtime.Object{&v12.Setting{}, &v12.SettingList{}}
}

func (s *Server) GVK() schema.GroupVersionKind {
	return schema.GroupVersionKind{
		Group:   v12.SchemeGroupVersion.Group,
		Version: v12.SchemeGroupVersion.Version,
		Kind:    "settings",
	}
}

func (s *Server) V1beta1Review(ctx context.Context, ar *v1beta1.AdmissionRequest) *v1beta1.AdmissionResponse {
	resp := s.V1Review(ctx, conversion.ConvertAdmissionRequestToV1(ar))
	return conversion.ConvertAdmissionResponseToV1beta1(resp)
}

func (s *Server) V1Review(ctx context.Context, ar *v1.AdmissionRequest) *v1.AdmissionResponse {
	resp := &v1.AdmissionResponse{}

	var newObj = &v12.Setting{}
	var oldObj = &v12.Setting{}
	var err error
	switch ar.Operation {
	case v1.Update:
		_, _, err = deserialize.Decode(ar.OldObject.Raw, nil, oldObj)
		if err != nil {
			glog.Errorf("error deserializing hobbyfarm.io/setting: %s", err.Error())
			return response.RespDenied("could not cast old object into hobbyfarm.io/setting")
		}
		fallthrough
	case v1.Create:
		fallthrough
	default:
		_, _, err = deserialize.Decode(ar.Object.Raw, nil, newObj)
		if err != nil {
			glog.Errorf("error deserializing hobbyfarm.io/setting: %s", err.Error())
			return response.RespDenied("could not cast new object into hobbyfarm.io/setting")
		}
	}

	if ar.Operation == v1.Update {
		if oldObj.DataType != newObj.DataType {
			return response.RespDenied("datatype field immutable")
		}

		if oldObj.ValueType != newObj.ValueType {
			return response.RespDenied("valuetype field immutable")
		}
	}

	// check if the scope label matches an existing scope
	if scope, ok := newObj.Labels[labels.SettingScope]; ok && scope != "" {
		scopes, err := s.hfclient.HobbyfarmV1().Scopes(util.GetReleaseNamespace()).List(ctx, v13.ListOptions{})
		if errors.IsNotFound(err) {
			return response.RespDenied("no matching v1.Scope found for labeled scope %s", scope)
		}

		if err != nil {
			return response.RespDenied("unable to retrieve scopes from kubernetes: %s", err.Error())
		}

		var ok = false
		for _, s := range scopes.Items {
			if scope == s.Name {
				ok = true
				break
			}
		}

		if !ok {
			return response.RespDenied("no matching v1.Scope found for labeled scope %s", scope)
		}
	}

	if err := newObj.Property.Validate(newObj.Value); err != nil {
		return response.RespDenied("invalid: %s", err.Error())
	}

	resp.Allowed = true
	return resp
}
