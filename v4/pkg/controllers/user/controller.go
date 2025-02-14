package user

import (
	"context"
	"github.com/hobbyfarm/gargantua/v4/pkg/apis/hobbyfarm.io/v4alpha1"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	kclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

const usersField = "spec.users"

func New(mgr manager.Manager) error {
	uac := &userAccessCodeController{
		kclient: mgr.GetClient(),
	}

	if err := mgr.GetFieldIndexer().IndexField(
		context.Background(),
		&v4alpha1.RoleBinding{},
		usersField,
		indexRoleBindingsByUsers); err != nil {
		return err
	}

	return builder.
		ControllerManagedBy(mgr).
		Named("user-accesscode-controller").
		For(&v4alpha1.User{}).
		Complete(uac)
}

func indexRoleBindingsByUsers(object kclient.Object) []string {
	rb, ok := object.(*v4alpha1.RoleBinding)
	if !ok {
		return []string{}
	}

	return rb.Users
}
