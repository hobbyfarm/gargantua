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
	v1 "github.com/hobbyfarm/gargantua/v3/pkg/apis/hobbyfarm.io/v1"
	hobbyfarmiov1 "github.com/hobbyfarm/gargantua/v3/pkg/client/clientset/versioned/typed/hobbyfarm.io/v1"
	gentype "k8s.io/client-go/gentype"
)

// fakeAccessCodes implements AccessCodeInterface
type fakeAccessCodes struct {
	*gentype.FakeClientWithList[*v1.AccessCode, *v1.AccessCodeList]
	Fake *FakeHobbyfarmV1
}

func newFakeAccessCodes(fake *FakeHobbyfarmV1, namespace string) hobbyfarmiov1.AccessCodeInterface {
	return &fakeAccessCodes{
		gentype.NewFakeClientWithList[*v1.AccessCode, *v1.AccessCodeList](
			fake.Fake,
			namespace,
			v1.SchemeGroupVersion.WithResource("accesscodes"),
			v1.SchemeGroupVersion.WithKind("AccessCode"),
			func() *v1.AccessCode { return &v1.AccessCode{} },
			func() *v1.AccessCodeList { return &v1.AccessCodeList{} },
			func(dst, src *v1.AccessCodeList) { dst.ListMeta = src.ListMeta },
			func(list *v1.AccessCodeList) []*v1.AccessCode { return gentype.ToPointerSlice(list.Items) },
			func(list *v1.AccessCodeList, items []*v1.AccessCode) { list.Items = gentype.FromPointerSlice(items) },
		),
		fake,
	}
}
