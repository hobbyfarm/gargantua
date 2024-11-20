package authentication

import (
	"context"
	mux2 "github.com/gorilla/mux"
	"github.com/hobbyfarm/gargantua/v4/pkg/apis/hobbyfarm.io/v4alpha1"
	"github.com/hobbyfarm/gargantua/v4/pkg/authentication/authenticators/token"
	"github.com/hobbyfarm/gargantua/v4/pkg/authentication/group"
	"github.com/hobbyfarm/gargantua/v4/pkg/authentication/providers/ldap"
	"github.com/hobbyfarm/gargantua/v4/pkg/authentication/providers/local"
	"github.com/hobbyfarm/gargantua/v4/pkg/gvkr"
	"github.com/hobbyfarm/gargantua/v4/pkg/scheme"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime/schema"
	kmux "k8s.io/apiserver/pkg/server/mux"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type HasIndexers interface {
	Indexers() map[string]client.IndexerFunc
}

func SetupAuthentication(ctx context.Context, cfg *rest.Config, mux *kmux.PathRecorderMux) ([]cache.Cache, error) {
	kclient, err := client.New(cfg, client.Options{
		Scheme: scheme.Scheme,
	})
	if err != nil {
		return nil, err
	}

	genericTokenGV := token.NewGenericGeneratorValidator(kclient)

	authRouter := mux2.NewRouter().PathPrefix("/auth/").Subrouter()
	mux.HandlePrefix("/auth/", authRouter)

	userCache, err := setupUserCache(ctx, cfg)
	if err != nil {
		return nil, err
	}

	local.New(kclient, userCache, genericTokenGV, authRouter.PathPrefix("/local/").Subrouter())
	ldap.New(kclient, userCache, genericTokenGV, authRouter.PathPrefix("/ldap/").Subrouter())

	return []cache.Cache{userCache}, nil
}

func setupUserCache(ctx context.Context, cfg *rest.Config) (cache.Cache, error) {
	userGroupCache, err := cache.New(cfg, cache.Options{
		Scheme: scheme.Scheme,
		Mapper: buildMapper(),
	})
	if err != nil {
		return nil, err
	}

	indexers := []map[string]client.IndexerFunc{
		ldap.Indexers(),
		local.Indexers(),
	}

	for _, v := range indexers {
		for k, vv := range v {
			if err := userGroupCache.IndexField(ctx, &v4alpha1.User{}, k, vv); err != nil {
				return nil, err
			}
		}
	}

	if err := userGroupCache.IndexField(ctx, &v4alpha1.Group{}, "group-user-members", group.GroupUserMemberIndexer); err != nil {
		return nil, err
	}

	if err := userGroupCache.IndexField(ctx, &v4alpha1.Group{}, "group-provider-members-ldap", group.GroupProviderIndexer("ldap")); err != nil {
		return nil, err
	}

	return userGroupCache, nil
}

func buildMapper() meta.RESTMapper {
	rm := meta.NewDefaultRESTMapper([]schema.GroupVersion{
		{Group: v4alpha1.APIGroup, Version: v4alpha1.Version},
	})

	gvk, gvrSingular, gvrPlural := gvkr.GVKR(v4alpha1.APIGroup, v4alpha1.Version, "User", "users")
	rm.AddSpecific(gvk, gvrSingular, gvrPlural, meta.RESTScopeRoot)

	gvk, gvrSingular, gvrPlural = gvkr.GVKR(v4alpha1.APIGroup, v4alpha1.Version, "Group", "groups")
	rm.AddSpecific(gvk, gvrSingular, gvrPlural, meta.RESTScopeRoot)

	return rm
}
