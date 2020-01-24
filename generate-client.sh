#!/bin/bash
./vendor/k8s.io/code-generator/generate-groups.sh all \
    github.com/hobbyfarm/gargantua/pkg/client \
    github.com/hobbyfarm/gargantua/pkg/apis \
    "hobbyfarm.io:v1 terraformcontroller.cattle.io:v1" \
    --go-header-file ./hack/boilerplate.go.txt
