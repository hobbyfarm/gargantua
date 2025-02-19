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

// fakeVirtualMachineClaims implements VirtualMachineClaimInterface
type fakeVirtualMachineClaims struct {
	*gentype.FakeClientWithList[*v1.VirtualMachineClaim, *v1.VirtualMachineClaimList]
	Fake *FakeHobbyfarmV1
}

func newFakeVirtualMachineClaims(fake *FakeHobbyfarmV1, namespace string) hobbyfarmiov1.VirtualMachineClaimInterface {
	return &fakeVirtualMachineClaims{
		gentype.NewFakeClientWithList[*v1.VirtualMachineClaim, *v1.VirtualMachineClaimList](
			fake.Fake,
			namespace,
			v1.SchemeGroupVersion.WithResource("virtualmachineclaims"),
			v1.SchemeGroupVersion.WithKind("VirtualMachineClaim"),
			func() *v1.VirtualMachineClaim { return &v1.VirtualMachineClaim{} },
			func() *v1.VirtualMachineClaimList { return &v1.VirtualMachineClaimList{} },
			func(dst, src *v1.VirtualMachineClaimList) { dst.ListMeta = src.ListMeta },
			func(list *v1.VirtualMachineClaimList) []*v1.VirtualMachineClaim {
				return gentype.ToPointerSlice(list.Items)
			},
			func(list *v1.VirtualMachineClaimList, items []*v1.VirtualMachineClaim) {
				list.Items = gentype.FromPointerSlice(items)
			},
		),
		fake,
	}
}
