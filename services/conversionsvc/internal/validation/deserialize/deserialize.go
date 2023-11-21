package deserialize

import (
	v1 "k8s.io/api/admission/v1"
	"k8s.io/api/admission/v1beta1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/runtime/serializer"
)

var (
	runtimeScheme = runtime.NewScheme()
	codecs        = serializer.NewCodecFactory(runtimeScheme)
	deserializer  = codecs.UniversalDeserializer()
)

func init() {
	runtimeScheme.AddKnownTypes(v1.SchemeGroupVersion,
		&v1.AdmissionReview{})

	runtimeScheme.AddKnownTypes(v1beta1.SchemeGroupVersion,
		&v1beta1.AdmissionReview{})
}

func RegisterScheme(gv schema.GroupVersion, types ...runtime.Object) {
	runtimeScheme.AddKnownTypes(gv, types...)
}

func Decode(data []byte, defaults *schema.GroupVersionKind, into runtime.Object) (runtime.Object, *schema.GroupVersionKind, error) {
	return deserializer.Decode(data, defaults, into)
}
