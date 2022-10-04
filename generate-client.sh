#!/bin/bash

$(pwd)/generate-groups.sh all \
    github.com/hobbyfarm/gargantua/pkg/client \
    github.com/hobbyfarm/gargantua/pkg/apis \
    "hobbyfarm.io:v1,v2 terraformcontroller.cattle.io:v1" \
    --go-header-file ./hack/boilerplate.go.txt
