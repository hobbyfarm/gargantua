package authorization

import (
	"context"
	"github.com/hobbyfarm/gargantua/v4/pkg/apis/hobbyfarm.io/v4alpha1"
	"github.com/hobbyfarm/mink/pkg/strategy"
	"k8s.io/apiserver/pkg/authentication/user"
	"k8s.io/apiserver/pkg/authorization/authorizer"
	"k8s.io/apiserver/pkg/storage"
	"regexp"
)

var _ authorizer.Authorizer = (*Authorizer)(nil)

type Authorizer struct {
	roleBindingLister strategy.Lister
	roleGetter        strategy.Getter
	unprotectedPaths  []string
}

func NewAuthorizer(roleBindingLister strategy.Lister, roleGetter strategy.Getter, unprotectedPaths ...string) Authorizer {
	return Authorizer{
		roleBindingLister: roleBindingLister,
		roleGetter:        roleGetter,
		unprotectedPaths:  unprotectedPaths,
	}
}

func (az Authorizer) Authorize(ctx context.Context, a authorizer.Attributes) (authorized authorizer.Decision, reason string, err error) {
	// regex match against unprotected paths
	for _, v := range az.unprotectedPaths {
		if ok, err := regexp.MatchString(v, a.GetPath()); ok && err == nil {
			return authorizer.DecisionAllow, "", nil
		}
	}

	if az.CheckSuperuser(a.GetUser()) {
		return authorizer.DecisionAllow, "", nil
	}

	// first, get bindings for the user and its groups
	bindings, err := az.GetBindings(ctx, a.GetUser())
	if err != nil {
		return authorizer.DecisionDeny, "error looking up permissions information", nil
	}

	for _, b := range bindings {
		// get rules for each binding
		r, err := az.roleGetter.Get(ctx, "", b.Role)
		if err != nil {
			// TODO : Log this failure
			continue
		}

		role := r.(*v4alpha1.Role)

		for _, rule := range role.Rules {
			if rule.Matches(a) {
				return authorizer.DecisionAllow, "", nil
			}
		}
	}

	return authorizer.DecisionDeny, "denied", nil
}

func (az Authorizer) GetBindings(ctx context.Context, u user.Info) (result []v4alpha1.RoleBinding, err error) {
	bindings, err := az.roleBindingLister.List(ctx, "", storage.ListOptions{})
	if err != nil {
		return nil, err
	}

	rbList := bindings.(*v4alpha1.RoleBindingList)
	for _, rb := range rbList.Items {
		if rb.MatchesUser(u) {
			result = append(result, rb)
		}
	}

	return
}

func (az Authorizer) CheckSuperuser(u user.Info) bool {
	for _, group := range u.GetGroups() {
		if group == user.SystemPrivilegedGroup {
			return true
		}
	}

	return false
}
