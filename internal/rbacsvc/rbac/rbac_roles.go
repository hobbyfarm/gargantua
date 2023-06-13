package rbacinstaller

import (
	"context"

	"github.com/hobbyfarm/gargantua/pkg/util"
	wranglerRbac "github.com/rancher/wrangler/pkg/generated/controllers/rbac"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/rest"
)

type Role struct {
	r rbacv1.Role
}

func list() []Role {
	return []Role{
		// RBAC Admin can edit roles and rolebindings rbac.authorization.k8s.io
		newRole("hf-admin", func(r Role) Role {
			return r.
				addRule([]string{"hobbyfarm.io"}, []string{"*"}, []string{"*"}).
				addRule([]string{"rbac.authorization.k8s.io"}, []string{"*"}, []string{"*"})
		}),
		// RBAC Admin can edit roles and rolebindings rbac.authorization.k8s.io
		newRole("rbac-admin", func(r Role) Role {
			return r.
				addRule([]string{"hobbyfarm.io"}, []string{"list", "get"}, []string{"users"}).
				addRule([]string{"rbac.authorization.k8s.io"}, []string{"*"}, []string{"roles", "rolebindings"})
		}),
		// Content Creator can create and edit scenarios and courses
		newRole("content-creator", func(r Role) Role {
			return r.
				addRule([]string{"hobbyfarm.io"}, []string{"*"}, []string{"scenarios", "courses"}).
				addRule([]string{"hobbyfarm.io"}, []string{"list", "get"}, []string{"virtualmachinetemplates"})
		}),
		// Infatructure Admin can edit environments and vmtemplates
		newRole("infrastructure-admin", func(r Role) Role {
			return r.
				addRule([]string{"hobbyfarm.io"}, []string{"*"}, []string{"environments", "virtualmachinetemplates"})
		}),
		// ScheduledEvent Creator can create and edit scheduled events + view dashboards
		newRole("scheduledevent-creator", func(r Role) Role {
			return r.
				addRule([]string{"hobbyfarm.io"}, []string{"*"}, []string{"scheduledevents", "accesscodes"}).
				addRule([]string{"hobbyfarm.io"}, []string{"list"}, []string{"environments"}).
				addRule([]string{"hobbyfarm.io"}, []string{"list", "get"}, []string{"scenarios", "courses", "virtualmachinetemplates", "virtualmachinesets", "users"}).
				addRule([]string{"hobbyfarm.io"}, []string{"list", "get", "watch"}, []string{"progresses", "virtualmachines", "virtualmachineclaims"}).
				addRule([]string{"hobbyfarm.io"}, []string{"update", "delete", "list", "get"}, []string{"sessions"})
		}),
		// ScheduledEvent Proctor is allowed to view scheduled events + dashboards
		newRole("scheduledevent-proctor", func(r Role) Role {
			return r.
				addRule([]string{"hobbyfarm.io"}, []string{"list", "get"}, []string{"scheduledevents", "accesscodes", "scenarios", "courses", "environments", "virtualmachinetemplates", "virtualmachinesets", "users"}).
				addRule([]string{"hobbyfarm.io"}, []string{"list"}, []string{"environments"}).
				addRule([]string{"hobbyfarm.io"}, []string{"list", "get", "watch"}, []string{"progresses", "virtualmachines", "virtualmachineclaims"}).
				addRule([]string{"hobbyfarm.io"}, []string{"update", "delete", "list", "get"}, []string{"sessions"})
		}),
		// User Manager can update and delete users
		newRole("user-manager", func(r Role) Role {
			return r.
				addRule([]string{"hobbyfarm.io"}, []string{"*"}, []string{"users"})
		}),
		// Read Only on users
		newRole("readonly-users", func(r Role) Role {
			return r.addRule([]string{"hobbyfarm.io"}, []string{"list", "get"}, []string{"users"})
		}),
	}
}

func Create(ctx context.Context, cfg *rest.Config) error {
	factory, err := wranglerRbac.NewFactoryFromConfig(cfg)
	if err != nil {
		return err
	}

	roles := list()
	rf := factory.Rbac().V1().Role()
	for _, role := range roles {
		rf.Create(&role.r)
	}

	return nil
}

func (role Role) addRule(APIGroups []string, verbs []string, resources []string) Role {
	role.r.Rules = append(role.r.Rules, rbacv1.PolicyRule{
		Verbs:     verbs,
		APIGroups: APIGroups,
		Resources: resources,
	})
	return role
}

func newRole(name string, customize func(Role) Role) Role {
	role := rbacv1.Role{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: util.GetReleaseNamespace(),
			Labels: map[string]string{
				util.RBACManagedLabel: "true",
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
