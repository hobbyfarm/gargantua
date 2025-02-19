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

// Code generated by client-gen. DO NOT EDIT.

package v1

import (
	context "context"

	hobbyfarmiov1 "github.com/hobbyfarm/gargantua/v3/pkg/apis/hobbyfarm.io/v1"
	scheme "github.com/hobbyfarm/gargantua/v3/pkg/client/clientset/versioned/scheme"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	types "k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"
	gentype "k8s.io/client-go/gentype"
)

// CostsGetter has a method to return a CostInterface.
// A group's client should implement this interface.
type CostsGetter interface {
	Costs(namespace string) CostInterface
}

// CostInterface has methods to work with Cost resources.
type CostInterface interface {
	Create(ctx context.Context, cost *hobbyfarmiov1.Cost, opts metav1.CreateOptions) (*hobbyfarmiov1.Cost, error)
	Update(ctx context.Context, cost *hobbyfarmiov1.Cost, opts metav1.UpdateOptions) (*hobbyfarmiov1.Cost, error)
	Delete(ctx context.Context, name string, opts metav1.DeleteOptions) error
	DeleteCollection(ctx context.Context, opts metav1.DeleteOptions, listOpts metav1.ListOptions) error
	Get(ctx context.Context, name string, opts metav1.GetOptions) (*hobbyfarmiov1.Cost, error)
	List(ctx context.Context, opts metav1.ListOptions) (*hobbyfarmiov1.CostList, error)
	Watch(ctx context.Context, opts metav1.ListOptions) (watch.Interface, error)
	Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts metav1.PatchOptions, subresources ...string) (result *hobbyfarmiov1.Cost, err error)
	CostExpansion
}

// costs implements CostInterface
type costs struct {
	*gentype.ClientWithList[*hobbyfarmiov1.Cost, *hobbyfarmiov1.CostList]
}

// newCosts returns a Costs
func newCosts(c *HobbyfarmV1Client, namespace string) *costs {
	return &costs{
		gentype.NewClientWithList[*hobbyfarmiov1.Cost, *hobbyfarmiov1.CostList](
			"costs",
			c.RESTClient(),
			scheme.ParameterCodec,
			namespace,
			func() *hobbyfarmiov1.Cost { return &hobbyfarmiov1.Cost{} },
			func() *hobbyfarmiov1.CostList { return &hobbyfarmiov1.CostList{} },
		),
	}
}
