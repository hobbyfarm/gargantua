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
	context "context"

	hobbyfarmiov1 "github.com/hobbyfarm/gargantua/v3/pkg/apis/hobbyfarm.io/v1"
	scheme "github.com/hobbyfarm/gargantua/v3/pkg/client/clientset/versioned/scheme"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	types "k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"
	gentype "k8s.io/client-go/gentype"
)

// PredefinedServicesGetter has a method to return a PredefinedServiceInterface.
// A group's client should implement this interface.
type PredefinedServicesGetter interface {
	PredefinedServices(namespace string) PredefinedServiceInterface
}

// PredefinedServiceInterface has methods to work with PredefinedService resources.
type PredefinedServiceInterface interface {
	Create(ctx context.Context, predefinedService *hobbyfarmiov1.PredefinedService, opts metav1.CreateOptions) (*hobbyfarmiov1.PredefinedService, error)
	Update(ctx context.Context, predefinedService *hobbyfarmiov1.PredefinedService, opts metav1.UpdateOptions) (*hobbyfarmiov1.PredefinedService, error)
	Delete(ctx context.Context, name string, opts metav1.DeleteOptions) error
	DeleteCollection(ctx context.Context, opts metav1.DeleteOptions, listOpts metav1.ListOptions) error
	Get(ctx context.Context, name string, opts metav1.GetOptions) (*hobbyfarmiov1.PredefinedService, error)
	List(ctx context.Context, opts metav1.ListOptions) (*hobbyfarmiov1.PredefinedServiceList, error)
	Watch(ctx context.Context, opts metav1.ListOptions) (watch.Interface, error)
	Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts metav1.PatchOptions, subresources ...string) (result *hobbyfarmiov1.PredefinedService, err error)
	PredefinedServiceExpansion
}

// predefinedServices implements PredefinedServiceInterface
type predefinedServices struct {
	*gentype.ClientWithList[*hobbyfarmiov1.PredefinedService, *hobbyfarmiov1.PredefinedServiceList]
}

// newPredefinedServices returns a PredefinedServices
func newPredefinedServices(c *HobbyfarmV1Client, namespace string) *predefinedServices {
	return &predefinedServices{
		gentype.NewClientWithList[*hobbyfarmiov1.PredefinedService, *hobbyfarmiov1.PredefinedServiceList](
			"predefinedservices",
			c.RESTClient(),
			scheme.ParameterCodec,
			namespace,
			func() *hobbyfarmiov1.PredefinedService { return &hobbyfarmiov1.PredefinedService{} },
			func() *hobbyfarmiov1.PredefinedServiceList { return &hobbyfarmiov1.PredefinedServiceList{} },
		),
	}
}
