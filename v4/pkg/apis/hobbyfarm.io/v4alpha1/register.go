package v4alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

var (
	SchemeGroupVersion = schema.GroupVersion{Group: Group, Version: Version}
	SchemeBuilder      = runtime.NewSchemeBuilder(addKnownTypes)
	AddToScheme        = SchemeBuilder.AddToScheme
)

func addKnownTypes(scheme *runtime.Scheme) error {
	scheme.AddKnownTypes(SchemeGroupVersion,
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
	)

	metav1.AddToGroupVersion(scheme, SchemeGroupVersion)

	return nil
}
