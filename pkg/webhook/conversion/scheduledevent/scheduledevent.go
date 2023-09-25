package scheduledevent

import (
	v2 "github.com/hobbyfarm/gargantua/pkg/apis/hobbyfarm.io/v2"
	"github.com/hobbyfarm/gargantua/pkg/webhook/conversion"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

func Init() {
	conversion.RegisterConverter(schema.GroupKind{
		Group: "hobbyfarm.io",
		Kind:  "scheduledevents",
	}, convert)
}

func convert(Object *unstructured.Unstructured, toVersion string) (*unstructured.Unstructured, metav1.Status) {
	convertedObject := Object.DeepCopy()
	fromVersion := Object.GetAPIVersion()

	if toVersion == fromVersion {
		return nil, conversion.StatusFailureWithMessage("cannot convert from/to same version")
	}

	switch Object.GetAPIVersion() {
	case "hobbyfarm.io/v2":
		switch toVersion {
		case "hobbyfarm.io/v1":
			if _, ok := convertedObject.Object["shared_vms"]; ok {
				delete(convertedObject.Object, "shared_vms")
			}
		default:
			return nil, conversion.StatusFailureWithMessage("unexpected version %v for conversion", toVersion)
		}
	case "hobbyfarm.io/v1":
		switch toVersion {
		case "hobbyfarm.io/v2":
			var sharedVMs []v2.SharedVirtualMachine
			sharedVMs = make([]v2.SharedVirtualMachine, 0)
			convertedObject.Object["shared_vms"] = sharedVMs
		default:
			return nil, conversion.StatusFailureWithMessage("unexpected version %v for conversion", toVersion)
		}
	}

	return convertedObject, metav1.Status{Status: metav1.StatusSuccess}
}