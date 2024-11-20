package local

import (
	"github.com/hobbyfarm/gargantua/v4/pkg/labels"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// usernameIndexer indexes client objects based on the username existing
// in an annotation. On success it returns the username, nil slice otherwise.
func usernameIndexer(obj client.Object) []string {
	username, ok := obj.GetAnnotations()[labels.LocalUsernameKey]
	if !ok {
		return nil
	}

	return []string{username}
}
