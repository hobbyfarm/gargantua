package user

import (
	"context"
	"github.com/hobbyfarm/gargantua/v4/pkg/apis/hobbyfarm.io/v4alpha1"
	"github.com/hobbyfarm/gargantua/v4/pkg/factoryhelpers"
	"github.com/hobbyfarm/gargantua/v4/pkg/labels"
	"github.com/rancher/lasso/pkg/client"
	"github.com/rancher/lasso/pkg/controller"
	"k8s.io/client-go/tools/cache"
)

type userController struct {
	roleBindingClient  *client.Client
	roleBindingIndexer cache.Indexer
}

func RegisterHandlers(factory controller.SharedControllerFactory) error {
	roleBindingClient, err := factoryhelpers.ClientForObject(&v4alpha1.RoleBinding{}, factory)
	if err != nil {
		return err
	}

	rolebindingInformer, err := factory.SharedCacheFactory().ForObject(&v4alpha1.RoleBinding{})
	if err != nil {
		return err
	}

	if err := rolebindingInformer.AddIndexers(map[string]cache.IndexFunc{
		labels.RoleBindingByUserIndex:  indexRoleBindingByUser,
		labels.RoleBindingByRole:       indexRoleBindingByRole,
		labels.RoleBindingByAccessCode: indexRoleBindingByAccessCode,
	}); err != nil {
		return err
	}

	userFactory, err := factory.ForObject(&v4alpha1.User{})
	if err != nil {
		return err
	}

	uc := &userController{
		roleBindingClient:  roleBindingClient,
		roleBindingIndexer: rolebindingInformer.GetIndexer(),
	}

	userFactory.RegisterHandler(context.TODO(), "ensure-rolebindings", controller.SharedControllerHandlerFunc(
		uc.ensureRoleBindings))

	return nil
}

func indexRoleBindingByUser(obj interface{}) ([]string, error) {
	rb, ok := obj.(*v4alpha1.RoleBinding)
	if !ok {
		return []string{}, nil
	}

	return rb.Users, nil
}

func indexRoleBindingByRole(obj interface{}) ([]string, error) {
	rb, ok := obj.(*v4alpha1.RoleBinding)
	if !ok {
		return []string{}, nil
	}

	return []string{rb.Role}, nil
}

func indexRoleBindingByAccessCode(obj interface{}) ([]string, error) {
	rb, ok := obj.(*v4alpha1.RoleBinding)
	if !ok {
		return []string{}, nil
	}

	if v, ok := rb.Labels[labels.AccessCodeLabel]; ok {
		return []string{v}, nil
	} else {
		return []string{}, nil
	}
}
