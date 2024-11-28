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
	v1 "github.com/hobbyfarm/gargantua/v3/pkg/client/clientset/versioned/typed/hobbyfarm.io/v1"
	rest "k8s.io/client-go/rest"
	testing "k8s.io/client-go/testing"
)

type FakeHobbyfarmV1 struct {
	*testing.Fake
}

func (c *FakeHobbyfarmV1) AccessCodes(namespace string) v1.AccessCodeInterface {
	return &FakeAccessCodes{c, namespace}
}

func (c *FakeHobbyfarmV1) Courses(namespace string) v1.CourseInterface {
	return &FakeCourses{c, namespace}
}

func (c *FakeHobbyfarmV1) DynamicBindConfigurations(namespace string) v1.DynamicBindConfigurationInterface {
	return &FakeDynamicBindConfigurations{c, namespace}
}

func (c *FakeHobbyfarmV1) Environments(namespace string) v1.EnvironmentInterface {
	return &FakeEnvironments{c, namespace}
}

func (c *FakeHobbyfarmV1) OneTimeAccessCodes(namespace string) v1.OneTimeAccessCodeInterface {
	return &FakeOneTimeAccessCodes{c, namespace}
}

func (c *FakeHobbyfarmV1) PasswordResetTokens(namespace string) v1.PasswordResetTokenInterface {
	return &FakePasswordResetTokens{c, namespace}
}

func (c *FakeHobbyfarmV1) PredefinedServices(namespace string) v1.PredefinedServiceInterface {
	return &FakePredefinedServices{c, namespace}
}

func (c *FakeHobbyfarmV1) Progresses(namespace string) v1.ProgressInterface {
	return &FakeProgresses{c, namespace}
}

func (c *FakeHobbyfarmV1) Scenarios(namespace string) v1.ScenarioInterface {
	return &FakeScenarios{c, namespace}
}

func (c *FakeHobbyfarmV1) ScheduledEvents(namespace string) v1.ScheduledEventInterface {
	return &FakeScheduledEvents{c, namespace}
}

func (c *FakeHobbyfarmV1) Scopes(namespace string) v1.ScopeInterface {
	return &FakeScopes{c, namespace}
}

func (c *FakeHobbyfarmV1) Sessions(namespace string) v1.SessionInterface {
	return &FakeSessions{c, namespace}
}

func (c *FakeHobbyfarmV1) Settings(namespace string) v1.SettingInterface {
	return &FakeSettings{c, namespace}
}

func (c *FakeHobbyfarmV1) Users(namespace string) v1.UserInterface {
	return &FakeUsers{c, namespace}
}

func (c *FakeHobbyfarmV1) VirtualMachines(namespace string) v1.VirtualMachineInterface {
	return &FakeVirtualMachines{c, namespace}
}

func (c *FakeHobbyfarmV1) VirtualMachineClaims(namespace string) v1.VirtualMachineClaimInterface {
	return &FakeVirtualMachineClaims{c, namespace}
}

func (c *FakeHobbyfarmV1) VirtualMachineSets(namespace string) v1.VirtualMachineSetInterface {
	return &FakeVirtualMachineSets{c, namespace}
}

func (c *FakeHobbyfarmV1) VirtualMachineTemplates(namespace string) v1.VirtualMachineTemplateInterface {
	return &FakeVirtualMachineTemplates{c, namespace}
}

// RESTClient returns a RESTClient that is used to communicate
// with API server by this client implementation.
func (c *FakeHobbyfarmV1) RESTClient() rest.Interface {
	var ret *rest.RESTClient
	return ret
}
