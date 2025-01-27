package server

import (
	"context"
	"github.com/hobbyfarm/gargantua/v4/pkg/authentication"
	"github.com/hobbyfarm/gargantua/v4/pkg/authentication/authenticators"
	"github.com/hobbyfarm/gargantua/v4/pkg/authentication/authenticators/cert"
	"github.com/hobbyfarm/gargantua/v4/pkg/authentication/authenticators/token"
	"github.com/hobbyfarm/gargantua/v4/pkg/authorization"
	"github.com/hobbyfarm/gargantua/v4/pkg/openapi/hobbyfarm_io"
	"github.com/hobbyfarm/gargantua/v4/pkg/scheme"
	"github.com/hobbyfarm/gargantua/v4/pkg/stores/kubernetes"
	"github.com/hobbyfarm/gargantua/v4/pkg/stores/registry"
	"github.com/hobbyfarm/mink/pkg/serializer"
	"github.com/hobbyfarm/mink/pkg/server"
	"github.com/hobbyfarm/mink/pkg/strategy"
	"k8s.io/apiserver/pkg/registry/rest"
	apiserver "k8s.io/apiserver/pkg/server"
	"k8s.io/apiserver/pkg/server/options"
	"k8s.io/component-base/version"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type KubernetesServerConfig struct {
	Client                client.WithWatch
	ForceStorageNamespace string
	CACertBundle          string
}

// NewKubernetesServer creates a new hobbyfarm-api server backed by a remote Kubernetes cluster.
// client is a k8s client with watch capabilities.
// ForceStorageNamespace is the namespace in the remote k8s cluster in which all resources are stored.
// It is necessary to pass a namespace so that the server can force objects into the proper namespace
// and not leak resource upon querying.
func NewKubernetesServer(ctx context.Context, config *KubernetesServerConfig) (*server.Server, error) {
	v4alpha1Storage := kubernetes.V4Alpha1Storages(config.Client, config.ForceStorageNamespace)

	v4alpha1ApiGroups, err := V4Alpha1APIGroups(v4alpha1Storage)
	if err != nil {
		return nil, err
	}

	storage, err := NewKubernetesStorage(v4alpha1ApiGroups)
	if err != nil {
		return nil, err
	}

	certAuthenticatior, err := cert.NewCertAuthenticator(config.CACertBundle)
	if err != nil {
		return nil, err
	}

	// authenticator := token.NewGenericGeneratorValidator(Client)
	authenticator := authenticators.NewChainAuthenticator(
		certAuthenticatior,
		token.NewGenericGeneratorValidator(config.Client))

	authorizer := authorization.NewAuthorizer(v4alpha1Storage["rolebindings"],
		v4alpha1Storage["roles"], "/auth/.*/login")

	svr, err := server.New(&server.Config{
		Name:                         "hobbyfarm-api",
		Version:                      "v4alpha1",
		HTTPListenPort:               8080,
		HTTPSListenPort:              8443,
		LongRunningVerbs:             []string{"watch"},
		LongRunningResources:         nil,
		Scheme:                       scheme.Scheme,
		CodecFactory:                 &scheme.Codec,
		DefaultOptions:               nil,
		AuditConfig:                  nil,
		SkipInClusterLookup:          true,
		RemoteKubeConfigFileOptional: true,
		IgnoreStartFailure:           false,
		EffectiveVersion:             version.DefaultKubeEffectiveVersion(),
		Middleware:                   nil,
		Authentication: &options.DelegatingAuthenticationOptions{
			SkipInClusterLookup:          true,
			RemoteKubeConfigFileOptional: true,
			ClientCert: options.ClientCertAuthenticationOptions{
				ClientCA: config.CACertBundle,
			},
		},
		Authenticator:         authenticator,
		Authorization:         authorizer,
		OpenAPIConfig:         hobbyfarm_io.GetOpenAPIDefinitions,
		APIGroups:             storage,
		PostStartFunc:         nil,
		SupportAPIAggregation: false,
		ReadinessCheckers:     nil,
	})
	if err != nil {
		return nil, err
	}

	caches, err := authentication.SetupAuthentication(ctx, svr.GenericAPIServer.LoopbackClientConfig,
		svr.GenericAPIServer.Handler.NonGoRestfulMux)
	if err != nil {
		return nil, err
	}

	if err := svr.GenericAPIServer.AddPostStartHook("auth-caches", func(_ apiserver.PostStartHookContext) error {
		for _, cache := range caches {
			if err := cache.Start(ctx); err != nil {
				return err
			}
		}

		return nil
	}); err != nil {
		return nil, err
	}

	return svr, nil
}

func V4Alpha1APIGroups(storages map[string]strategy.CompleteStrategy) (map[string]rest.Storage, error) {
	providerStorage, err := registry.NewProviderStorage(storages["providers"],
		storages["machinesets"], storages["machines"], storages["environments"])
	if err != nil {
		return nil, err
	}

	machineTemplateStorage, err := registry.NewMachineTemplateStorage(storages["machinetemplates"])
	if err != nil {
		return nil, err
	}

	environmentStorage, err := registry.NewEnvironmentStorage(storages["environments"], storages["providers"],
		storages["machinesets"], storages["machines"], storages["scheduledevents"])
	if err != nil {
		return nil, err
	}

	environmentStatusStorage := registry.NewEnvironmentStatusStorage(storages["environments"].Scheme(), storages["environments"])

	machineSetStorage, err := registry.NewMachineSetStorage(storages["machinesets"])
	if err != nil {
		return nil, err
	}

	machineSetStatusStorage := registry.NewMachineSetStatusStorage(storages["machinesets"].Scheme(), storages["machinesets"])

	machineStorage, err := registry.NewMachineStorage(storages["machines"])
	if err != nil {
		return nil, err
	}

	machineStatusStorage := registry.NewMachineStatusStorage(storages["machines"].Scheme(), storages["machines"])

	machineClaimStorage, err := registry.NewMachineClaimStorage(storages["machineclaims"])
	if err != nil {
		return nil, err
	}

	machineClaimStatusStorage := registry.NewMachineClaimStatusStorage(storages["machineclaims"].Scheme(), storages["machineclaims"])

	scheduledEventStorage, err := registry.NewScheduledEventStorage(storages["scheduledevents"])
	if err != nil {
		return nil, err
	}

	scheduledEventStatusStorage := registry.NewScheduledEventStatusStorage(storages["scheduledevents"].Scheme(), storages["scheduledevents"])

	accessCodeStorage, err := registry.NewAccessCodeStorage(storages["accesscodes"])
	if err != nil {
		return nil, err
	}
	accessCodeStatusStorage := registry.NewAccessCodeStatusStorage(storages["accesscodes"].Scheme(), storages["accesscodes"])

	sessionStorage, err := registry.NewSessionStorage(storages["sessions"])
	if err != nil {
		return nil, err
	}

	sessionStatusStorage := registry.NewSessionStatusStorage(storages["sessions"].Scheme(), storages["sessions"])

	courseStorage, err := registry.NewCourseStorage(storages["courses"])
	if err != nil {
		return nil, err
	}

	otacStorage, err := registry.NewOneTimeAccessCodeStorage(storages["onetimeaccesscodes"])
	if err != nil {
		return nil, err
	}

	otacStatusStorage := registry.NewOneTimeAccessCodeStatusStorage(storages["onetimeaccesscodes"].Scheme(), storages["onetimeaccesscodes"])

	predefinedServiceStorage, err := registry.NewPredefinedServiceStorage(storages["predefinedservices"])
	if err != nil {
		return nil, err
	}

	progressStorage, err := registry.NewProgressStorage(storages["progresses"])
	if err != nil {
		return nil, err
	}

	progressStatusStorage := registry.NewProgressStatusStorage(storages["progresses"].Scheme(), storages["progresses"])

	scenarioStorage, err := registry.NewScenarioStorage(storages["scenarios"])
	if err != nil {
		return nil, err
	}

	scenarioStepStorage, err := registry.NewScenarioStepStorage(storages["scenariosteps"])
	if err != nil {
		return nil, err
	}

	scenarioStepStatusStorage := registry.NewScenarioStepStatusStorage(storages["scenariosteps"].Scheme(), storages["scenariosteps"])

	scopeStorage, err := registry.NewScopeStorage(storages["scopes"])
	if err != nil {
		return nil, err
	}

	settingStorage, err := registry.NewSettingStorage(storages["settings"])
	if err != nil {
		return nil, err
	}

	userStorage, err := registry.NewUserStorage(storages["users"])
	if err != nil {
		return nil, err
	}

	userStatusStorage := registry.NewUserStatusStorage(storages["users"].Scheme(), storages["users"])

	serviceAccountStorage, err := registry.NewServiceAccountStorage(storages["serviceaccounts"])
	if err != nil {
		return nil, err
	}

	configMapStorage, err := registry.NewConfigMapStorage(storages["configmaps"])
	if err != nil {
		return nil, err
	}

	secretStorage, err := registry.NewSecretStorage(storages["secrets"])
	if err != nil {
		return nil, err
	}

	roleStorage, err := registry.NewRoleStorage(storages["roles"])
	if err != nil {
		return nil, err
	}

	roleBindingStorage, err := registry.NewRoleBindingStorage(storages["rolebindings"])
	if err != nil {
		return nil, err
	}

	ldapConfigStorage, err := registry.NewLdapConfigStorage(storages["ldapconfigs"])
	if err != nil {
		return nil, err
	}

	ldapConfigStatusStorage, err := registry.NewLdapConfigStatusStorage(storages["ldapconfigs"].Scheme(), storages["ldapconfigs"])
	if err != nil {
		return nil, err
	}

	groupStorage, err := registry.NewGroupStorage(storages["groups"])
	if err != nil {
		return nil, err
	}

	otacSetStorage, err := registry.NewOneTimeAccessCodeSetStorage(storages["onetimeaccesscodesets"])
	if err != nil {
		return nil, err
	}

	otacSetStatusStorage, err := registry.NewOneTimeAccessCodeSetStatusStorage(storages["onetimeaccesscodesets"].Scheme(), storages["onetimeaccesscodesets"])
	if err != nil {
		return nil, err
	}

	stores := map[string]rest.Storage{
		"providers":                    providerStorage,
		"machinetemplates":             machineTemplateStorage,
		"environments":                 environmentStorage,
		"environments/status":          environmentStatusStorage,
		"machinesets":                  machineSetStorage,
		"machinesets/status":           machineSetStatusStorage,
		"machines":                     machineStorage,
		"machines/status":              machineStatusStorage,
		"machineclaims":                machineClaimStorage,
		"machineclaims/status":         machineClaimStatusStorage,
		"scheduledevents":              scheduledEventStorage,
		"scheduledevents/status":       scheduledEventStatusStorage,
		"accesscodes":                  accessCodeStorage,
		"accesscodes/status":           accessCodeStatusStorage,
		"sessions":                     sessionStorage,
		"sessions/status":              sessionStatusStorage,
		"courses":                      courseStorage,
		"onetimeaccesscodes":           otacStorage,
		"onetimeaccesscodes/status":    otacStatusStorage,
		"predefinedservices":           predefinedServiceStorage,
		"progresses":                   progressStorage,
		"progresses/status":            progressStatusStorage,
		"scenarios":                    scenarioStorage,
		"scenariosteps":                scenarioStepStorage,
		"scenariosteps/status":         scenarioStepStatusStorage,
		"scopes":                       scopeStorage,
		"settings":                     settingStorage,
		"users":                        userStorage,
		"users/status":                 userStatusStorage,
		"serviceaccounts":              serviceAccountStorage,
		"secrets":                      secretStorage,
		"configmaps":                   configMapStorage,
		"roles":                        roleStorage,
		"rolebindings":                 roleBindingStorage,
		"ldapconfigs":                  ldapConfigStorage,
		"ldapconfigs/status":           ldapConfigStatusStorage,
		"groups":                       groupStorage,
		"onetimeaccesscodesets":        otacSetStorage,
		"onetimeaccesscodesets/status": otacSetStatusStorage,
	}

	return stores, nil
}

func NewKubernetesStorage(
	v4alpha1Stores map[string]rest.Storage) ([]*apiserver.APIGroupInfo, error) {
	apiGroupInfo := apiserver.NewDefaultAPIGroupInfo(
		"hobbyfarm.io",
		scheme.Scheme,
		scheme.ParameterCodec,
		scheme.Codec,
	)

	apiGroupInfo.VersionedResourcesStorageMap = map[string]map[string]rest.Storage{
		"v4alpha1": v4alpha1Stores,
	}
	apiGroupInfo.NegotiatedSerializer = serializer.NewNoProtobufSerializer(apiGroupInfo.NegotiatedSerializer)

	return []*apiserver.APIGroupInfo{&apiGroupInfo}, nil
}
