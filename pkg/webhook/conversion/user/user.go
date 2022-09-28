package user

import (
	"github.com/golang/glog"
	"github.com/hobbyfarm/gargantua/pkg/webhook/conversion"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

func init() {
	conversion.RegisterConverter("user", convert)
}

func convert(Object *unstructured.Unstructured, toVersion string) (*unstructured.Unstructured, metav1.Status) {
	glog.V(4).Infof("converting user crd to version " + toVersion)

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
