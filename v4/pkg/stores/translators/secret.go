package translators

import (
	"github.com/hobbyfarm/gargantua/v4/pkg/apis/hobbyfarm.io/v4alpha1"
	"github.com/hobbyfarm/mink/pkg/strategy/translation"
	"github.com/hobbyfarm/mink/pkg/types"
	corev1 "k8s.io/api/core/v1"
)

var _ translation.SimpleTranslator = (*SecretTranslator)(nil)

type SecretTranslator struct {
	Namespace string
}

func (s SecretTranslator) FromPublic(obj types.Object) types.Object {
	pub := obj.(*v4alpha1.Secret)
	out := new(corev1.Secret)

	out.ObjectMeta = pub.ObjectMeta
	out.UID = pub.UID
	out.SetNamespace(s.Namespace)

	out.Data = make(map[string][]byte)
	for k, v := range pub.Data {
		out.Data[k] = v
	}

	out.StringData = make(map[string]string)
	for k, v := range pub.StringData {
		out.StringData[k] = v
	}

	out.Type = corev1.SecretType(pub.Type)

	return out
}

func (s SecretTranslator) ToPublic(obj types.Object) types.Object {
	priv := obj.(*corev1.Secret)
	out := new(v4alpha1.Secret)

	out.ObjectMeta = priv.ObjectMeta
	out.UID = priv.UID
	out.SetNamespace("")

	out.Data = make(map[string][]byte)
	for k, v := range priv.Data {
		out.Data[k] = v
	}

	out.StringData = make(map[string]string)
	for k, v := range priv.StringData {
		out.StringData[k] = v
	}

	out.Type = string(priv.Type)

	return out
}
