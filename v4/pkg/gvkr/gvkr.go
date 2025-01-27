package gvkr

import (
	"k8s.io/apimachinery/pkg/runtime/schema"
	"strings"
)

func GVKR(group string, version string, kind string, plural string) (gvk schema.GroupVersionKind, gvrPlural schema.GroupVersionResource,
	gvrSingular schema.GroupVersionResource) {

	gvk = schema.GroupVersionKind{Group: group, Version: version, Kind: kind}
	gvrPlural = schema.GroupVersionResource{Group: group, Version: version, Resource: plural}
	gvrSingular = schema.GroupVersionResource{Group: group, Version: version, Resource: strings.ToLower(kind)}

	return
}
