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
