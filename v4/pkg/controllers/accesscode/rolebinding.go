package accesscode

import (
	"context"
	"github.com/hobbyfarm/gargantua/v4/pkg/apis/hobbyfarm.io/v4alpha1"
	labels2 "github.com/hobbyfarm/gargantua/v4/pkg/labels"
	"github.com/hobbyfarm/gargantua/v4/pkg/uid"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"log/slog"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

func (acc *accessCodeController) ReconcileRoleBinding(ctx context.Context, request reconcile.Request) (reconcile.Result, error) {
	roleBindingList := &v4alpha1.RoleBindingList{}
	if err := acc.kclient.List(ctx, roleBindingList, client.MatchingLabels{
		labels2.CodeRoleBindingLabel: request.Name,
	}); err != nil {
		return reconcile.Result{}, err
	}

	ac := &v4alpha1.AccessCode{}
	if err := acc.kclient.Get(ctx, request.NamespacedName, ac); err != nil {
		return reconcile.Result{}, client.IgnoreNotFound(err)
	}

	// Remove to avoid issues with ownership
	ac.UID = uid.RemoveUIDPublic(ac.UID)

	var requeue = false
	if len(roleBindingList.Items) == 0 {
		if err := acc.createRoleBinding(ctx, ac); err != nil {
			return reconcile.Result{}, err
		}

		requeue = true
	}

	if len(roleBindingList.Items) == 1 {
		var rolebinding = &roleBindingList.Items[0]

		// set ownership, everything else (membership) is handled elsewhere
		if err := controllerutil.SetControllerReference(ac, rolebinding, acc.scheme); err != nil {
			return reconcile.Result{}, err
		}

		if err := acc.kclient.Update(ctx, rolebinding); err != nil {
			return reconcile.Result{}, err
		}
	}

	if len(roleBindingList.Items) > 1 {
		slog.Error("more than one rolebinding exists for accesscode", "accesscode", ac.Name)
	}

	return reconcile.Result{Requeue: requeue}, nil
}

func (acc *accessCodeController) createRoleBinding(ctx context.Context, ac *v4alpha1.AccessCode) error {
	slog.Debug("rolebinding does not exist for accesscode, creating it", "accesscode", ac.Name)
	rb := &v4alpha1.RoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "code-",
			Labels: map[string]string{
				labels2.CodeRoleBindingLabel: ac.Name,
			},
		},
	}

	if err := acc.kclient.Create(ctx, rb); err != nil {
		slog.Error("error creating rolebinding for accesscode", "error", err.Error(),
			"accesscode", ac.Name)
		return err
	}

	slog.Debug("created rolebinding for accesscode", "rolebinding", rb.Name,
		"accesscode", ac.Name)

	return nil
}
