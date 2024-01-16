package scenario

import (
	v2 "github.com/hobbyfarm/gargantua/v3/pkg/apis/hobbyfarm.io/v2"
	"github.com/hobbyfarm/gargantua/services/conversionsvc/v3/internal/conversion"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

func Init() {
	conversion.RegisterConverter(schema.GroupKind{
		Group: "hobbyfarm.io",
		Kind:  "scenarios",
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
			if _, ok := convertedObject.Object["vm_tasks"]; ok {
				delete(convertedObject.Object, "vm_tasks")
			}
		default:
			return nil, conversion.StatusFailureWithMessage("unexpected version %v for conversion", toVersion)
		}
	case "hobbyfarm.io/v1":
		switch toVersion {
		case "hobbyfarm.io/v2":
			var vmTasks []v2.VirtualMachineTasks
			vmTasks = make([]v2.VirtualMachineTasks, 0)
			convertedObject.Object["vm_tasks"] = vmTasks
		default:
			return nil, conversion.StatusFailureWithMessage("unexpected version %v for conversion", toVersion)
		}
	}

	return convertedObject, metav1.Status{Status: metav1.StatusSuccess}
}