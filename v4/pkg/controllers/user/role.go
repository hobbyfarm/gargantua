package user

import (
	"context"
	"fmt"
	"github.com/hobbyfarm/gargantua/v4/pkg/apis/hobbyfarm.io/v4alpha1"
	"github.com/hobbyfarm/gargantua/v4/pkg/labels"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

func (uc *userController) ensureRoleBindings(key string, obj runtime.Object) (runtime.Object, error) {
	user, ok := obj.(*v4alpha1.User)
	if !ok {
		return nil, fmt.Errorf("could not convert %s to User", key)
	}

	// deleting?
	if user.DeletionTimestamp != nil {
		// remove any corresponding rolebindings

		// get rolebindings for the user
		rolebindings, err := uc.roleBindingIndexer.ByIndex(labels.RoleBindingByUserIndex, key)
		if err != nil {
			return nil, err
		}

		for _, rb := range rolebindings {
			binding, ok := rb.(*v4alpha1.RoleBinding)
			if !ok {
				continue
			}

			principals := principalsFromRoleBinding(binding)

			if len(principals) == 0 || (len(principals) == 1 && principals[0] == key) {
				// deleting because its either a useless rolebinding
				// or its a single-principal rolebinding for the user that will no longer exist
				if err := uc.roleBindingClient.Delete(context.TODO(), "", binding.Name, v1.DeleteOptions{}); err != nil {
					return nil, err
				}
			}

			// find index of the user
			userIndex := indexOf(binding.Users, key)
			if userIndex > -1 {
				// remove the user from the rolebinding
				binding.Users = append(binding.Users[:userIndex], binding.Users[userIndex+1:]...)
				// update the rolebinding
				if err := uc.roleBindingClient.Update(context.TODO(), "", binding, binding, v1.UpdateOptions{}); err != nil {
					return nil, err
				}
			}
		}
	}

	// not deleting
	// the user has accesscodes assigned to them
	// make sure that they are in the appropriate rolebindings for those accesscodes

	// get the rolebindings for the accesscodes
	bindings := make([]any, 0)
	for _, ac := range user.Spec.AccessCodes {
		bs, err := uc.roleBindingIndexer.ByIndex(labels.RoleBindingByAccessCode, ac)
		if err != nil {
			continue
			// TODO - Log this
		}

		bindings = append(bindings, bs...)
	}

	// for each binding, make sure the user is listed
	for _, bRaw := range bindings {
		b, ok := bRaw.(*v4alpha1.RoleBinding)
		if !ok {
			continue
			// TODO - Log this
		}

		if indexOf(b.Users, key) == -1 {
			b.Users = append(b.Users, key)
			if err := uc.roleBindingClient.Update(context.TODO(), "", b, b, v1.UpdateOptions{}); err != nil {
				return nil, err
			}
		}
	}

	return user, nil
}

func principalsFromRoleBinding(binding *v4alpha1.RoleBinding) []string {
	var out = make([]string, 0)

	out = append(out, binding.Users...)
	out = append(out, binding.Groups...)

	return out
}

func indexOf(slice []string, val string) int {
	for i, v := range slice {
		if v == val {
			return i
		}
	}

	return -1
}
