package setting

import (
	"context"
	"fmt"
	"github.com/golang/glog"
	v12 "github.com/hobbyfarm/gargantua/pkg/apis/hobbyfarm.io/v1"
	"github.com/hobbyfarm/gargantua/pkg/webhook/validation/admitters"
	"github.com/hobbyfarm/gargantua/pkg/webhook/validation/conversion"
	"github.com/hobbyfarm/gargantua/pkg/webhook/validation/deserialize"
	v1 "k8s.io/api/admission/v1"
	"k8s.io/api/admission/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

func Register() (schema.GroupVersionKind, admitters.Admitters) {
	deserialize.RegisterScheme(v12.SchemeGroupVersion, &v12.Setting{}, &v12.SettingList{})

	return schema.GroupVersionKind{
			Group:   v12.SchemeGroupVersion.Group,
			Version: v12.SchemeGroupVersion.Version,
			Kind:    "settings",
		}, admitters.Admitters{
			V1beta1: V1beta1Review,
			V1:      V1Review,
		}
}

func V1beta1Review(ctx context.Context, ar *v1beta1.AdmissionRequest) *v1beta1.AdmissionResponse {
	resp := V1Review(ctx, conversion.ConvertAdmissionRequestToV1(ar))
	return conversion.ConvertAdmissionResponseToV1beta1(resp)
}

func V1Review(_ context.Context, ar *v1.AdmissionRequest) *v1.AdmissionResponse {
	resp := &v1.AdmissionResponse{}

	var newObj = &v12.Setting{}
	var oldObj = &v12.Setting{}
	var err error
	switch ar.Operation {
	case v1.Update:
		_, _, err = deserialize.Decode(ar.OldObject.Raw, nil, oldObj)
		if err != nil {
			glog.Errorf("error deserializing hobbyfarm.io/setting: %s", err.Error())
			return respDenied("could not cast old object into hobbyfarm.io/setting")
		}
		fallthrough
	case v1.Create:
		fallthrough
	default:
		_, _, err = deserialize.Decode(ar.Object.Raw, nil, newObj)
		if err != nil {
			glog.Errorf("error deserializing hobbyfarm.io/setting: %s", err.Error())
			return respDenied("could not cast new object into hobbyfarm.io/setting")
		}
	}

	if ar.Operation == v1.Update {
		if oldObj.DataType != newObj.DataType {
			return respDenied("datatype field immutable")
		}

		if oldObj.ValueType != newObj.ValueType {
			return respDenied("valuetype field immutable")
		}
	}

	if err := newObj.Property.Validate(newObj.Value); err != nil {
		return respDenied("invalid: %s", err.Error())
	}

	resp.Allowed = true
	return resp
}

func respDenied(msg string, fields ...interface{}) *v1.AdmissionResponse {
	return &v1.AdmissionResponse{
		Allowed: false,
		Result: &metav1.Status{
			Message: fmt.Sprintf(msg, fields...),
		},
	}
}
