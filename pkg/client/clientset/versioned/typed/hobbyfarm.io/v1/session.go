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

// SessionsGetter has a method to return a SessionInterface.
// A group's client should implement this interface.
type SessionsGetter interface {
	Sessions() SessionInterface
}

// SessionInterface has methods to work with Session resources.
type SessionInterface interface {
	Create(*v1.Session) (*v1.Session, error)
	Update(*v1.Session) (*v1.Session, error)
	UpdateStatus(*v1.Session) (*v1.Session, error)
	Delete(name string, options *metav1.DeleteOptions) error
	DeleteCollection(options *metav1.DeleteOptions, listOptions metav1.ListOptions) error
	Get(name string, options metav1.GetOptions) (*v1.Session, error)
	List(opts metav1.ListOptions) (*v1.SessionList, error)
	Watch(opts metav1.ListOptions) (watch.Interface, error)
	Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *v1.Session, err error)
	SessionExpansion
}

// sessions implements SessionInterface
type sessions struct {
	client rest.Interface
}

// newSessions returns a Sessions
func newSessions(c *HobbyfarmV1Client) *sessions {
	return &sessions{
		client: c.RESTClient(),
	}
}

// Get takes name of the session, and returns the corresponding session object, and an error if there is any.
func (c *sessions) Get(name string, options metav1.GetOptions) (result *v1.Session, err error) {
	result = &v1.Session{}
	err = c.client.Get().
		Resource("sessions").
		Name(name).
		VersionedParams(&options, scheme.ParameterCodec).
		Do().
		Into(result)
	return
}

// List takes label and field selectors, and returns the list of Sessions that match those selectors.
func (c *sessions) List(opts metav1.ListOptions) (result *v1.SessionList, err error) {
	var timeout time.Duration
	if opts.TimeoutSeconds != nil {
		timeout = time.Duration(*opts.TimeoutSeconds) * time.Second
	}
	result = &v1.SessionList{}
	err = c.client.Get().
		Resource("sessions").
		VersionedParams(&opts, scheme.ParameterCodec).
		Timeout(timeout).
		Do().
		Into(result)
	return
}

// Watch returns a watch.Interface that watches the requested sessions.
func (c *sessions) Watch(opts metav1.ListOptions) (watch.Interface, error) {
	var timeout time.Duration
	if opts.TimeoutSeconds != nil {
		timeout = time.Duration(*opts.TimeoutSeconds) * time.Second
	}
	opts.Watch = true
	return c.client.Get().
		Resource("sessions").
		VersionedParams(&opts, scheme.ParameterCodec).
		Timeout(timeout).
		Watch()
}

// Create takes the representation of a session and creates it.  Returns the server's representation of the session, and an error, if there is any.
func (c *sessions) Create(session *v1.Session) (result *v1.Session, err error) {
	result = &v1.Session{}
	err = c.client.Post().
		Resource("sessions").
		Body(session).
		Do().
		Into(result)
	return
}

// Update takes the representation of a session and updates it. Returns the server's representation of the session, and an error, if there is any.
func (c *sessions) Update(session *v1.Session) (result *v1.Session, err error) {
	result = &v1.Session{}
	err = c.client.Put().
		Resource("sessions").
		Name(session.Name).
		Body(session).
		Do().
		Into(result)
	return
}

// UpdateStatus was generated because the type contains a Status member.
// Add a +genclient:noStatus comment above the type to avoid generating UpdateStatus().

func (c *sessions) UpdateStatus(session *v1.Session) (result *v1.Session, err error) {
	result = &v1.Session{}
	err = c.client.Put().
		Resource("sessions").
		Name(session.Name).
		SubResource("status").
		Body(session).
		Do().
		Into(result)
	return
}

// Delete takes name of the session and deletes it. Returns an error if one occurs.
func (c *sessions) Delete(name string, options *metav1.DeleteOptions) error {
	return c.client.Delete().
		Resource("sessions").
		Name(name).
		Body(options).
		Do().
		Error()
}

// DeleteCollection deletes a collection of objects.
func (c *sessions) DeleteCollection(options *metav1.DeleteOptions, listOptions metav1.ListOptions) error {
	var timeout time.Duration
	if listOptions.TimeoutSeconds != nil {
		timeout = time.Duration(*listOptions.TimeoutSeconds) * time.Second
	}
	return c.client.Delete().
		Resource("sessions").
		VersionedParams(&listOptions, scheme.ParameterCodec).
		Timeout(timeout).
		Body(options).
		Do().
		Error()
}

// Patch applies the patch and returns the patched session.
func (c *sessions) Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *v1.Session, err error) {
	result = &v1.Session{}
	err = c.client.Patch(pt).
		Resource("sessions").
		SubResource(subresources...).
		Name(name).
		Body(data).
		Do().
		Into(result)
	return
}