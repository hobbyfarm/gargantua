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
	v1 "github.com/hobbyfarm/gargantua/pkg/apis/hobbyfarm.io/v1"
	"github.com/hobbyfarm/gargantua/pkg/client/clientset/versioned/scheme"
	rest "k8s.io/client-go/rest"
	"net/http"
)


type HobbyfarmV1Interface interface {
    RESTClient() rest.Interface
     AccessCodesGetter
     CoursesGetter
     DynamicBindConfigurationsGetter
     DynamicBindRequestsGetter
     EnvironmentsGetter
     ProgressesGetter
     ScenariosGetter
     ScheduledEventsGetter
     SessionsGetter
     UsersGetter
     VirtualMachinesGetter
     VirtualMachineClaimsGetter
     VirtualMachineSetsGetter
     VirtualMachineTemplatesGetter
    
}

// HobbyfarmV1Client is used to interact with features provided by the hobbyfarm.io group.
type HobbyfarmV1Client struct {
	restClient rest.Interface
}

func (c *HobbyfarmV1Client) AccessCodes(namespace string) AccessCodeInterface {
	return newAccessCodes(c, namespace)
}

func (c *HobbyfarmV1Client) Courses(namespace string) CourseInterface {
	return newCourses(c, namespace)
}

func (c *HobbyfarmV1Client) DynamicBindConfigurations(namespace string) DynamicBindConfigurationInterface {
	return newDynamicBindConfigurations(c, namespace)
}

func (c *HobbyfarmV1Client) DynamicBindRequests(namespace string) DynamicBindRequestInterface {
	return newDynamicBindRequests(c, namespace)
}

func (c *HobbyfarmV1Client) Environments(namespace string) EnvironmentInterface {
	return newEnvironments(c, namespace)
}

func (c *HobbyfarmV1Client) Progresses(namespace string) ProgressInterface {
	return newProgresses(c, namespace)
}

func (c *HobbyfarmV1Client) Scenarios(namespace string) ScenarioInterface {
	return newScenarios(c, namespace)
}

func (c *HobbyfarmV1Client) ScheduledEvents(namespace string) ScheduledEventInterface {
	return newScheduledEvents(c, namespace)
}

func (c *HobbyfarmV1Client) Sessions(namespace string) SessionInterface {
	return newSessions(c, namespace)
}

func (c *HobbyfarmV1Client) Users(namespace string) UserInterface {
	return newUsers(c, namespace)
}

func (c *HobbyfarmV1Client) VirtualMachines(namespace string) VirtualMachineInterface {
	return newVirtualMachines(c, namespace)
}

func (c *HobbyfarmV1Client) VirtualMachineClaims(namespace string) VirtualMachineClaimInterface {
	return newVirtualMachineClaims(c, namespace)
}

func (c *HobbyfarmV1Client) VirtualMachineSets(namespace string) VirtualMachineSetInterface {
	return newVirtualMachineSets(c, namespace)
}

func (c *HobbyfarmV1Client) VirtualMachineTemplates(namespace string) VirtualMachineTemplateInterface {
	return newVirtualMachineTemplates(c, namespace)
}

// NewForConfig creates a new HobbyfarmV1Client for the given config.
// NewForConfig is equivalent to NewForConfigAndClient(c, httpClient),
// where httpClient was generated with rest.HTTPClientFor(c).
func NewForConfig(c *rest.Config) (*HobbyfarmV1Client, error) {
	config := *c
	if err := setConfigDefaults(&config); err != nil {
		return nil, err
	}
	httpClient, err := rest.HTTPClientFor(&config)
	if err != nil {
		return nil, err
	}
	return NewForConfigAndClient(&config, httpClient)
}

// NewForConfigAndClient creates a new HobbyfarmV1Client for the given config and http client.
// Note the http client provided takes precedence over the configured transport values.
func NewForConfigAndClient(c *rest.Config, h *http.Client) (*HobbyfarmV1Client, error) {
	config := *c
	if err := setConfigDefaults(&config); err != nil {
		return nil, err
	}
	client, err := rest.RESTClientForConfigAndClient(&config, h)
	if err != nil {
		return nil, err
	}
	return &HobbyfarmV1Client{client}, nil
}

// NewForConfigOrDie creates a new HobbyfarmV1Client for the given config and
// panics if there is an error in the config.
func NewForConfigOrDie(c *rest.Config) *HobbyfarmV1Client {
	client, err := NewForConfig(c)
	if err != nil {
		panic(err)
	}
	return client
}

// New creates a new HobbyfarmV1Client for the given RESTClient.
func New(c rest.Interface) *HobbyfarmV1Client {
	return &HobbyfarmV1Client{c}
}

func setConfigDefaults(config *rest.Config) error {
	gv := v1.SchemeGroupVersion
	config.GroupVersion =  &gv
	config.APIPath = "/apis"
	config.NegotiatedSerializer = scheme.Codecs.WithoutConversion()

	if config.UserAgent == "" {
		config.UserAgent = rest.DefaultKubernetesUserAgent()
	}

	return nil
}

// RESTClient returns a RESTClient that is used to communicate
// with API server by this client implementation.
func (c *HobbyfarmV1Client) RESTClient() rest.Interface {
	if c == nil {
		return nil
	}
	return c.restClient
}
