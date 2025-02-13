package user

import (
	"context"
	"github.com/hobbyfarm/gargantua/v3/pkg/labels"
	"github.com/hobbyfarm/gargantua/v4/pkg/apis/hobbyfarm.io/v4alpha1"
	labels3 "github.com/hobbyfarm/gargantua/v4/pkg/labels"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/fields"
	labels2 "k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/selection"
	"log/slog"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

/*
This controller is responsible for ensuring that a user is in the correct roles (via rolebindings)
for the accesscodes they have on their User object.
*/

type userAccessCodeController struct {
	kclient client.Client
}

func (cx *userAccessCodeController) Reconcile(ctx context.Context, request reconcile.Request) (reconcile.Result, error) {
	user := &v4alpha1.User{}
	if err := cx.kclient.Get(ctx, request.NamespacedName, user); err != nil {
		return reconcile.Result{}, client.IgnoreNotFound(err)
	}

	// Ensure that the user is in the right RoleBindings
	for idx, code := range user.Spec.AccessCodes {
		// get the code to verify that its active
		ac := &v4alpha1.AccessCode{}
		err := cx.kclient.Get(ctx, client.ObjectKey{Namespace: user.Namespace, Name: code}, ac)

		if err != nil {
			if errors.IsNotFound(err) {
				// this accesscode is invalid, remove it
				user.Spec.AccessCodes = append(user.Spec.AccessCodes[:idx], user.Spec.AccessCodes[idx+1:]...)

				if err := cx.kclient.Update(ctx, user); err != nil {
					slog.Error("error updating user to remove accesscode", "user", user.Name,
						"accesscode", code, "error", err.Error())
				}

				continue
			} else {
				return reconcile.Result{}, err // something went wrong
			}
		}

		if !ac.IsActive() {
			// accesscode is not active. leave it on the user, but don't add to rolebinding
			slog.Debug("inactive access code, not adding to rolebindings for user", "user", user.Name,
				"accesscode", ac.Name)
			continue
		}

		// get the rolebinding for this accesscode
		rblist := &v4alpha1.RoleBindingList{}
		if err := cx.kclient.List(ctx, rblist, client.MatchingLabels{
			labels.AccessCodeLabel: code,
		}); err != nil {
			return reconcile.Result{}, err
		}

		if len(rblist.Items) != 1 {
			slog.Error("invalid number of rolebindings found for accesscode", "accesscode",
				ac.Name, "count", len(rblist.Items))
			return reconcile.Result{
				Requeue: true,
			}, nil
		}

		rb := rblist.Items[0]

		if err := cx.ensureMembership(ctx, &rb, user.Name); err != nil {
			return reconcile.Result{}, err
		}
	}

	// Ensure the user isn't in any inappropriate RoleBindings (accesscodes removed, etc.)
	// to do this, we list all rolebindings that the user is a member of
	rblist, err := cx.getCodeRoleBindingsForUser(ctx, user.Name)
	if err != nil {
		return reconcile.Result{}, err
	}

	for idx, rb := range rblist.Items {
		// get the code for this rolebinding
		code, ok := rb.Labels[labels.AccessCodeLabel]
		if !ok {
			continue
		}

		if !hasMatch(code, user.Spec.AccessCodes) {
			// user should not be in this role, remove them
			// take rb
			roleBinding := rblist.Items[idx]
			// make an OG copy for patch
			ogRoleBinding := roleBinding.DeepCopy()

			// remove user
			roleBinding.Users = removeString(user.Name, roleBinding.Users)

			// update rolebinding
			if err := cx.kclient.Patch(ctx, &roleBinding, client.MergeFrom(ogRoleBinding)); err != nil {
				return reconcile.Result{}, err
			}
		}
	}

	return reconcile.Result{}, nil
}

func (cx *userAccessCodeController) ensureMembership(ctx context.Context, rb *v4alpha1.RoleBinding, user string) error {
	var ogRb = rb.DeepCopy()
	var ok = false
	for _, u := range rb.Users {
		if u == user {
			ok = true
			break
		}
	}

	if !ok {
		rb.Users = append(rb.Users, user)

		return cx.kclient.Patch(ctx, rb, client.MergeFrom(ogRb))
	}

	return nil
}

func (cx *userAccessCodeController) getCodeRoleBindingsForUser(ctx context.Context, user string) (*v4alpha1.RoleBindingList, error) {
	rblist := &v4alpha1.RoleBindingList{}

	// We only want to get roles that are access-code related
	requirement, err := labels2.NewRequirement(labels3.AccessCodeLabel, selection.Exists, []string{})
	if err != nil {
		return nil, err
	}

	sel := labels2.NewSelector()
	sel = sel.Add(*requirement)

	listOpts := client.ListOptions{
		LabelSelector: sel,
		FieldSelector: fields.OneTermEqualSelector(usersField, user), // select based on user index
	}

	if err := cx.kclient.List(ctx, rblist, &listOpts); err != nil {
		return nil, err
	}

	return rblist, nil
}

func hasMatch(needle string, haystack []string) bool {
	for _, h := range haystack {
		if h == needle {
			return true
		}
	}
	return false
}

func removeString(needle string, haystack []string) []string {
	for idx, elem := range haystack {
		if elem == needle {
			haystack = append(haystack[:idx], haystack[idx+1:]...)
		}
	}

	return haystack
}
