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
	"context"
	"time"

	v1 "github.com/hobbyfarm/gargantua/v3/pkg/apis/hobbyfarm.io/v1"
	scheme "github.com/hobbyfarm/gargantua/v3/pkg/client/clientset/versioned/scheme"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	types "k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"
	rest "k8s.io/client-go/rest"
)

// CostsGetter has a method to return a CostInterface.
// A group's client should implement this interface.
type CostsGetter interface {
	Costs(namespace string) CostInterface
}

// CostInterface has methods to work with Cost resources.
type CostInterface interface {
	Create(ctx context.Context, cost *v1.Cost, opts metav1.CreateOptions) (*v1.Cost, error)
	Update(ctx context.Context, cost *v1.Cost, opts metav1.UpdateOptions) (*v1.Cost, error)
	Delete(ctx context.Context, name string, opts metav1.DeleteOptions) error
	DeleteCollection(ctx context.Context, opts metav1.DeleteOptions, listOpts metav1.ListOptions) error
	Get(ctx context.Context, name string, opts metav1.GetOptions) (*v1.Cost, error)
	List(ctx context.Context, opts metav1.ListOptions) (*v1.CostList, error)
	Watch(ctx context.Context, opts metav1.ListOptions) (watch.Interface, error)
	Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts metav1.PatchOptions, subresources ...string) (result *v1.Cost, err error)
	CostExpansion
}

// costs implements CostInterface
type costs struct {
	client rest.Interface
	ns     string
}

// newCosts returns a Costs
func newCosts(c *HobbyfarmV1Client, namespace string) *costs {
	return &costs{
		client: c.RESTClient(),
		ns:     namespace,
	}
}

// Get takes name of the cost, and returns the corresponding cost object, and an error if there is any.
func (c *costs) Get(ctx context.Context, name string, options metav1.GetOptions) (result *v1.Cost, err error) {
	result = &v1.Cost{}
	err = c.client.Get().
		Namespace(c.ns).
		Resource("costs").
		Name(name).
		VersionedParams(&options, scheme.ParameterCodec).
		Do(ctx).
		Into(result)
	return
}

// List takes label and field selectors, and returns the list of Costs that match those selectors.
func (c *costs) List(ctx context.Context, opts metav1.ListOptions) (result *v1.CostList, err error) {
	var timeout time.Duration
	if opts.TimeoutSeconds != nil {
		timeout = time.Duration(*opts.TimeoutSeconds) * time.Second
	}
	result = &v1.CostList{}
	err = c.client.Get().
		Namespace(c.ns).
		Resource("costs").
		VersionedParams(&opts, scheme.ParameterCodec).
		Timeout(timeout).
		Do(ctx).
		Into(result)
	return
}

// Watch returns a watch.Interface that watches the requested costs.
func (c *costs) Watch(ctx context.Context, opts metav1.ListOptions) (watch.Interface, error) {
	var timeout time.Duration
	if opts.TimeoutSeconds != nil {
		timeout = time.Duration(*opts.TimeoutSeconds) * time.Second
	}
	opts.Watch = true
	return c.client.Get().
		Namespace(c.ns).
		Resource("costs").
		VersionedParams(&opts, scheme.ParameterCodec).
		Timeout(timeout).
		Watch(ctx)
}

// Create takes the representation of a cost and creates it.  Returns the server's representation of the cost, and an error, if there is any.
func (c *costs) Create(ctx context.Context, cost *v1.Cost, opts metav1.CreateOptions) (result *v1.Cost, err error) {
	result = &v1.Cost{}
	err = c.client.Post().
		Namespace(c.ns).
		Resource("costs").
		VersionedParams(&opts, scheme.ParameterCodec).
		Body(cost).
		Do(ctx).
		Into(result)
	return
}

// Update takes the representation of a cost and updates it. Returns the server's representation of the cost, and an error, if there is any.
func (c *costs) Update(ctx context.Context, cost *v1.Cost, opts metav1.UpdateOptions) (result *v1.Cost, err error) {
	result = &v1.Cost{}
	err = c.client.Put().
		Namespace(c.ns).
		Resource("costs").
		Name(cost.Name).
		VersionedParams(&opts, scheme.ParameterCodec).
		Body(cost).
		Do(ctx).
		Into(result)
	return
}

// Delete takes name of the cost and deletes it. Returns an error if one occurs.
func (c *costs) Delete(ctx context.Context, name string, opts metav1.DeleteOptions) error {
	return c.client.Delete().
		Namespace(c.ns).
		Resource("costs").
		Name(name).
		Body(&opts).
		Do(ctx).
		Error()
}

// DeleteCollection deletes a collection of objects.
func (c *costs) DeleteCollection(ctx context.Context, opts metav1.DeleteOptions, listOpts metav1.ListOptions) error {
	var timeout time.Duration
	if listOpts.TimeoutSeconds != nil {
		timeout = time.Duration(*listOpts.TimeoutSeconds) * time.Second
	}
	return c.client.Delete().
		Namespace(c.ns).
		Resource("costs").
		VersionedParams(&listOpts, scheme.ParameterCodec).
		Timeout(timeout).
		Body(&opts).
		Do(ctx).
		Error()
}

// Patch applies the patch and returns the patched cost.
func (c *costs) Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts metav1.PatchOptions, subresources ...string) (result *v1.Cost, err error) {
	result = &v1.Cost{}
	err = c.client.Patch(pt).
		Namespace(c.ns).
		Resource("costs").
		Name(name).
		SubResource(subresources...).
		VersionedParams(&opts, scheme.ParameterCodec).
		Body(data).
		Do(ctx).
		Into(result)
	return
}
