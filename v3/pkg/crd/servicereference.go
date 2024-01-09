package crd

import (
	v1 "k8s.io/api/admissionregistration/v1"
	apiextv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"k8s.io/utils/pointer"
)

type ServiceReference struct {
	Namespace string
	Name      string
	Path      *string
	Port      *int32
}

func (s ServiceReference) Toapiextv1WithPath(path string) (out apiextv1.ServiceReference) {
	out.Namespace = s.Namespace
	out.Name = s.Name
	out.Path = pointer.String(path)
	out.Port = s.Port

	return
}

func (s ServiceReference) ToadmissionRegistrationv1WithPath(path string) (out v1.ServiceReference) {
	out.Namespace = s.Namespace
	out.Name = s.Name
	out.Path = pointer.String(path)
	out.Port = s.Port

	return
}
