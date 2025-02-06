package accesscode

import (
	"errors"
	"github.com/hobbyfarm/gargantua/v4/pkg/apis/hobbyfarm.io/v4alpha1"
	"github.com/hobbyfarm/gargantua/v4/pkg/controllers/helpers"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	client2 "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

type accessCodeController struct {
	kclient client2.Client
	scheme  *runtime.Scheme
}

func New(mgr manager.Manager) error {
	acc := &accessCodeController{
		kclient: mgr.GetClient(),
		scheme:  mgr.GetScheme(),
	}

	errs := make([]error, 0)

	if err := builder.
		ControllerManagedBy(mgr).
		Owns(&v4alpha1.Role{}, builder.MatchEveryOwner).
		Named("accesscode-role").
		For(&v4alpha1.AccessCode{}).Complete(helpers.ReconcileFunc(acc.ReconcileRole)); err != nil {
		errs = append(errs, err)
	}

	if err := builder.
		ControllerManagedBy(mgr).
		Owns(&v4alpha1.RoleBinding{}, builder.MatchEveryOwner).
		Named("accesscode-rolebinding").
		For(&v4alpha1.AccessCode{}).Complete(helpers.ReconcileFunc(acc.ReconcileRoleBinding)); err != nil {
		errs = append(errs, err)
	}

	return errors.Join(errs...)
}
