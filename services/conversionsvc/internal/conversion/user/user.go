package user

import (
	"github.com/hobbyfarm/gargantua/v3/services/conversionsvc/internal/conversion"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

func Init() {
	conversion.RegisterConverter(schema.GroupKind{
		Group: "hobbyfarm.io",
		Kind:  "users",
	}, convert)
}

func convert(Object *unstructured.Unstructured, toVersion string) (*unstructured.Unstructured, metav1.Status) {
	convertedObject := Object.DeepCopy()
	fromVersion := Object.GetAPIVersion()

	if toVersion == fromVersion {
		return nil, conversion.StatusFailureWithMessage("cannot convert from/to same version")
	}

	switch Object.GetAPIVersion() {
	case "hobbyfarm.io/v1":
		switch toVersion {
		case "hobbyfarm.io/v2":
			if _, ok := convertedObject.Object["admin"]; ok {
				delete(convertedObject.Object, "admin")
			}
		default:
			return nil, conversion.StatusFailureWithMessage("unexpected version %v for conversion", toVersion)
		}
	case "hobbyfarm.io/v2":
		switch toVersion {
		case "hobbyfarm.io/v1":
			convertedObject.Object["admin"] = false
		default:
			return nil, conversion.StatusFailureWithMessage("unexpected version %v for conversion", toVersion)
		}
	}

	return convertedObject, metav1.Status{Status: metav1.StatusSuccess}
}
