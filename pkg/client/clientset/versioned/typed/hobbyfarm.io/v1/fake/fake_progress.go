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

	hobbyfarmiov1 "github.com/hobbyfarm/gargantua/v3/pkg/apis/hobbyfarm.io/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	labels "k8s.io/apimachinery/pkg/labels"
	schema "k8s.io/apimachinery/pkg/runtime/schema"
	types "k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"
	testing "k8s.io/client-go/testing"
)

// FakeProgresses implements ProgressInterface
type FakeProgresses struct {
	Fake *FakeHobbyfarmV1
	ns   string
}

var progressesResource = schema.GroupVersionResource{Group: "hobbyfarm.io", Version: "v1", Resource: "progresses"}

var progressesKind = schema.GroupVersionKind{Group: "hobbyfarm.io", Version: "v1", Kind: "Progress"}

// Get takes name of the progress, and returns the corresponding progress object, and an error if there is any.
func (c *FakeProgresses) Get(ctx context.Context, name string, options v1.GetOptions) (result *hobbyfarmiov1.Progress, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewGetAction(progressesResource, c.ns, name), &hobbyfarmiov1.Progress{})

	if obj == nil {
		return nil, err
	}
	return obj.(*hobbyfarmiov1.Progress), err
}

// List takes label and field selectors, and returns the list of Progresses that match those selectors.
func (c *FakeProgresses) List(ctx context.Context, opts v1.ListOptions) (result *hobbyfarmiov1.ProgressList, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewListAction(progressesResource, progressesKind, c.ns, opts), &hobbyfarmiov1.ProgressList{})

	if obj == nil {
		return nil, err
	}

	label, _, _ := testing.ExtractFromListOptions(opts)
	if label == nil {
		label = labels.Everything()
	}
	list := &hobbyfarmiov1.ProgressList{ListMeta: obj.(*hobbyfarmiov1.ProgressList).ListMeta}
	for _, item := range obj.(*hobbyfarmiov1.ProgressList).Items {
		if label.Matches(labels.Set(item.Labels)) {
			list.Items = append(list.Items, item)
		}
	}
	return list, err
}

// Watch returns a watch.Interface that watches the requested progresses.
func (c *FakeProgresses) Watch(ctx context.Context, opts v1.ListOptions) (watch.Interface, error) {
	return c.Fake.
		InvokesWatch(testing.NewWatchAction(progressesResource, c.ns, opts))

}

// Create takes the representation of a progress and creates it.  Returns the server's representation of the progress, and an error, if there is any.
func (c *FakeProgresses) Create(ctx context.Context, progress *hobbyfarmiov1.Progress, opts v1.CreateOptions) (result *hobbyfarmiov1.Progress, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewCreateAction(progressesResource, c.ns, progress), &hobbyfarmiov1.Progress{})

	if obj == nil {
		return nil, err
	}
	return obj.(*hobbyfarmiov1.Progress), err
}

// Update takes the representation of a progress and updates it. Returns the server's representation of the progress, and an error, if there is any.
func (c *FakeProgresses) Update(ctx context.Context, progress *hobbyfarmiov1.Progress, opts v1.UpdateOptions) (result *hobbyfarmiov1.Progress, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewUpdateAction(progressesResource, c.ns, progress), &hobbyfarmiov1.Progress{})

	if obj == nil {
		return nil, err
	}
	return obj.(*hobbyfarmiov1.Progress), err
}

// Delete takes name of the progress and deletes it. Returns an error if one occurs.
func (c *FakeProgresses) Delete(ctx context.Context, name string, opts v1.DeleteOptions) error {
	_, err := c.Fake.
		Invokes(testing.NewDeleteActionWithOptions(progressesResource, c.ns, name, opts), &hobbyfarmiov1.Progress{})

	return err
}

// DeleteCollection deletes a collection of objects.
func (c *FakeProgresses) DeleteCollection(ctx context.Context, opts v1.DeleteOptions, listOpts v1.ListOptions) error {
	action := testing.NewDeleteCollectionAction(progressesResource, c.ns, listOpts)

	_, err := c.Fake.Invokes(action, &hobbyfarmiov1.ProgressList{})
	return err
}

// Patch applies the patch and returns the patched progress.
func (c *FakeProgresses) Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts v1.PatchOptions, subresources ...string) (result *hobbyfarmiov1.Progress, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewPatchSubresourceAction(progressesResource, c.ns, name, pt, data, subresources...), &hobbyfarmiov1.Progress{})

	if obj == nil {
		return nil, err
	}
	return obj.(*hobbyfarmiov1.Progress), err
}
