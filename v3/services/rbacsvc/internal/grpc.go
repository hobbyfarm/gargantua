package rbac

import (
	"github.com/hobbyfarm/gargantua/v3/pkg/util"
	rbacpb "github.com/hobbyfarm/gargantua/v3/protos/rbac"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	rbacv1 "k8s.io/client-go/kubernetes/typed/rbac/v1"
	listersv1 "k8s.io/client-go/listers/rbac/v1"
	"k8s.io/client-go/tools/cache"
)

type GrpcRbacServer struct {
	rbacpb.UnimplementedRbacSvcServer
	roleClient        rbacv1.RoleInterface
	roleBindingClient rbacv1.RoleBindingInterface
	userIndex         *Index
	groupIndex        *Index
	roleLister        listersv1.RoleLister
	roleSynced        cache.InformerSynced
	roleBindingLister listersv1.RoleBindingLister
	roleBindingSynced cache.InformerSynced
}

func NewGrpcRbacServer(kubeClientSet *kubernetes.Clientset, namespace string, kubeInformerFactory informers.SharedInformerFactory) (*GrpcRbacServer, error) {
	rClient := kubeClientSet.RbacV1().Roles(util.GetReleaseNamespace())
	rbClient := kubeClientSet.RbacV1().RoleBindings(util.GetReleaseNamespace())
	rbInformer := kubeInformerFactory.Rbac().V1().RoleBindings().Informer()
	crbInformer := kubeInformerFactory.Rbac().V1().ClusterRoleBindings().Informer()
	rInformer := kubeInformerFactory.Rbac().V1().Roles().Informer()
	crInformer := kubeInformerFactory.Rbac().V1().ClusterRoles().Informer()
	rLister := kubeInformerFactory.Rbac().V1().Roles().Lister()
	rbLister := kubeInformerFactory.Rbac().V1().RoleBindings().Lister()
	rSynced := rInformer.HasSynced
	rbSynced := rbInformer.HasSynced
	userIndex, err := NewIndex("User", namespace, rbInformer, crbInformer, rInformer, crInformer)
	if err != nil {
		return nil, err
	}

	groupIndex, err := NewIndex("Group", namespace, rbInformer, crbInformer, rInformer, crInformer)
	if err != nil {
		return nil, err
	}

	return &GrpcRbacServer{
		roleClient:        rClient,
		roleBindingClient: rbClient,
		userIndex:         userIndex,
		groupIndex:        groupIndex,
		roleLister:        rLister,
		roleSynced:        rSynced,
		roleBindingLister: rbLister,
		roleBindingSynced: rbSynced,
	}, nil
}
