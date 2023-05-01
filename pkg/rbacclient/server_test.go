package rbacclient

import (
	"context"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/fake"
	"testing"
)

func Test_RbacClient(t *testing.T) {
	client := fake.NewSimpleClientset()

	for _, f := range []func(p kubernetes.Interface) error{
		SetupRole,
		SetupClusterRole,
		SetupRoleBinding,
		SetupClusterRoleBinding,
	} {
		if err := f(client); err != nil {
			t.Errorf("error calling setup func: %s", err.Error())
		}
	}

	sif := informers.NewSharedInformerFactory(client, 0)

	rbacclient, err := NewRbacClient(FakeNamespace, sif)
	if err != nil {
		t.Errorf("error setting up RbacClient: %s", err.Error())
	}

	sif.Start(context.TODO().Done())

	sif.WaitForCacheSync(context.TODO().Done())

	t.Run("test get accessset", func(t *testing.T) {
		as, err := rbacclient.GetAccessSet(FakeEmail)
		if err != nil {
			t.Errorf("error getting access set: %s", err.Error())
		}

		if len(as.Access) == 0 {
			t.Error("empty access set")
		}
	})

	t.Run("test role permissions allowed", func(t *testing.T) {
		perms := RbacRequest().HobbyfarmPermission(RoleResource, RoleVerb).GetPermissions()

		for _, p := range perms {
			allowed, err := rbacclient.Grants(FakeEmail, p)
			if err != nil {
				t.Errorf("error while calling rbacclient.Grants: %s", err.Error())
				return
			}

			if !allowed {
				t.Errorf("rbac permission %s/%s/%s not granted, should be",
					RoleAPIGroup, RoleResource, RoleVerb)
			}
		}
	})

	t.Run("test clusterrole permissions allowed", func(t *testing.T) {
		perms := RbacRequest().HobbyfarmPermission(ClusterRoleResource, ClusterRoleVerb).GetPermissions()

		for _, p := range perms {
			allowed, err := rbacclient.Grants(FakeEmail, p)
			if err != nil {
				t.Errorf("error while calling rbacclient.Grants: %s", err.Error())
				return
			}

			if !allowed {
				t.Errorf("rbac permission %s/%s/%s not granted, should be",
					RoleAPIGroup, ClusterRoleResource, ClusterRoleVerb)
			}
		}
	})

	t.Run("test courses not allowed", func(t *testing.T) {
		perms := RbacRequest().HobbyfarmPermission(NotAllowedResource, NotAllowedVerb).GetPermissions()

		for _, p := range perms {
			allowed, err := rbacclient.Grants(FakeEmail, p)
			if err != nil {
				t.Errorf("error while calling rbacclient.grants: %s", err.Error())
				return
			}

			if allowed {
				t.Errorf("rbac permission %s/%s/%s allowed, should NOT be",
					RoleAPIGroup, NotAllowedResource, NotAllowedVerb)
			}
		}
	})
}
