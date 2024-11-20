package v4alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

var (
	SchemeGroupVersion = schema.GroupVersion{Group: APIGroup, Version: Version}
	SchemeBuilder      = runtime.NewSchemeBuilder(addKnownTypes)
	AddToScheme        = SchemeBuilder.AddToScheme
)

func addKnownTypes(scheme *runtime.Scheme) error {
	scheme.AddKnownTypes(SchemeGroupVersion,
		&Group{},
		&GroupList{},
		&LdapConfig{},
		&LdapConfigList{},
		&Provider{},
		&ProviderList{},
		&MachineTemplate{},
		&MachineTemplateList{},
		&Environment{},
		&EnvironmentList{},
		&MachineSet{},
		&MachineSetList{},
		&Machine{},
		&MachineList{},
		&MachineClaim{},
		&MachineClaimList{},
		&ScheduledEvent{},
		&ScheduledEventList{},
		&AccessCode{},
		&AccessCodeList{},
		&Session{},
		&SessionList{},
		&Course{},
		&CourseList{},
		&OneTimeAccessCode{},
		&OneTimeAccessCodeList{},
		&PredefinedService{},
		&PredefinedServiceList{},
		&Progress{},
		&ProgressList{},
		&Scenario{},
		&ScenarioList{},
		&ScenarioStep{},
		&ScenarioStepList{},
		&Scope{},
		&ScopeList{},
		&Setting{},
		&SettingList{},
		&User{},
		&UserList{},
		&ServiceAccount{},
		&ServiceAccountList{},
		&ConfigMap{},
		&ConfigMapList{},
		&Secret{},
		&SecretList{},
		&Role{},
		&RoleList{},
		&RoleBinding{},
		&RoleBindingList{},
	)

	metav1.AddToGroupVersion(scheme, SchemeGroupVersion)

	return nil
}
