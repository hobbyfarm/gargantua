package translators

import (
	"github.com/hobbyfarm/mink/pkg/strategy/translation"
	"github.com/hobbyfarm/mink/pkg/types"
)

var _ translation.SimpleTranslator = (*SetNamespaceTranslator)(nil)

type SetNamespaceTranslator struct {
	namespace string
}

func NewSetNamespaceTranslator(namespace string) SetNamespaceTranslator {
	return SetNamespaceTranslator{
		namespace: namespace,
	}
}

func (s SetNamespaceTranslator) FromPublic(obj types.Object) types.Object {
	obj.SetNamespace(s.namespace)

	return obj
}

func (s SetNamespaceTranslator) ToPublic(obj types.Object) types.Object {
	obj.SetNamespace("") // remove it

	return obj
}
