package rbac

import (
	rbacProto "github.com/hobbyfarm/gargantua/protos/rbac"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
)

type GrpcRbacServer struct {
	rbacProto.UnimplementedRbacSvcServer
	kubeClientSet *kubernetes.Clientset
	userIndex     *Index
	groupIndex    *Index
}

func NewGrpcRbacServer(kubeClientSet *kubernetes.Clientset, namespace string, kubeInformerFactory informers.SharedInformerFactory) (*GrpcRbacServer, error) {
	userIndex, err := NewIndex("User", namespace, kubeInformerFactory)
	if err != nil {
		return nil, err
	}

	groupIndex, err := NewIndex("Group", namespace, kubeInformerFactory)
	if err != nil {
		return nil, err
	}

	return &GrpcRbacServer{
		kubeClientSet: kubeClientSet,
		userIndex:     userIndex,
		groupIndex:    groupIndex,
	}, nil
}
