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

// Code generated by informer-gen. DO NOT EDIT.

package v1

import (
	internalinterfaces "github.com/hobbyfarm/gargantua/pkg/client/informers/externalversions/internalinterfaces"
)

// Interface provides access to all the informers in this group version.
type Interface interface {
	// AccessCodes returns a AccessCodeInformer.
	AccessCodes() AccessCodeInformer
	// Courses returns a CourseInformer.
	Courses() CourseInformer
	// DynamicBindConfigurations returns a DynamicBindConfigurationInformer.
	DynamicBindConfigurations() DynamicBindConfigurationInformer
	// Environments returns a EnvironmentInformer.
	Environments() EnvironmentInformer
	// OneTimeAccessCodes returns a OneTimeAccessCodeInformer.
	OneTimeAccessCodes() OneTimeAccessCodeInformer
	// PredefinedServices returns a PredefinedServiceInformer.
	PredefinedServices() PredefinedServiceInformer
	// Progresses returns a ProgressInformer.
	Progresses() ProgressInformer
	// Scenarios returns a ScenarioInformer.
	Scenarios() ScenarioInformer
	// ScheduledEvents returns a ScheduledEventInformer.
	ScheduledEvents() ScheduledEventInformer
	// Scopes returns a ScopeInformer.
	Scopes() ScopeInformer
	// Sessions returns a SessionInformer.
	Sessions() SessionInformer
	// Settings returns a SettingInformer.
	Settings() SettingInformer
	// Users returns a UserInformer.
	Users() UserInformer
	// VirtualMachines returns a VirtualMachineInformer.
	VirtualMachines() VirtualMachineInformer
	// VirtualMachineClaims returns a VirtualMachineClaimInformer.
	VirtualMachineClaims() VirtualMachineClaimInformer
	// VirtualMachineSets returns a VirtualMachineSetInformer.
	VirtualMachineSets() VirtualMachineSetInformer
	// VirtualMachineTemplates returns a VirtualMachineTemplateInformer.
	VirtualMachineTemplates() VirtualMachineTemplateInformer
}

type version struct {
	factory          internalinterfaces.SharedInformerFactory
	namespace        string
	tweakListOptions internalinterfaces.TweakListOptionsFunc
}

// New returns a new Interface.
func New(f internalinterfaces.SharedInformerFactory, namespace string, tweakListOptions internalinterfaces.TweakListOptionsFunc) Interface {
	return &version{factory: f, namespace: namespace, tweakListOptions: tweakListOptions}
}

// AccessCodes returns a AccessCodeInformer.
func (v *version) AccessCodes() AccessCodeInformer {
	return &accessCodeInformer{factory: v.factory, namespace: v.namespace, tweakListOptions: v.tweakListOptions}
}

// Courses returns a CourseInformer.
func (v *version) Courses() CourseInformer {
	return &courseInformer{factory: v.factory, namespace: v.namespace, tweakListOptions: v.tweakListOptions}
}

// DynamicBindConfigurations returns a DynamicBindConfigurationInformer.
func (v *version) DynamicBindConfigurations() DynamicBindConfigurationInformer {
	return &dynamicBindConfigurationInformer{factory: v.factory, namespace: v.namespace, tweakListOptions: v.tweakListOptions}
}

// Environments returns a EnvironmentInformer.
func (v *version) Environments() EnvironmentInformer {
	return &environmentInformer{factory: v.factory, namespace: v.namespace, tweakListOptions: v.tweakListOptions}
}

// OneTimeAccessCodes returns a OneTimeAccessCodeInformer.
func (v *version) OneTimeAccessCodes() OneTimeAccessCodeInformer {
	return &oneTimeAccessCodeInformer{factory: v.factory, namespace: v.namespace, tweakListOptions: v.tweakListOptions}
}

// PredefinedServices returns a PredefinedServiceInformer.
func (v *version) PredefinedServices() PredefinedServiceInformer {
	return &predefinedServiceInformer{factory: v.factory, namespace: v.namespace, tweakListOptions: v.tweakListOptions}
}

// Progresses returns a ProgressInformer.
func (v *version) Progresses() ProgressInformer {
	return &progressInformer{factory: v.factory, namespace: v.namespace, tweakListOptions: v.tweakListOptions}
}

// Scenarios returns a ScenarioInformer.
func (v *version) Scenarios() ScenarioInformer {
	return &scenarioInformer{factory: v.factory, namespace: v.namespace, tweakListOptions: v.tweakListOptions}
}

// ScheduledEvents returns a ScheduledEventInformer.
func (v *version) ScheduledEvents() ScheduledEventInformer {
	return &scheduledEventInformer{factory: v.factory, namespace: v.namespace, tweakListOptions: v.tweakListOptions}
}

// Scopes returns a ScopeInformer.
func (v *version) Scopes() ScopeInformer {
	return &scopeInformer{factory: v.factory, namespace: v.namespace, tweakListOptions: v.tweakListOptions}
}

// Sessions returns a SessionInformer.
func (v *version) Sessions() SessionInformer {
	return &sessionInformer{factory: v.factory, namespace: v.namespace, tweakListOptions: v.tweakListOptions}
}

// Settings returns a SettingInformer.
func (v *version) Settings() SettingInformer {
	return &settingInformer{factory: v.factory, namespace: v.namespace, tweakListOptions: v.tweakListOptions}
}

// Users returns a UserInformer.
func (v *version) Users() UserInformer {
	return &userInformer{factory: v.factory, namespace: v.namespace, tweakListOptions: v.tweakListOptions}
}

// VirtualMachines returns a VirtualMachineInformer.
func (v *version) VirtualMachines() VirtualMachineInformer {
	return &virtualMachineInformer{factory: v.factory, namespace: v.namespace, tweakListOptions: v.tweakListOptions}
}

// VirtualMachineClaims returns a VirtualMachineClaimInformer.
func (v *version) VirtualMachineClaims() VirtualMachineClaimInformer {
	return &virtualMachineClaimInformer{factory: v.factory, namespace: v.namespace, tweakListOptions: v.tweakListOptions}
}

// VirtualMachineSets returns a VirtualMachineSetInformer.
func (v *version) VirtualMachineSets() VirtualMachineSetInformer {
	return &virtualMachineSetInformer{factory: v.factory, namespace: v.namespace, tweakListOptions: v.tweakListOptions}
}

// VirtualMachineTemplates returns a VirtualMachineTemplateInformer.
func (v *version) VirtualMachineTemplates() VirtualMachineTemplateInformer {
	return &virtualMachineTemplateInformer{factory: v.factory, namespace: v.namespace, tweakListOptions: v.tweakListOptions}
}
