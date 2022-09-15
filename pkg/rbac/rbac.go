package rbac

import (
	"context"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/client-go/rest"
	wranglerRbac "github.com/rancher/wrangler/pkg/generated/controllers/rbac"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"github.com/hobbyfarm/gargantua/pkg/util"
)

type Role struct {
	r rbacv1.Role
}

const(
	rbacManagedLabel = "rbac.hobbyfarm.io/managed"
)

func List() []Role{
	return []Role{
		newRole("testRole", func(r Role) Role {
			return r.
				addRule([] string {"hobbyfarm.io"}, [] string {"*"}, [] string {"roles", "rolebindings"}).
			  	addRule([] string {"hobbyfarm.io"}, [] string {"*"}, [] string {"users", "virtualmachinesets"})
		}),
	}
}

func Create(ctx context.Context, cfg *rest.Config) error {
	factory, err := wranglerRbac.NewFactoryFromConfig(cfg)
	if err != nil {
		return err
	}

	roles := List()
	rf := factory.Rbac().V1().Role()
	for _, role := range roles {
		rf.Create(&role.r)
	}

	return nil
}

func (role Role) addRule(APIGroups []string, verbs []string, resources []string) Role {
	role.r.Rules = append(role.r.Rules, rbacv1.PolicyRule{
		Verbs: verbs,
		APIGroups: APIGroups,
		Resources: resources,
	})
	return role
}

func newRole(name string, customize func (Role) Role) Role {
	role := rbacv1.Role{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: util.GetReleaseNamespace(),
			Labels: map[string]string{
				rbacManagedLabel: "true",
			},
		},
		Rules: []rbacv1.PolicyRule{},
	}
	rObj := Role{role}
	if customize != nil {
		rObj = customize(rObj)
	}
	return rObj
}
