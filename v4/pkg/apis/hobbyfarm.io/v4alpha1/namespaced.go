package v4alpha1

import "github.com/hobbyfarm/mink/pkg/strategy"

var _ strategy.NamespaceScoper = (*Namespaced)(nil)
var _ strategy.NamespaceScoper = (*NonNamespaced)(nil)

type Namespaced struct{}

func (n Namespaced) NamespaceScoped() bool {
	return true
}

type NonNamespaced struct{}

func (n NonNamespaced) NamespaceScoped() bool {
	return false
}
