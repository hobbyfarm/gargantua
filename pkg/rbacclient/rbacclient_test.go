package rbacclient

import (
	"context"
	v1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

var (
	FakeEmail = "fake@fake.com"

	FakeRoleName               = "fake-role"
	FakeClusterRoleName        = "fake-clusterrole"
	FakeRoleBindingName        = "fake-rolebinding"
	FakeClusterRoleBindingName = "fake-clusterrolebinding"
	FakeNamespace              = "fake"

	RoleAPIGroup        = "hobbyfarm.io"
	RoleResource        = "scheduledevents"
	ClusterRoleResource = "scenarios"
	RoleVerb            = "list"
	ClusterRoleVerb     = "get"

	NotAllowedResource = "courses"
	NotAllowedVerb     = "update"

	UserKind        = "User"
	GroupKind       = "Group"
	RoleKind        = "Role"
	ClusterRoleKind = "ClusterRole"
)

func SetupRole(client kubernetes.Interface) error {
	// add a role
	role := v1.Role{
		ObjectMeta: metav1.ObjectMeta{
			Name: FakeRoleName,
		},
		Rules: []v1.PolicyRule{
			{
				APIGroups: []string{RoleAPIGroup},
				Resources: []string{RoleResource},
				Verbs:     []string{RoleVerb},
			},
		},
	}

	_, err := client.RbacV1().Roles(FakeNamespace).Create(context.TODO(), &role, metav1.CreateOptions{})
	return err
}

func SetupClusterRole(client kubernetes.Interface) error {
	clusterRole := v1.ClusterRole{
		ObjectMeta: metav1.ObjectMeta{
			Name: FakeClusterRoleName,
		},
		Rules: []v1.PolicyRule{
			{
				APIGroups: []string{RoleAPIGroup},
				Resources: []string{ClusterRoleResource},
				Verbs:     []string{ClusterRoleVerb},
			},
		},
	}

	_, err := client.RbacV1().ClusterRoles().Create(context.TODO(), &clusterRole, metav1.CreateOptions{})
	return err
}

func SetupRoleBinding(client kubernetes.Interface) error {
	roleBinding := v1.RoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name: FakeRoleBindingName,
		},
		Subjects: []v1.Subject{
			{
				Kind:      UserKind,
				APIGroup:  RbacGroup,
				Name:      FakeEmail,
				Namespace: FakeNamespace,
			},
		},
		RoleRef: v1.RoleRef{
			APIGroup: RbacGroup,
			Kind:     RoleKind,
			Name:     FakeRoleName,
		},
	}

	_, err := client.RbacV1().RoleBindings(FakeNamespace).Create(context.TODO(), &roleBinding, metav1.CreateOptions{})
	return err
}

func SetupClusterRoleBinding(client kubernetes.Interface) error {
	clusterRolebinding := v1.ClusterRoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name: FakeClusterRoleBindingName,
		},
		Subjects: []v1.Subject{
			{
				Kind:      UserKind,
				APIGroup:  RbacGroup,
				Name:      FakeEmail,
				Namespace: FakeNamespace,
			},
		},
		RoleRef: v1.RoleRef{
			APIGroup: RbacGroup,
			Kind:     ClusterRoleKind,
			Name:     FakeClusterRoleName,
		},
	}

	_, err := client.RbacV1().ClusterRoleBindings().Create(context.TODO(), &clusterRolebinding, metav1.CreateOptions{})
	return err
}
