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
	v1 "github.com/hobbyfarm/gargantua/pkg/apis/hobbyfarm.io/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/tools/cache"
)

// DynamicBindRequestLister helps list DynamicBindRequests.
// All objects returned here must be treated as read-only.
type DynamicBindRequestLister interface {
	// List lists all DynamicBindRequests in the indexer.
	// Objects returned here must be treated as read-only.
	List(selector labels.Selector) (ret []*v1.DynamicBindRequest, err error)
	// DynamicBindRequests returns an object that can list and get DynamicBindRequests.
	DynamicBindRequests(namespace string) DynamicBindRequestNamespaceLister
	DynamicBindRequestListerExpansion
}

// dynamicBindRequestLister implements the DynamicBindRequestLister interface.
type dynamicBindRequestLister struct {
	indexer cache.Indexer
}

// NewDynamicBindRequestLister returns a new DynamicBindRequestLister.
func NewDynamicBindRequestLister(indexer cache.Indexer) DynamicBindRequestLister {
	return &dynamicBindRequestLister{indexer: indexer}
}

// List lists all DynamicBindRequests in the indexer.
func (s *dynamicBindRequestLister) List(selector labels.Selector) (ret []*v1.DynamicBindRequest, err error) {
	err = cache.ListAll(s.indexer, selector, func(m interface{}) {
		ret = append(ret, m.(*v1.DynamicBindRequest))
	})
	return ret, err
}

// DynamicBindRequests returns an object that can list and get DynamicBindRequests.
func (s *dynamicBindRequestLister) DynamicBindRequests(namespace string) DynamicBindRequestNamespaceLister {
	return dynamicBindRequestNamespaceLister{indexer: s.indexer, namespace: namespace}
}

// DynamicBindRequestNamespaceLister helps list and get DynamicBindRequests.
// All objects returned here must be treated as read-only.
type DynamicBindRequestNamespaceLister interface {
	// List lists all DynamicBindRequests in the indexer for a given namespace.
	// Objects returned here must be treated as read-only.
	List(selector labels.Selector) (ret []*v1.DynamicBindRequest, err error)
	// Get retrieves the DynamicBindRequest from the indexer for a given namespace and name.
	// Objects returned here must be treated as read-only.
	Get(name string) (*v1.DynamicBindRequest, error)
	DynamicBindRequestNamespaceListerExpansion
}

// dynamicBindRequestNamespaceLister implements the DynamicBindRequestNamespaceLister
// interface.
type dynamicBindRequestNamespaceLister struct {
	indexer   cache.Indexer
	namespace string
}

// List lists all DynamicBindRequests in the indexer for a given namespace.
func (s dynamicBindRequestNamespaceLister) List(selector labels.Selector) (ret []*v1.DynamicBindRequest, err error) {
	err = cache.ListAllByNamespace(s.indexer, s.namespace, selector, func(m interface{}) {
		ret = append(ret, m.(*v1.DynamicBindRequest))
	})
	return ret, err
}

// Get retrieves the DynamicBindRequest from the indexer for a given namespace and name.
func (s dynamicBindRequestNamespaceLister) Get(name string) (*v1.DynamicBindRequest, error) {
	obj, exists, err := s.indexer.GetByKey(s.namespace + "/" + name)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, errors.NewNotFound(v1.Resource("dynamicbindrequest"), name)
	}
	return obj.(*v1.DynamicBindRequest), nil
}
