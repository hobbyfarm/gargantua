#!/bin/bash

source kube_codegen.sh

mkdir -p pkg/client/clientset
mkdir -p pkg/client/listers
mkdir -p pkg/client/informers

kube::codegen::gen_helpers \
  --input-pkg-root github.com/hobbyfarm/gargantua/pkg/apis \
  --output-base "${GOPATH}"/src

kube::codegen::gen_client \
  --input-pkg-root github.com/hobbyfarm/gargantua/pkg/apis \
  --output-pkg-root github.com/hobbyfarm/gargantua/pkg/client \
  --boilerplate ./hack/boilerplate.go.txt \
  --output-base "${GOPATH}"/src \
  --clientset-name clientset \
  --versioned-name versioned \
  --listers-name listers \
  --informers-name informers \
  --with-watch

kube::codegen::gen_helpers \
  --input-pkg-root github.com/hobbyfarm/gargantua/pkg/property \
  --output-base "${GOPATH}"/src 2>&1 | grep -v -e "GenericType"

# The previous command ignores any lines w/ string GenericType
# This is to suppress a warning about deepcopy-gen not being able to support a generic type
# used as a type constraint. This shouldn't affect anything and is only cosmetic for developers
# to not worry about the error.
# The error is logged below for posterity.
#
# W0523 16:50:47.161458   74104 parse.go:862] Making unsupported type entry "GenericType" for: &types.TypeParam{check:(*types.Checker)(nil), id:0x2, obj:(*types.TypeName)(0x140086f8f00), index:0, bound:(*types.Interface)(0x140000c0aa0)}