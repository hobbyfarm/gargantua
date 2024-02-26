#!/bin/bash

if [[ "$(pwd)" != */github.com/hobbyfarm/gargantua ]]
then
  echo "You must run this script from within a path that ends in /github.com/hobbyfarm/gargantua"
  exit 1
fi

PKG_PATH=$(pwd | sed -e 's/github.com\/hobbyfarm\/gargantua//')

source kube_codegen.sh

mkdir -p v3/pkg/client/clientset
mkdir -p v3/pkg/client/listers
mkdir -p v3/pkg/client/informers

kube::codegen::gen_helpers \
  --input-pkg-root github.com/hobbyfarm/gargantua/v3/pkg/apis \
  --output-base "${PKG_PATH}"

kube::codegen::gen_client \
  --input-pkg-root github.com/hobbyfarm/gargantua/v3/pkg/apis \
  --output-pkg-root github.com/hobbyfarm/gargantua/v3/pkg/client \
  --boilerplate ./hack/boilerplate.go.txt \
  --output-base "${PKG_PATH}" \
  --clientset-name clientset \
  --versioned-name versioned \
  --listers-name listers \
  --informers-name informers \
  --with-watch

kube::codegen::gen_helpers \
  --input-pkg-root github.com/hobbyfarm/gargantua/v3/pkg/property \
  --output-base "${PKG_PATH}" 2>&1 | grep -v -e "GenericType"

# The previous command ignores any lines w/ string GenericType
# This is to suppress a warning about deepcopy-gen not being able to support a generic type
# used as a type constraint. This shouldn't affect anything and is only cosmetic for developers
# to not worry about the error.
# The error is logged below for posterity.
#
# W0523 16:50:47.161458   74104 parse.go:862] Making unsupported type entry "GenericType" for: &types.TypeParam{check:(*types.Checker)(nil), id:0x2, obj:(*types.TypeName)(0x140086f8f00), index:0, bound:(*types.Interface)(0x140000c0aa0)}
