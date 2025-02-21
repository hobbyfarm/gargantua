//go:generate go run ./v4/cmd/deepcopy ./v4/pkg/apis/hobbyfarm.io/v4alpha1
//go:generate go run ./v4/cmd/deepcopy ./v4/pkg/genericcondition
//go:generate go run k8s.io/kube-openapi/cmd/openapi-gen --output-dir ./v4/pkg/openapi/hobbyfarm_io  --output-pkg /v4/pkg/openapi/hobbyfarm_io --go-header-file hack/boilerplate.go.txt github.com/hobbyfarm/gargantua/v4/pkg/apis/hobbyfarm.io/v4alpha1 github.com/hobbyfarm/gargantua/v4/pkg/genericcondition github.com/hobbyfarm/gargantua/v4/pkg/property k8s.io/apimachinery/pkg/apis/meta/v1 k8s.io/apimachinery/pkg/runtime k8s.io/apimachinery/pkg/version k8s.io/apimachinery/pkg/api/resource k8s.io/api/core/v1 k8s.io/api/rbac/v1 k8s.io/apimachinery/pkg/util/intstr
//go:generate echo "==== If you see 'API rule violation' errors, it is safe to ignore them! ==="

package main

import (
	_ "k8s.io/kube-openapi/cmd/openapi-gen/args"
)
