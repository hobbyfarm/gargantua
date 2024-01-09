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

package fake

import (
	"context"

	v1 "github.com/hobbyfarm/gargantua/v3/pkg/apis/hobbyfarm.io/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	labels "k8s.io/apimachinery/pkg/labels"
	types "k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"
	testing "k8s.io/client-go/testing"
)

// FakePredefinedServices implements PredefinedServiceInterface
type FakePredefinedServices struct {
	Fake *FakeHobbyfarmV1
	ns   string
}

var predefinedservicesResource = v1.SchemeGroupVersion.WithResource("predefinedservices")

var predefinedservicesKind = v1.SchemeGroupVersion.WithKind("PredefinedService")

// Get takes name of the predefinedService, and returns the corresponding predefinedService object, and an error if there is any.
func (c *FakePredefinedServices) Get(ctx context.Context, name string, options metav1.GetOptions) (result *v1.PredefinedService, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewGetAction(predefinedservicesResource, c.ns, name), &v1.PredefinedService{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1.PredefinedService), err
}

// List takes label and field selectors, and returns the list of PredefinedServices that match those selectors.
func (c *FakePredefinedServices) List(ctx context.Context, opts metav1.ListOptions) (result *v1.PredefinedServiceList, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewListAction(predefinedservicesResource, predefinedservicesKind, c.ns, opts), &v1.PredefinedServiceList{})

	if obj == nil {
		return nil, err
	}

	label, _, _ := testing.ExtractFromListOptions(opts)
	if label == nil {
		label = labels.Everything()
	}
	list := &v1.PredefinedServiceList{ListMeta: obj.(*v1.PredefinedServiceList).ListMeta}
	for _, item := range obj.(*v1.PredefinedServiceList).Items {
		if label.Matches(labels.Set(item.Labels)) {
			list.Items = append(list.Items, item)
		}
	}
	return list, err
}

// Watch returns a watch.Interface that watches the requested predefinedServices.
func (c *FakePredefinedServices) Watch(ctx context.Context, opts metav1.ListOptions) (watch.Interface, error) {
	return c.Fake.
		InvokesWatch(testing.NewWatchAction(predefinedservicesResource, c.ns, opts))

}

// Create takes the representation of a predefinedService and creates it.  Returns the server's representation of the predefinedService, and an error, if there is any.
func (c *FakePredefinedServices) Create(ctx context.Context, predefinedService *v1.PredefinedService, opts metav1.CreateOptions) (result *v1.PredefinedService, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewCreateAction(predefinedservicesResource, c.ns, predefinedService), &v1.PredefinedService{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1.PredefinedService), err
}

// Update takes the representation of a predefinedService and updates it. Returns the server's representation of the predefinedService, and an error, if there is any.
func (c *FakePredefinedServices) Update(ctx context.Context, predefinedService *v1.PredefinedService, opts metav1.UpdateOptions) (result *v1.PredefinedService, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewUpdateAction(predefinedservicesResource, c.ns, predefinedService), &v1.PredefinedService{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1.PredefinedService), err
}

// Delete takes name of the predefinedService and deletes it. Returns an error if one occurs.
func (c *FakePredefinedServices) Delete(ctx context.Context, name string, opts metav1.DeleteOptions) error {
	_, err := c.Fake.
		Invokes(testing.NewDeleteActionWithOptions(predefinedservicesResource, c.ns, name, opts), &v1.PredefinedService{})

	return err
}

// DeleteCollection deletes a collection of objects.
func (c *FakePredefinedServices) DeleteCollection(ctx context.Context, opts metav1.DeleteOptions, listOpts metav1.ListOptions) error {
	action := testing.NewDeleteCollectionAction(predefinedservicesResource, c.ns, listOpts)

	_, err := c.Fake.Invokes(action, &v1.PredefinedServiceList{})
	return err
}

// Patch applies the patch and returns the patched predefinedService.
func (c *FakePredefinedServices) Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts metav1.PatchOptions, subresources ...string) (result *v1.PredefinedService, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewPatchSubresourceAction(predefinedservicesResource, c.ns, name, pt, data, subresources...), &v1.PredefinedService{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1.PredefinedService), err
}
