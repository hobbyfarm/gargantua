package rbacclient

import (
	"context"
	v1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/fake"
	"testing"
	"time"
)

var (
	fakeEmail           = "fake@fake.com"
	fakeRoleName        = "fake-role"
	fakeRoleBindingName = "fake-rolebinding"
	fakeNamespace       = "fake"

	roleAPIGroups = []string{"hobbyfarm.io"}
	roleResources = []string{"scheduledevents"}
	roleVerbs     = []string{"list"}

	userKind  = "User"
	groupKind = "Group"
	roleKind  = "Role"
)

func createIndex(client kubernetes.Interface, kind string) (*Index, error) {
	sif := informers.NewSharedInformerFactory(client, time.Second*10)

	return NewIndex(kind, fakeNamespace, sif)
}

func setupRole(client kubernetes.Interface) error {
	// add a role
	role := v1.Role{
		ObjectMeta: metav1.ObjectMeta{
			Name: fakeRoleName,
		},
		Rules: []v1.PolicyRule{
			{
				APIGroups: roleAPIGroups,
				Resources: roleResources,
				Verbs:     roleVerbs,
			},
		},
	}

	_, err := client.RbacV1().Roles(fakeNamespace).Create(context.TODO(), &role, metav1.CreateOptions{})
	return err
}

func setupRoleBinding(client kubernetes.Interface) error {
	roleBinding := v1.RoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name: fakeRoleBindingName,
		},
		Subjects: []v1.Subject{
			{
				Kind:      userKind,
				APIGroup:  RbacGroup,
				Name:      fakeEmail,
				Namespace: fakeNamespace,
			},
		},
		RoleRef: v1.RoleRef{
			APIGroup: RbacGroup,
			Kind:     roleKind,
			Name:     fakeRoleName,
		},
	}

	_, err := client.RbacV1().RoleBindings(fakeNamespace).Create(context.TODO(), &roleBinding, metav1.CreateOptions{})
	return err
}

func Test_CreateRoleAndBinding(t *testing.T) {
	client := fake.NewSimpleClientset()

	if err := setupRole(client); err != nil {
		t.Error(err)
	}

	if err := setupRoleBinding(client); err != nil {
		t.Error(err)
	}
}

func Test_CreateIndex(t *testing.T) {
	client := fake.NewSimpleClientset()

	_, err := createIndex(client, userKind)

	if err != nil {
		t.Error(err)
	}
}

func Test_UserAccessSet(t *testing.T) {
	client := fake.NewSimpleClientset()

	userIndex, err := createIndex(client, userKind)
	if err != nil {
		t.Error(err)
	}

	if err := setupRole(client); err != nil {
		t.Error(err)
	}

	if err := setupRoleBinding(client); err != nil {
		t.Error(err)
	}

	accessSet, err := userIndex.GetAccessSet(fakeEmail)
	if err != nil {
		t.Error(err)
	}

	t.Run("subject matches", func(t *testing.T) {
		if accessSet.Subject != fakeEmail {
			t.Errorf("access set subject %s did not match email %s", accessSet.Subject, fakeEmail)
		}
	})

	t.Run("access set not nil", func(t *testing.T) {
		if accessSet.Access != nil {
			t.Error("access set nil")
		}
	})

}
