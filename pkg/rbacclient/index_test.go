package rbacclient

import (
	"fmt"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/tools/cache"
	"sync"
	"testing"
)

func addInformerEventHandler(informer cache.SharedIndexInformer, notification chan metav1.Object) {
	informer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			if o, ok := obj.(metav1.Object); ok {
				fmt.Println("received object " + o.GetName())
				notification <- o
			}
		},
	})
}

func createIndex(kind string, namespace string,
	roleBindingInformer cache.SharedIndexInformer,
	clusterRoleBindingInformer cache.SharedIndexInformer,
	roleInformer cache.SharedIndexInformer,
	clusterRoleInformer cache.SharedIndexInformer) (*Index, error) {
	return NewIndex(kind, namespace, roleBindingInformer, clusterRoleBindingInformer, roleInformer, clusterRoleInformer)
}

func Test_CreateRoleAndBinding(t *testing.T) {
	client := fake.NewSimpleClientset()

	if err := SetupRole(client); err != nil {
		t.Error(err)
	}

	if err := SetupRoleBinding(client); err != nil {
		t.Error(err)
	}
}

func Test_CreateIndex(t *testing.T) {
	client := fake.NewSimpleClientset()

	stopCh := make(chan struct{}, 0)
	defer close(stopCh)

	sif := informers.NewSharedInformerFactory(client, 0)

	_, err := createIndex(
		UserKind,
		FakeNamespace,
		sif.Rbac().V1().RoleBindings().Informer(),
		sif.Rbac().V1().ClusterRoleBindings().Informer(),
		sif.Rbac().V1().Roles().Informer(),
		sif.Rbac().V1().ClusterRoles().Informer())

	if err != nil {
		t.Error(err)
	}
}

func channeledSetup(setupFunc func(p kubernetes.Interface) error, client kubernetes.Interface, name string, notif chan metav1.Object) error {
	if err := setupFunc(client); err != nil {
		return err
	}

	obj := <-notif
	if obj.GetName() != name {
		return fmt.Errorf("channel returned object with mismatched name. got %s, expected %s", obj.GetName(),
			name)
	}

	return nil
}

func Test_UserAccessSet(t *testing.T) {
	client := fake.NewSimpleClientset()

	stopCh := make(chan struct{}, 0)
	defer close(stopCh)

	sif := informers.NewSharedInformerFactory(client, 0)

	// create the informers
	roleInformer := sif.Rbac().V1().Roles().Informer()
	roleBindingInformer := sif.Rbac().V1().RoleBindings().Informer()
	clusterRoleInformer := sif.Rbac().V1().ClusterRoles().Informer()
	clusterRoleBindingInformer := sif.Rbac().V1().ClusterRoleBindings().Informer()

	// we want to know when roles and rolebindings have been populated
	// in the index. so we add handlers that send added objects back in a channel
	roleChan := make(chan metav1.Object, 1)
	addInformerEventHandler(roleInformer, roleChan)

	roleBindingChan := make(chan metav1.Object, 1)
	addInformerEventHandler(roleBindingInformer, roleBindingChan)

	clusterRoleChan := make(chan metav1.Object, 1)
	addInformerEventHandler(clusterRoleInformer, clusterRoleChan)

	clusterRoleBindingChan := make(chan metav1.Object, 1)
	addInformerEventHandler(clusterRoleBindingInformer, clusterRoleBindingChan)

	userIndex, err := createIndex(UserKind, FakeNamespace,
		roleBindingInformer,
		clusterRoleBindingInformer,
		roleInformer,
		clusterRoleInformer)
	if err != nil {
		t.Error(err)
	}

	// start the informers
	// and wait for cache sync
	sif.Start(stopCh)
	cache.WaitForCacheSync(stopCh,
		roleInformer.HasSynced,
		roleBindingInformer.HasSynced,
		clusterRoleInformer.HasSynced,
		clusterRoleBindingInformer.HasSynced)

	wg := sync.WaitGroup{}

	// we need to wait for the role to be populated in the cache that the Index uses
	// this test will run too quickly if we don't wait, and the cache won't have a chance
	// to get the object before its done. therefore, we wait on the chan that informs us
	// that the object has been added.
	wg.Add(4)
	go func() {
		defer wg.Done()
		if err := channeledSetup(SetupRole, client, FakeRoleName, roleChan); err != nil {
			t.Error(err)
		}
	}()
	go func() {
		defer wg.Done()
		if err := channeledSetup(SetupRoleBinding, client, FakeRoleBindingName, roleBindingChan); err != nil {
			t.Error(err)
		}
	}()
	go func() {
		defer wg.Done()
		if err := channeledSetup(SetupClusterRole, client, FakeClusterRoleName, clusterRoleChan); err != nil {
			t.Error(err)
		}
	}()
	go func() {
		defer wg.Done()
		if err := channeledSetup(SetupClusterRoleBinding, client, FakeClusterRoleBindingName, clusterRoleBindingChan); err != nil {
			t.Error(err)
		}
	}()

	wg.Wait()

	accessSet, err := userIndex.GetAccessSet(FakeEmail)
	if err != nil {
		t.Error(err)
		return
	}

	t.Run("subject matches", func(t *testing.T) {
		if accessSet.Subject != FakeEmail {
			t.Errorf("access set subject %s did not match email %s", accessSet.Subject, FakeEmail)
		}
	})

	t.Run("access set not empty", func(t *testing.T) {
		if len(accessSet.Access) == 0 {
			t.Error("access set is empty")
		}
	})

	perms := RbacRequest().HobbyfarmPermission(RoleResource, RoleVerb).
		HobbyfarmPermission(ClusterRoleResource, ClusterRoleVerb).GetPermissions()

	for _, p := range perms {
		t.Run(fmt.Sprintf("asserting permission %s/%s/%s", p.GetAPIGroup(), p.GetResource(), p.GetVerb()), func(t *testing.T) {
			if !accessSet.Grants(p) {
				t.Error("permission check failed")
			}
		})
	}
}
