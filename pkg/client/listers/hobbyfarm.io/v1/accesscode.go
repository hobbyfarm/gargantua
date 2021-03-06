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

// AccessCodeLister helps list AccessCodes.
type AccessCodeLister interface {
	// List lists all AccessCodes in the indexer.
	List(selector labels.Selector) (ret []*v1.AccessCode, err error)
	// Get retrieves the AccessCode from the index for a given name.
	Get(name string) (*v1.AccessCode, error)
	AccessCodeListerExpansion
}

// accessCodeLister implements the AccessCodeLister interface.
type accessCodeLister struct {
	indexer cache.Indexer
}

// NewAccessCodeLister returns a new AccessCodeLister.
func NewAccessCodeLister(indexer cache.Indexer) AccessCodeLister {
	return &accessCodeLister{indexer: indexer}
}

// List lists all AccessCodes in the indexer.
func (s *accessCodeLister) List(selector labels.Selector) (ret []*v1.AccessCode, err error) {
	err = cache.ListAll(s.indexer, selector, func(m interface{}) {
		ret = append(ret, m.(*v1.AccessCode))
	})
	return ret, err
}

// Get retrieves the AccessCode from the index for a given name.
func (s *accessCodeLister) Get(name string) (*v1.AccessCode, error) {
	obj, exists, err := s.indexer.GetByKey(name)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, errors.NewNotFound(v1.Resource("accesscode"), name)
	}
	return obj.(*v1.AccessCode), nil
}
