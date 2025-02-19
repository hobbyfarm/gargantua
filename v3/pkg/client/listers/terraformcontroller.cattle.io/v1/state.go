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
	terraformcontrollercattleiov1 "github.com/hobbyfarm/gargantua/v3/pkg/apis/terraformcontroller.cattle.io/v1"
	labels "k8s.io/apimachinery/pkg/labels"
	listers "k8s.io/client-go/listers"
	cache "k8s.io/client-go/tools/cache"
)

// StateLister helps list States.
// All objects returned here must be treated as read-only.
type StateLister interface {
	// List lists all States in the indexer.
	// Objects returned here must be treated as read-only.
	List(selector labels.Selector) (ret []*terraformcontrollercattleiov1.State, err error)
	// States returns an object that can list and get States.
	States(namespace string) StateNamespaceLister
	StateListerExpansion
}

// stateLister implements the StateLister interface.
type stateLister struct {
	listers.ResourceIndexer[*terraformcontrollercattleiov1.State]
}

// NewStateLister returns a new StateLister.
func NewStateLister(indexer cache.Indexer) StateLister {
	return &stateLister{listers.New[*terraformcontrollercattleiov1.State](indexer, terraformcontrollercattleiov1.Resource("state"))}
}

// States returns an object that can list and get States.
func (s *stateLister) States(namespace string) StateNamespaceLister {
	return stateNamespaceLister{listers.NewNamespaced[*terraformcontrollercattleiov1.State](s.ResourceIndexer, namespace)}
}

// StateNamespaceLister helps list and get States.
// All objects returned here must be treated as read-only.
type StateNamespaceLister interface {
	// List lists all States in the indexer for a given namespace.
	// Objects returned here must be treated as read-only.
	List(selector labels.Selector) (ret []*terraformcontrollercattleiov1.State, err error)
	// Get retrieves the State from the indexer for a given namespace and name.
	// Objects returned here must be treated as read-only.
	Get(name string) (*terraformcontrollercattleiov1.State, error)
	StateNamespaceListerExpansion
}

// stateNamespaceLister implements the StateNamespaceLister
// interface.
type stateNamespaceLister struct {
	listers.ResourceIndexer[*terraformcontrollercattleiov1.State]
}
