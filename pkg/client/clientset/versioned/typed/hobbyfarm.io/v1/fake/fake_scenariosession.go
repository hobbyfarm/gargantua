// Code generated by client-gen. DO NOT EDIT.

package fake

import (
	hobbyfarmiov1 "github.com/hobbyfarm/gargantua/pkg/apis/hobbyfarm.io/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	labels "k8s.io/apimachinery/pkg/labels"
	schema "k8s.io/apimachinery/pkg/runtime/schema"
	types "k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"
	testing "k8s.io/client-go/testing"
)

// FakeScenarioSessions implements ScenarioSessionInterface
type FakeScenarioSessions struct {
	Fake *FakeHobbyfarmV1
}

var scenariosessionsResource = schema.GroupVersionResource{Group: "hobbyfarm.io", Version: "v1", Resource: "scenariosessions"}

var scenariosessionsKind = schema.GroupVersionKind{Group: "hobbyfarm.io", Version: "v1", Kind: "ScenarioSession"}

// Get takes name of the scenarioSession, and returns the corresponding scenarioSession object, and an error if there is any.
func (c *FakeScenarioSessions) Get(name string, options v1.GetOptions) (result *hobbyfarmiov1.ScenarioSession, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewRootGetAction(scenariosessionsResource, name), &hobbyfarmiov1.ScenarioSession{})
	if obj == nil {
		return nil, err
	}
	return obj.(*hobbyfarmiov1.ScenarioSession), err
}

// List takes label and field selectors, and returns the list of ScenarioSessions that match those selectors.
func (c *FakeScenarioSessions) List(opts v1.ListOptions) (result *hobbyfarmiov1.ScenarioSessionList, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewRootListAction(scenariosessionsResource, scenariosessionsKind, opts), &hobbyfarmiov1.ScenarioSessionList{})
	if obj == nil {
		return nil, err
	}

	label, _, _ := testing.ExtractFromListOptions(opts)
	if label == nil {
		label = labels.Everything()
	}
	list := &hobbyfarmiov1.ScenarioSessionList{ListMeta: obj.(*hobbyfarmiov1.ScenarioSessionList).ListMeta}
	for _, item := range obj.(*hobbyfarmiov1.ScenarioSessionList).Items {
		if label.Matches(labels.Set(item.Labels)) {
			list.Items = append(list.Items, item)
		}
	}
	return list, err
}

// Watch returns a watch.Interface that watches the requested scenarioSessions.
func (c *FakeScenarioSessions) Watch(opts v1.ListOptions) (watch.Interface, error) {
	return c.Fake.
		InvokesWatch(testing.NewRootWatchAction(scenariosessionsResource, opts))
}

// Create takes the representation of a scenarioSession and creates it.  Returns the server's representation of the scenarioSession, and an error, if there is any.
func (c *FakeScenarioSessions) Create(scenarioSession *hobbyfarmiov1.ScenarioSession) (result *hobbyfarmiov1.ScenarioSession, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewRootCreateAction(scenariosessionsResource, scenarioSession), &hobbyfarmiov1.ScenarioSession{})
	if obj == nil {
		return nil, err
	}
	return obj.(*hobbyfarmiov1.ScenarioSession), err
}

// Update takes the representation of a scenarioSession and updates it. Returns the server's representation of the scenarioSession, and an error, if there is any.
func (c *FakeScenarioSessions) Update(scenarioSession *hobbyfarmiov1.ScenarioSession) (result *hobbyfarmiov1.ScenarioSession, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewRootUpdateAction(scenariosessionsResource, scenarioSession), &hobbyfarmiov1.ScenarioSession{})
	if obj == nil {
		return nil, err
	}
	return obj.(*hobbyfarmiov1.ScenarioSession), err
}

// UpdateStatus was generated because the type contains a Status member.
// Add a +genclient:noStatus comment above the type to avoid generating UpdateStatus().
func (c *FakeScenarioSessions) UpdateStatus(scenarioSession *hobbyfarmiov1.ScenarioSession) (*hobbyfarmiov1.ScenarioSession, error) {
	obj, err := c.Fake.
		Invokes(testing.NewRootUpdateSubresourceAction(scenariosessionsResource, "status", scenarioSession), &hobbyfarmiov1.ScenarioSession{})
	if obj == nil {
		return nil, err
	}
	return obj.(*hobbyfarmiov1.ScenarioSession), err
}

// Delete takes name of the scenarioSession and deletes it. Returns an error if one occurs.
func (c *FakeScenarioSessions) Delete(name string, options *v1.DeleteOptions) error {
	_, err := c.Fake.
		Invokes(testing.NewRootDeleteAction(scenariosessionsResource, name), &hobbyfarmiov1.ScenarioSession{})
	return err
}

// DeleteCollection deletes a collection of objects.
func (c *FakeScenarioSessions) DeleteCollection(options *v1.DeleteOptions, listOptions v1.ListOptions) error {
	action := testing.NewRootDeleteCollectionAction(scenariosessionsResource, listOptions)

	_, err := c.Fake.Invokes(action, &hobbyfarmiov1.ScenarioSessionList{})
	return err
}

// Patch applies the patch and returns the patched scenarioSession.
func (c *FakeScenarioSessions) Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *hobbyfarmiov1.ScenarioSession, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewRootPatchSubresourceAction(scenariosessionsResource, name, pt, data, subresources...), &hobbyfarmiov1.ScenarioSession{})
	if obj == nil {
		return nil, err
	}
	return obj.(*hobbyfarmiov1.ScenarioSession), err
}
