package group

import (
	"github.com/hobbyfarm/gargantua/v4/pkg/apis/hobbyfarm.io/v4alpha1"
	"log/slog"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func convertGroup(obj client.Object) (*v4alpha1.Group, bool) {
	group, ok := obj.(*v4alpha1.Group)
	if !ok {
		slog.Error("converting client.Object to *v4alpha1.Group", "objectName", obj.GetName())
		return nil, false
	}

	return group, true
}

func GroupProviderIndexer(provider string) client.IndexerFunc {
	return func(object client.Object) []string {
		group, ok := convertGroup(object)
		if !ok {
			return nil
		}

		if members, ok := group.Spec.ProviderMembers[provider]; ok {
			return members
		}

		return nil
	}
}

func GroupUserMemberIndexer(obj client.Object) []string {
	group, ok := convertGroup(obj)
	if !ok {
		return nil
	}

	return group.Spec.UserMembers
}
