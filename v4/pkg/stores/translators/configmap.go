package translators

import (
	"github.com/hobbyfarm/gargantua/v4/pkg/apis/hobbyfarm.io/v4alpha1"
	"github.com/hobbyfarm/mink/pkg/strategy/translation"
	"github.com/hobbyfarm/mink/pkg/types"
	corev1 "k8s.io/api/core/v1"
)

// TODO : Refactor this to not use SimpleTranslator
// Likely that is why you are running in to namespace issues
// I think you need to fully implement the translator to yield the desired result (namespaces immutability)

var _ translation.SimpleTranslator = (*ConfigMapTranslator)(nil)

type ConfigMapTranslator struct {
	Namespace string
}

func (c ConfigMapTranslator) FromPublic(obj types.Object) types.Object {
	pub := obj.(*v4alpha1.ConfigMap)
	out := new(corev1.ConfigMap)

	out.ObjectMeta = pub.ObjectMeta
	out.UID = pub.UID
	out.Namespace = c.Namespace

	out.Data = make(map[string]string)
	for k, v := range pub.Data {
		out.Data[k] = v
	}

	out.BinaryData = make(map[string][]byte)
	for k, v := range pub.BinaryData {
		out.BinaryData[k] = v
	}

	return out
}

func (c ConfigMapTranslator) ToPublic(obj types.Object) types.Object {
	priv := obj.(*corev1.ConfigMap)
	out := new(v4alpha1.ConfigMap)

	out.ObjectMeta = priv.ObjectMeta
	out.UID = priv.UID
	out.SetNamespace("")

	out.Data = make(map[string]string)
	for k, v := range priv.Data {
		out.Data[k] = v
	}

	out.BinaryData = map[string][]byte{}
	for k, v := range priv.BinaryData {
		out.BinaryData[k] = v
	}

	return out
}
