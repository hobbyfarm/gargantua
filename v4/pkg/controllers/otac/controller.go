package otac

import (
	"github.com/hobbyfarm/gargantua/v4/pkg/apis/hobbyfarm.io/v4alpha1"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

func New(mgr manager.Manager) error {
	builder.
		ControllerManagedBy(mgr).
		For(&v4alpha1.OneTimeAccessCodeSet{}).
		Named("otacset-scale-controller")
}
