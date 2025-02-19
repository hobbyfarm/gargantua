/*
Copyright The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

// Code generated by lister-gen. DO NOT EDIT.

package v1

import (
	hobbyfarmiov1 "github.com/hobbyfarm/gargantua/v3/pkg/apis/hobbyfarm.io/v1"
	labels "k8s.io/apimachinery/pkg/labels"
	listers "k8s.io/client-go/listers"
	cache "k8s.io/client-go/tools/cache"
)

// VirtualMachineClaimLister helps list VirtualMachineClaims.
// All objects returned here must be treated as read-only.
type VirtualMachineClaimLister interface {
	// List lists all VirtualMachineClaims in the indexer.
	// Objects returned here must be treated as read-only.
	List(selector labels.Selector) (ret []*hobbyfarmiov1.VirtualMachineClaim, err error)
	// VirtualMachineClaims returns an object that can list and get VirtualMachineClaims.
	VirtualMachineClaims(namespace string) VirtualMachineClaimNamespaceLister
	VirtualMachineClaimListerExpansion
}

// virtualMachineClaimLister implements the VirtualMachineClaimLister interface.
type virtualMachineClaimLister struct {
	listers.ResourceIndexer[*hobbyfarmiov1.VirtualMachineClaim]
}

// NewVirtualMachineClaimLister returns a new VirtualMachineClaimLister.
func NewVirtualMachineClaimLister(indexer cache.Indexer) VirtualMachineClaimLister {
	return &virtualMachineClaimLister{listers.New[*hobbyfarmiov1.VirtualMachineClaim](indexer, hobbyfarmiov1.Resource("virtualmachineclaim"))}
}

// VirtualMachineClaims returns an object that can list and get VirtualMachineClaims.
func (s *virtualMachineClaimLister) VirtualMachineClaims(namespace string) VirtualMachineClaimNamespaceLister {
	return virtualMachineClaimNamespaceLister{listers.NewNamespaced[*hobbyfarmiov1.VirtualMachineClaim](s.ResourceIndexer, namespace)}
}

// VirtualMachineClaimNamespaceLister helps list and get VirtualMachineClaims.
// All objects returned here must be treated as read-only.
type VirtualMachineClaimNamespaceLister interface {
	// List lists all VirtualMachineClaims in the indexer for a given namespace.
	// Objects returned here must be treated as read-only.
	List(selector labels.Selector) (ret []*hobbyfarmiov1.VirtualMachineClaim, err error)
	// Get retrieves the VirtualMachineClaim from the indexer for a given namespace and name.
	// Objects returned here must be treated as read-only.
	Get(name string) (*hobbyfarmiov1.VirtualMachineClaim, error)
	VirtualMachineClaimNamespaceListerExpansion
}

// virtualMachineClaimNamespaceLister implements the VirtualMachineClaimNamespaceLister
// interface.
type virtualMachineClaimNamespaceLister struct {
	listers.ResourceIndexer[*hobbyfarmiov1.VirtualMachineClaim]
}
