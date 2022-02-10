package rbac

import (
	"fmt"
	v1 "k8s.io/api/rbac/v1"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/tools/cache"
)

const (
	rbIndex = "rbac.hobbyfarm.io/rb-index"
	RbacGroup = "rbac.authorization.k8s.io"
	All = "*"
)

var (
	KindUser = "User"
	KindGroup = "Group"
	KindServiceAccount = "ServiceAccount"
)

type Index struct {
	kind string

	roleBindingIndexer cache.Indexer
	clusterRoleBindingIndexer cache.Indexer

	roleIndexer cache.Indexer
	clusterRoleIndexer cache.Indexer
}

func NewIndex(kind string, informerFactory informers.SharedInformerFactory) (*Index, error) {
	i := &Index{
		kind: kind,
	}

	// build an informer for each of rolebinding and clusterrolebinding
	rbInformer := informerFactory.Rbac().V1().RoleBindings().Informer()
	crbInformer := informerFactory.Rbac().V1().ClusterRoleBindings().Informer()

	// add the indexers to a map...
	rbIndexers := map[string]cache.IndexFunc{rbIndex + "-" + kind: i.roleBindingSubjectIndexer}
	crbIndexers := map[string]cache.IndexFunc{rbIndex + "-" + kind: i.clusterRoleBindingSubjectIndexer}

	// ... then tell the informers to use those indexers
	if err := rbInformer.AddIndexers(rbIndexers); err != nil {
		return nil, err
	}

	if err := crbInformer.AddIndexers(crbIndexers); err != nil {
		return nil, err
	}

	// finally, generate the indexers and store in the index struct
	i.roleBindingIndexer = rbInformer.GetIndexer()
	i.clusterRoleBindingIndexer = crbInformer.GetIndexer()

	i.roleIndexer = informerFactory.Rbac().V1().Roles().Informer().GetIndexer()
	i.clusterRoleIndexer = informerFactory.Rbac().V1().ClusterRoleBindings().Informer().GetIndexer()

	return i, nil
}

/*
indexes RoleBindings in kubernetes
the index is based on the subject(s) of the rolebinding
if the index type (i.kind) is user, we index user subjects of rolebindings. if group, index groups.
for both types, we index serviceaccounts as well as they are special

this lets us perform quick lookups of the rolebindings for a subject
 */
func (i *Index) roleBindingSubjectIndexer(obj interface{}) (result []string, err error) {
	rb, ok := obj.(*v1.RoleBinding)
	if !ok {
		return
	}

	for _, subj := range rb.Subjects {
		if subj.APIGroup == RbacGroup && subj.Kind == i.kind {
			result = append(result, subj.Name)
		} else if subj.APIGroup == RbacGroup && i.kind == KindUser && subj.Kind == KindServiceAccount {
			result = append(result, fmt.Sprintf("serviceaccount:%s:%s", subj.Namespace, subj.Name))
		}
	}

	return
}

/*
indexes clusterrolebindings in kubernetes
the index is based on the subject(s) of the clusterrolebinding
if the index type (i.kind) is user, we index user subjects of clusterrolebindings. if group, index groups.
for both types, we index serviceaccounts as well as they are special

this lets us perform quick lookups of the clusterrolebindings for a subject
 */
func (i *Index) clusterRoleBindingSubjectIndexer(obj interface{}) (result []string, err error) {
	crb, ok := obj.(*v1.ClusterRoleBinding)
	if !ok {
		return
	}

	for _, subj := range crb.Subjects {
		if subj.APIGroup == RbacGroup && subj.Kind == i.kind {
			result = append(result, subj.Name)
		} else if subj.APIGroup == "" && i.kind == "User" && subj.Kind == "ServiceAccount" && subj.Namespace != "" && crb.RoleRef.Kind == "ClusterRole" {
			result = append(result, fmt.Sprintf("serviceaccount:%s:%s", subj.Namespace, subj.Name))
		}
	}

	return
}
