// Code generated by lister-gen. DO NOT EDIT.

package v1

import (
	v1 "github.com/hobbyfarm/gargantua/pkg/apis/hobbyfarm.io/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/tools/cache"
)

// VirtualMachineClaimLister helps list VirtualMachineClaims.
type VirtualMachineClaimLister interface {
	// List lists all VirtualMachineClaims in the indexer.
	List(selector labels.Selector) (ret []*v1.VirtualMachineClaim, err error)
	// Get retrieves the VirtualMachineClaim from the index for a given name.
	Get(name string) (*v1.VirtualMachineClaim, error)
	VirtualMachineClaimListerExpansion
}

// virtualMachineClaimLister implements the VirtualMachineClaimLister interface.
type virtualMachineClaimLister struct {
	indexer cache.Indexer
}

// NewVirtualMachineClaimLister returns a new VirtualMachineClaimLister.
func NewVirtualMachineClaimLister(indexer cache.Indexer) VirtualMachineClaimLister {
	return &virtualMachineClaimLister{indexer: indexer}
}

// List lists all VirtualMachineClaims in the indexer.
func (s *virtualMachineClaimLister) List(selector labels.Selector) (ret []*v1.VirtualMachineClaim, err error) {
	err = cache.ListAll(s.indexer, selector, func(m interface{}) {
		ret = append(ret, m.(*v1.VirtualMachineClaim))
	})
	return ret, err
}

// Get retrieves the VirtualMachineClaim from the index for a given name.
func (s *virtualMachineClaimLister) Get(name string) (*v1.VirtualMachineClaim, error) {
	obj, exists, err := s.indexer.GetByKey(name)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, errors.NewNotFound(v1.Resource("virtualmachineclaim"), name)
	}
	return obj.(*v1.VirtualMachineClaim), nil
}
