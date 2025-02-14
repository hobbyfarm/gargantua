package accesscode

import (
	"context"
	"fmt"
	"github.com/hobbyfarm/gargantua/v4/pkg/apis/hobbyfarm.io/v4alpha1"
	labels2 "github.com/hobbyfarm/gargantua/v4/pkg/labels"
	"github.com/hobbyfarm/gargantua/v4/pkg/uid"
	"log/slog"
	client2 "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

func (acc *accessCodeController) ReconcileRole(ctx context.Context, request reconcile.Request) (reconcile.Result, error) {
	roleList := &v4alpha1.RoleList{}
	if err := acc.kclient.List(ctx, roleList, client2.MatchingLabels{
		labels2.AccessCodeLabel: request.Name,
	}); err != nil {
		return reconcile.Result{}, err
	}

	ac := &v4alpha1.AccessCode{}
	if err := acc.kclient.Get(ctx, request.NamespacedName, ac); err != nil {
		return reconcile.Result{}, client2.IgnoreNotFound(err)
	}

	// Because mink adds "-p" to end of UIDs for some reason
	// We need to remove it because it breaks references to other objects
	// Especially when using owner refs
	ac.UID = uid.RemoveUIDPublic(ac.UID)

	var requeue bool

	if len(roleList.Items) == 0 {
		err := acc.createRole(ctx, request, ac)
		if err != nil {
			return reconcile.Result{}, err
		}

		requeue = true
	}

	if len(roleList.Items) == 1 {
		var role = &roleList.Items[0]

		acc.setRules(ac, role)

		if err := controllerutil.SetOwnerReference(ac, role, acc.scheme); err != nil {
			slog.Error("error setting owner reference for role", "error", err.Error())
			return reconcile.Result{}, err
		}

		if err := acc.kclient.Update(ctx, role); err != nil {
			return reconcile.Result{}, err
		}
	}

	if len(roleList.Items) > 1 {
		// there should not be more than one role for a code
		slog.Error("> 1 role exists for accesscode", "accesscode", request.Name)
		return reconcile.Result{}, fmt.Errorf("> 1 role exists for accesscode")
	}

	return reconcile.Result{
		Requeue: requeue,
	}, nil
}

func (acc *accessCodeController) createRole(ctx context.Context, request reconcile.Request, accessCode *v4alpha1.AccessCode) error {
	var role = &v4alpha1.Role{}
	slog.Debug("role does not exist for accesscode, creating it", "accesscode", request.Name)
	role.GenerateName = "coderole-"
	role.Labels = map[string]string{
		labels2.AccessCodeLabel: request.Name,
	}

	if err := acc.kclient.Create(ctx, role); err != nil {
		slog.Error("error creating role for accesscode", "error", err.Error(), "accesscode", request.Name)
		return err
	}

	slog.Debug("role created for accesscode", "accesscode", request.Name, "role", role.Name)
	return nil
}

func (acc *accessCodeController) setRules(accessCode *v4alpha1.AccessCode, role *v4alpha1.Role) {
	role.Rules = []v4alpha1.Rule{
		makeRule("scenarios", accessCode.Spec.Scenarios),
		makeRule("courses", accessCode.Spec.Courses),
		makeRule("scheduledEvents", accessCode.Spec.ScheduledEvents),
		makeRule("machineSets", accessCode.Spec.MachineSets),
	}
}

func makeRule(resources string, resourceNames []string) v4alpha1.Rule {
	return v4alpha1.Rule{
		APIGroups:     []string{"hobbyfarm.io"},
		Resources:     []string{resources},
		ResourceNames: resourceNames,
	}
}
