#!/bin/bash
CODEGEN=""
if [[ -z "${GOPATH}" ]]; then
  CODEGEN=$HOME/go/pkg/mod/k8s.io/code-generator@v0.20.2
else
  CODEGEN=$GOPATH/pkg/mod/k8s.io/code-generator@v0.20.2
fi

${CODEGEN}/generate-groups.sh all \
    github.com/hobbyfarm/gargantua/pkg/client \
    github.com/hobbyfarm/gargantua/pkg/apis \
    "hobbyfarm.io:v1 terraformcontroller.cattle.io:v1" \
    --go-header-file ./hack/boilerplate.go.txt
