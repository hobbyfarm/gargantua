package v4alpha1

import (
	"github.com/hobbyfarm/mink/pkg/authz/binding"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/apiserver/pkg/authorization/authorizer"
	"strings"
)

const (
	all = "*"
)

var (
	All               = []string{all}
	AllSet            = sets.New(All...)
	DefaultReadVerbs  = []string{"get", "list", "watch"}
	DefaultWriteVerbs = []string{"get", "list", "watch", "create", "update", "delete", "patch"}
)

var _ binding.Rule = (*Rule)(nil)

type Rule struct {
	APIGroups     []string `json:"apiGroups"`
	Resources     []string `json:"resources"`
	SubResources  []string `json:"subResources"`
	ResourceNames []string `json:"resourceNames"`
	Verbs         []string `json:"verbs"`
	Paths         []string `json:"paths"`
}

func Matches(str string, allowed []string) bool {
	for _, allow := range allowed {
		if allow == all || str == allow {
			return true
		}
		if strings.HasSuffix(allow, all) && strings.HasPrefix(str, allow[:len(allow)-len(all)]) {
			return true
		}
	}

	return false
}

func (r Rule) Matches(attr authorizer.Attributes) bool {
	if !attr.IsResourceRequest() {
		return Matches(attr.GetPath(), r.Paths)
	}
	if len(r.SubResources) > 0 && !Matches(attr.GetSubresource(), r.SubResources) {
		return false
	}
	if len(r.SubResources) == 0 && len(attr.GetSubresource()) > 0 && !Matches(all, r.Resources) {
		return false
	}
	if len(r.ResourceNames) > 0 && !Matches(attr.GetName(), r.ResourceNames) {
		return false
	}
	return Matches(attr.GetNamespace(), r.GetNamespaces()) &&
		Matches(attr.GetVerb(), r.Verbs) &&
		Matches(attr.GetAPIGroup(), r.APIGroups) &&
		Matches(attr.GetResource(), r.Resources)
}

func (r Rule) GetNamespaces() []string {
	return []string{all}
}

func (r Rule) GetAPIGroups() []string {
	return r.APIGroups
}

func (r Rule) GetResources() []string {
	return r.Resources
}

func (r Rule) GetSubResources() []string {
	return r.SubResources
}

func (r Rule) GetResourceNames() []string {
	return r.ResourceNames
}

func (r Rule) GetVerbs() []string {
	return r.Verbs
}

func (r Rule) GetPaths() []string {
	return r.Paths
}
