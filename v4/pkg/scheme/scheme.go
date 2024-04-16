package scheme

import (
	hfv1 "github.com/hobbyfarm/gargantua/v3/pkg/apis/hobbyfarm.io/v1"
	hfv2 "github.com/hobbyfarm/gargantua/v3/pkg/apis/hobbyfarm.io/v2"
	"github.com/hobbyfarm/gargantua/v4/pkg/apis/hobbyfarm.io/v4alpha1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
)

var (
	Scheme = runtime.NewScheme()

	Codec = serializer.NewCodecFactory(Scheme)

	ParameterCodec = runtime.NewParameterCodec(Scheme)
)

func AddToScheme(scheme *runtime.Scheme) error {
	metav1.AddToGroupVersion(scheme, v4alpha1.SchemeGroupVersion)

	if err := v4alpha1.AddToScheme(scheme); err != nil {
		return err
	}

	if err := v1.AddToScheme(scheme); err != nil {
		return err
	}

	if err := hfv1.AddToScheme(scheme); err != nil {
		return err
	}

	if err := hfv2.AddToScheme(scheme); err != nil {
		return err
	}

	return nil
}

func init() {
	utilruntime.Must(AddToScheme(Scheme))
}
