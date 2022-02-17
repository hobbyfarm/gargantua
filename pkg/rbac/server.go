package rbac

import (
	"k8s.io/client-go/informers"
)

const (
	VerbList = "list"
	VerbGet = "get"
	VerbCreate = "create"
	VerbUpdate = "update"
	VerbDelete = "delete"
	VerbWatch = "watch"
)

type Server struct {
	userIndex *Index
	groupIndex *Index
}

func NewRbacServer(namespace string, kubeInformerFactory informers.SharedInformerFactory) (*Server, error) {
	userIndex, err := NewIndex("User", namespace, kubeInformerFactory)
	if err != nil {
		return nil, err
	}

	groupIndex, err := NewIndex("Group", namespace, kubeInformerFactory)
	if err != nil {
		return nil, err
	}

	return &Server{
		userIndex: userIndex,
		groupIndex: groupIndex,
	}, nil
}

func (rs *Server) Grants(user string, permission Permission) (bool, error) {
	as, err := rs.userIndex.GetAccessSet(user)
	if err != nil {
		return false, err
	}

	return as.Grants(permission), nil
}

func (rs *Server) GetAccessSet(user string) (*AccessSet, error) {
	return rs.userIndex.GetAccessSet(user)
}

