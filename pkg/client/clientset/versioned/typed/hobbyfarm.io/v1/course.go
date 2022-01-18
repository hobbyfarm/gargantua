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

	v1 "github.com/hobbyfarm/gargantua/pkg/apis/hobbyfarm.io/v1"
	scheme "github.com/hobbyfarm/gargantua/pkg/client/clientset/versioned/scheme"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	types "k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"
	rest "k8s.io/client-go/rest"
)

// CoursesGetter has a method to return a CourseInterface.
// A group's client should implement this interface.
type CoursesGetter interface {
	Courses(namespace string) CourseInterface
}

// CourseInterface has methods to work with Course resources.
type CourseInterface interface {
Create(ctx context.Context, course *v1.Course, opts metav1.CreateOptions) (*v1.Course, error)
Update(ctx context.Context, course *v1.Course, opts metav1.UpdateOptions) (*v1.Course, error)
Delete(ctx context.Context, name string, opts metav1.DeleteOptions) error
DeleteCollection(ctx context.Context, opts metav1.DeleteOptions, listOpts metav1.ListOptions) error
Get(ctx context.Context, name string, opts metav1.GetOptions) (*v1.Course, error)
List(ctx context.Context, opts metav1.ListOptions) (*v1.CourseList, error)
Watch(ctx context.Context, opts metav1.ListOptions) (watch.Interface, error)
Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts metav1.PatchOptions, subresources ...string) (result *v1.Course, err error)
	CourseExpansion
}

// courses implements CourseInterface
type courses struct {
	client rest.Interface
	ns     string
}

// newCourses returns a Courses
func newCourses(c *HobbyfarmV1Client, namespace string) *courses {
	return &courses{
		client: c.RESTClient(),
		ns:     namespace,
	}
}

// Get takes name of the course, and returns the corresponding course object, and an error if there is any.
func (c *courses) Get(ctx context.Context, name string, options metav1.GetOptions) (result *v1.Course, err error) {
	result = &v1.Course{}
	err = c.client.Get().
		Namespace(c.ns).
		Resource("courses").
		Name(name).
		VersionedParams(&options, scheme.ParameterCodec).
		Do(ctx).
		Into(result)
	return
}

// List takes label and field selectors, and returns the list of Courses that match those selectors.
func (c *courses) List(ctx context.Context, opts metav1.ListOptions) (result *v1.CourseList, err error) {
	var timeout time.Duration
	if opts.TimeoutSeconds != nil{
		timeout = time.Duration(*opts.TimeoutSeconds) * time.Second
	}
	result = &v1.CourseList{}
	err = c.client.Get().
		Namespace(c.ns).
		Resource("courses").
		VersionedParams(&opts, scheme.ParameterCodec).
		Timeout(timeout).
		Do(ctx).
		Into(result)
	return
}

// Watch returns a watch.Interface that watches the requested courses.
func (c *courses) Watch(ctx context.Context, opts metav1.ListOptions) (watch.Interface, error) {
	var timeout time.Duration
	if opts.TimeoutSeconds != nil{
		timeout = time.Duration(*opts.TimeoutSeconds) * time.Second
	}
	opts.Watch = true
	return c.client.Get().
		Namespace(c.ns).
		Resource("courses").
		VersionedParams(&opts, scheme.ParameterCodec).
		Timeout(timeout).
		Watch(ctx)
}

// Create takes the representation of a course and creates it.  Returns the server's representation of the course, and an error, if there is any.
func (c *courses) Create(ctx context.Context, course *v1.Course, opts metav1.CreateOptions) (result *v1.Course, err error) {
	result = &v1.Course{}
	err = c.client.Post().
		Namespace(c.ns).
		Resource("courses").
		VersionedParams(&opts, scheme.ParameterCodec).
		Body(course).
		Do(ctx).
		Into(result)
	return
}

// Update takes the representation of a course and updates it. Returns the server's representation of the course, and an error, if there is any.
func (c *courses) Update(ctx context.Context, course *v1.Course, opts metav1.UpdateOptions) (result *v1.Course, err error) {
	result = &v1.Course{}
	err = c.client.Put().
		Namespace(c.ns).
		Resource("courses").
		Name(course.Name).
		VersionedParams(&opts, scheme.ParameterCodec).
		Body(course).
		Do(ctx).
		Into(result)
	return
}

// Delete takes name of the course and deletes it. Returns an error if one occurs.
func (c *courses) Delete(ctx context.Context, name string, opts metav1.DeleteOptions) error {
	return c.client.Delete().
		Namespace(c.ns).
		Resource("courses").
		Name(name).
		Body(&opts).
		Do(ctx).
		Error()
}

// DeleteCollection deletes a collection of objects.
func (c *courses) DeleteCollection(ctx context.Context, opts metav1.DeleteOptions, listOpts metav1.ListOptions) error {
	var timeout time.Duration
	if listOpts.TimeoutSeconds != nil{
		timeout = time.Duration(*listOpts.TimeoutSeconds) * time.Second
	}
	return c.client.Delete().
		Namespace(c.ns).
		Resource("courses").
		VersionedParams(&listOpts, scheme.ParameterCodec).
		Timeout(timeout).
		Body(&opts).
		Do(ctx).
		Error()
}

// Patch applies the patch and returns the patched course.
func (c *courses) Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts metav1.PatchOptions, subresources ...string) (result *v1.Course, err error) {
	result = &v1.Course{}
	err = c.client.Patch(pt).
		Namespace(c.ns).
		Resource("courses").
		Name(name).
		SubResource(subresources...).
		VersionedParams(&opts, scheme.ParameterCodec).
		Body(data).
		Do(ctx).
		Into(result)
	return
}
