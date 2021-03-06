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
	"time"

	v1 "github.com/hobbyfarm/gargantua/pkg/apis/hobbyfarm.io/v1"
	scheme "github.com/hobbyfarm/gargantua/pkg/client/clientset/versioned/scheme"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	types "k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"
	rest "k8s.io/client-go/rest"
)

// EnvironmentsGetter has a method to return a EnvironmentInterface.
// A group's client should implement this interface.
type EnvironmentsGetter interface {
	Environments() EnvironmentInterface
}

// EnvironmentInterface has methods to work with Environment resources.
type EnvironmentInterface interface {
	Create(*v1.Environment) (*v1.Environment, error)
	Update(*v1.Environment) (*v1.Environment, error)
	UpdateStatus(*v1.Environment) (*v1.Environment, error)
	Delete(name string, options *metav1.DeleteOptions) error
	DeleteCollection(options *metav1.DeleteOptions, listOptions metav1.ListOptions) error
	Get(name string, options metav1.GetOptions) (*v1.Environment, error)
	List(opts metav1.ListOptions) (*v1.EnvironmentList, error)
	Watch(opts metav1.ListOptions) (watch.Interface, error)
	Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *v1.Environment, err error)
	EnvironmentExpansion
}

// environments implements EnvironmentInterface
type environments struct {
	client rest.Interface
}

// newEnvironments returns a Environments
func newEnvironments(c *HobbyfarmV1Client) *environments {
	return &environments{
		client: c.RESTClient(),
	}
}

// Get takes name of the environment, and returns the corresponding environment object, and an error if there is any.
func (c *environments) Get(name string, options metav1.GetOptions) (result *v1.Environment, err error) {
	result = &v1.Environment{}
	err = c.client.Get().
		Resource("environments").
		Name(name).
		VersionedParams(&options, scheme.ParameterCodec).
		Do().
		Into(result)
	return
}

// List takes label and field selectors, and returns the list of Environments that match those selectors.
func (c *environments) List(opts metav1.ListOptions) (result *v1.EnvironmentList, err error) {
	var timeout time.Duration
	if opts.TimeoutSeconds != nil {
		timeout = time.Duration(*opts.TimeoutSeconds) * time.Second
	}
	result = &v1.EnvironmentList{}
	err = c.client.Get().
		Resource("environments").
		VersionedParams(&opts, scheme.ParameterCodec).
		Timeout(timeout).
		Do().
		Into(result)
	return
}

// Watch returns a watch.Interface that watches the requested environments.
func (c *environments) Watch(opts metav1.ListOptions) (watch.Interface, error) {
	var timeout time.Duration
	if opts.TimeoutSeconds != nil {
		timeout = time.Duration(*opts.TimeoutSeconds) * time.Second
	}
	opts.Watch = true
	return c.client.Get().
		Resource("environments").
		VersionedParams(&opts, scheme.ParameterCodec).
		Timeout(timeout).
		Watch()
}

// Create takes the representation of a environment and creates it.  Returns the server's representation of the environment, and an error, if there is any.
func (c *environments) Create(environment *v1.Environment) (result *v1.Environment, err error) {
	result = &v1.Environment{}
	err = c.client.Post().
		Resource("environments").
		Body(environment).
		Do().
		Into(result)
	return
}

// Update takes the representation of a environment and updates it. Returns the server's representation of the environment, and an error, if there is any.
func (c *environments) Update(environment *v1.Environment) (result *v1.Environment, err error) {
	result = &v1.Environment{}
	err = c.client.Put().
		Resource("environments").
		Name(environment.Name).
		Body(environment).
		Do().
		Into(result)
	return
}

// UpdateStatus was generated because the type contains a Status member.
// Add a +genclient:noStatus comment above the type to avoid generating UpdateStatus().

func (c *environments) UpdateStatus(environment *v1.Environment) (result *v1.Environment, err error) {
	result = &v1.Environment{}
	err = c.client.Put().
		Resource("environments").
		Name(environment.Name).
		SubResource("status").
		Body(environment).
		Do().
		Into(result)
	return
}

// Delete takes name of the environment and deletes it. Returns an error if one occurs.
func (c *environments) Delete(name string, options *metav1.DeleteOptions) error {
	return c.client.Delete().
		Resource("environments").
		Name(name).
		Body(options).
		Do().
		Error()
}

// DeleteCollection deletes a collection of objects.
func (c *environments) DeleteCollection(options *metav1.DeleteOptions, listOptions metav1.ListOptions) error {
	var timeout time.Duration
	if listOptions.TimeoutSeconds != nil {
		timeout = time.Duration(*listOptions.TimeoutSeconds) * time.Second
	}
	return c.client.Delete().
		Resource("environments").
		VersionedParams(&listOptions, scheme.ParameterCodec).
		Timeout(timeout).
		Body(options).
		Do().
		Error()
}

// Patch applies the patch and returns the patched environment.
func (c *environments) Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *v1.Environment, err error) {
	result = &v1.Environment{}
	err = c.client.Patch(pt).
		Resource("environments").
		SubResource(subresources...).
		Name(name).
		Body(data).
		Do().
		Into(result)
	return
}
