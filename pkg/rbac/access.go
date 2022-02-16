package rbac

import (
	"fmt"
	rbacv1 "k8s.io/api/rbac/v1"
)

// map[AccessKey]bool determines permission
// when determining permissions, we need to query
// for both verb+group+resource as specified by the request, AND
// *+*+* in case this is allowed via the glob
// so what would be basically O(n^3) is really just 2 lookups per level.
// two verbs, two groups, two resources. (e.g. * & get, * & hobbyfarm.io, * & ScheduledEvent)


type AccessSet struct {
	Subject string `json:"subject"`

	// key is /apigroup/resource/verb
	Access map[string]bool `json:"access"`
}

func (as *AccessSet) Grants(request Request) bool {
	// key is /apigroup/resource/verb
	for _, a := range []string{request.GetAPIGroup(), All} {
		for _, r := range []string{request.GetResource(), All} {
			for _, v := range []string{request.GetVerb(), All} {
				if as.Access[fmt.Sprintf("/%s/%s/%s", a, r, v)] {
					return true
				}
			}
		}
	}

	return false
}

func (i *Index) GetAccessSet(subj string) (*AccessSet, error) {
	var as = &AccessSet{
		Subject: subj,
		Access:  map[string]bool{},
	}

	// get the rolebindings for the subject
	rb, err := i.getRoleBindings(subj)
	if err != nil {
		return nil, err
	}

	crb, err := i.getClusterRoleBindings(subj)
	if err != nil {
		return nil, err
	}

	// take the rolebindings and clusterrolebindings and derive their policy rules
	for _, roleBinding := range rb {
		rules, err := i.getRules(roleBinding.Namespace, roleBinding.RoleRef)
		if err != nil {
			return nil, err
		}

		i.addToAccessSet(as, roleBinding.Namespace, rules)
	}

	for _, clusterRoleBinding := range crb {
		rules, err := i.getRules("", clusterRoleBinding.RoleRef)
		if err != nil {
			return nil, err
		}

		i.addToAccessSet(as, "", rules)
	}

	return as, nil
}

func (i *Index) addToAccessSet(accessSet *AccessSet, namespace string, rules []rbacv1.PolicyRule) {
	// we only care about rules that are global, or apply to our namespace
	// any others can be discarded
	// this simplifies the frontend from having to worry about what namespace HF is installed into
	if namespace != "" && namespace != i.namespace {
		return // this ruleset does not apply to us
		// it is neither global nor in our installed namespace
	}
	for _, rule := range rules {
		// for each rule
		for _, apiGroup := range rule.APIGroups {
			// for each api group in the rule
			for _, resource := range rule.Resources {
				// for each resource in the rule
				for _, verb := range rule.Verbs {
					key := fmt.Sprintf("/%s/%s/%s", apiGroup, resource, verb)
					accessSet.Access[key] = true
				}
			}
		}
	}
}

/*
get policy rules associated with roleRef.
if roleRef is for a Role object, namespace is used to look up the object

policyrules are listing of policies a role provides, e.g. apiGroups: *, resources: *, verbs: get,put
*/
func (i *Index) getRules(namespace string, roleRef rbacv1.RoleRef) ([]rbacv1.PolicyRule, error) {
	switch roleRef.Kind {
	case "ClusterRole":
		obj, exists, err := i.clusterRoleIndexer.GetByKey(roleRef.Name)
		if err != nil {
			return nil, err
		}
		if !exists {
			return nil, fmt.Errorf("not found")
		}

		return obj.(*rbacv1.ClusterRole).Rules, nil
	case "Role":
		obj, exists, err := i.roleIndexer.GetByKey(namespace + "/" + roleRef.Name)
		if err != nil {
			return nil, err
		}
		if !exists {
			return nil, fmt.Errorf("not found")
		}

		return obj.(*rbacv1.Role).Rules, nil
	}

	return nil, nil
}

func (i *Index) getRoleBindings(subj string) ([]*rbacv1.RoleBinding, error) {
	obj, err := i.roleBindingIndexer.ByIndex(rbIndex+"-"+i.kind, subj)
	if err != nil {
		return nil, err
	}

	var roleBindings []*rbacv1.RoleBinding

	for _, v := range obj {
		rb, ok := v.(*rbacv1.RoleBinding)
		if ok {
			roleBindings = append(roleBindings, rb)
		}
	}

	return roleBindings, nil
}

func (i *Index) getClusterRoleBindings(subj string) ([]*rbacv1.ClusterRoleBinding, error) {
	obj, err := i.clusterRoleBindingIndexer.ByIndex(rbIndex+"-"+i.kind, subj)
	if err != nil {
		return nil, err
	}

	var clusterRoleBindings []*rbacv1.ClusterRoleBinding

	for _, v := range obj {
		crb, ok := v.(*rbacv1.ClusterRoleBinding)
		if ok {
			clusterRoleBindings = append(clusterRoleBindings, crb)
		}
	}

	return clusterRoleBindings, nil
}
