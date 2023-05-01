package rbacclient

import (
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/client-go/informers"
)

const (
	VerbList   = "list"
	VerbGet    = "get"
	VerbCreate = "create"
	VerbUpdate = "update"
	VerbDelete = "delete"
	VerbWatch  = "watch"
)

type Client struct {
	userIndex  *Index
	groupIndex *Index
}

func NewRbacClient(namespace string, kubeInformerFactory informers.SharedInformerFactory) (*Client, error) {
	rbInformer := kubeInformerFactory.Rbac().V1().RoleBindings().Informer()
	crbInformer := kubeInformerFactory.Rbac().V1().ClusterRoleBindings().Informer()
	rInformer := kubeInformerFactory.Rbac().V1().Roles().Informer()
	crInformer := kubeInformerFactory.Rbac().V1().ClusterRoles().Informer()

	userIndex, err := NewIndex("User", namespace, rbInformer, crbInformer, rInformer, crInformer)
	if err != nil {
		return nil, err
	}

	groupIndex, err := NewIndex("Group", namespace, rbInformer, crbInformer, rInformer, crInformer)
	if err != nil {
		return nil, err
	}

	return &Client{
		userIndex:  userIndex,
		groupIndex: groupIndex,
	}, nil
}

func (rs *Client) Grants(user string, permission Permission) (bool, error) {
	as, err := rs.userIndex.GetAccessSet(user)
	if err != nil {
		return false, err
	}

	return as.Grants(permission), nil
}

func (rs *Client) GetAccessSet(user string) (*AccessSet, error) {
	return rs.userIndex.GetAccessSet(user)
}

func (rs *Client) GetHobbyfarmRoleBindings(user string) ([]*rbacv1.RoleBinding, error) {
	return rs.userIndex.getRoleBindings(user)
}
