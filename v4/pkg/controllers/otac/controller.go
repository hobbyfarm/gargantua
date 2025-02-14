package otac

import (
	"github.com/hobbyfarm/gargantua/v4/pkg/apis/hobbyfarm.io/v4alpha1"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

func New(mgr manager.Manager) error {
	osc := &otacSetScaleController{kclient: mgr.GetClient()}

	return builder.
		ControllerManagedBy(mgr).
		For(&v4alpha1.OneTimeAccessCodeSet{}).
		Owns(&v4alpha1.OneTimeAccessCode{}).
		Named("otacset-scale-controller").Complete(osc)
}
