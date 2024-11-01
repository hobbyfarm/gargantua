package server

import (
	"github.com/hobbyfarm/gargantua/v4/pkg/authentication"
	"github.com/hobbyfarm/gargantua/v4/pkg/authentication/token"
	"github.com/hobbyfarm/gargantua/v4/pkg/authorization"
	"github.com/hobbyfarm/gargantua/v4/pkg/openapi/hobbyfarm_io"
	"github.com/hobbyfarm/gargantua/v4/pkg/scheme"
	"github.com/hobbyfarm/gargantua/v4/pkg/stores/kubernetes"
	"github.com/hobbyfarm/gargantua/v4/pkg/stores/registry"
	"github.com/hobbyfarm/mink/pkg/serializer"
	"github.com/hobbyfarm/mink/pkg/server"
	"github.com/hobbyfarm/mink/pkg/strategy"
	"k8s.io/apimachinery/pkg/version"
	"k8s.io/apiserver/pkg/registry/rest"
	apiserver "k8s.io/apiserver/pkg/server"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// NewKubernetesServer creates a new hobbyfarm-api server backed by a remote Kubernetes cluster.
// client is a k8s client with watch capabilities.
// forceStorageNamespace is the namespace in the remote k8s cluster in which all resources are stored.
// It is necessary to pass a namespace so that the server can force objects into the proper namespace
// and not leak resource upon querying.
func NewKubernetesServer(kclient client.WithWatch, forceStorageNamespace string) (*server.Server, error) {
	v4alpha1Storage := kubernetes.V4Alpha1Storages(kclient, forceStorageNamespace)

	v4alpha1ApiGroups, err := V4Alpha1APIGroups(v4alpha1Storage)
	if err != nil {
		return nil, err
	}

	storage, err := NewKubernetesStorage(v4alpha1ApiGroups)
	if err != nil {
		return nil, err
	}

	authenticator := token.NewGenericGeneratorValidator(kclient)

	authorizer := authorization.NewAuthorizer(v4alpha1Storage["rolebindings"],
		v4alpha1Storage["roles"], "/auth/.*/login")

	if err != nil {
		return nil, err
	}
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
		Middleware:                   nil,
		Authenticator:                authenticator,
		Authorization:                authorizer,
		OpenAPIConfig:                hobbyfarm_io.GetOpenAPIDefinitions,
		APIGroups:                    storage,
		PostStartFunc:                nil,
		SupportAPIAggregation:        false,
		ReadinessCheckers:            nil,
	})
	if err != nil {
		return nil, err
	}

	// TODO - A cache can be setup here by passing options to client.Options
	// May want to investigate that if auth becomes bound by the API
	authClient, err := client.New(svr.GenericAPIServer.LoopbackClientConfig, client.Options{
		Scheme: scheme.Scheme,
	})
	if err != nil {
		return nil, err
	}
	authentication.RegisterHandlers(authClient, svr.GenericAPIServer.Handler.NonGoRestfulMux)

	svr.GenericAPIServer.Version = &version.Info{
		Major: "4",
		Minor: "0.0-dev",
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

	machineSetStorage, err := registry.NewMachineSetStorage(storages["machinesets"])
	if err != nil {
		return nil, err
	}

	machineStorage, err := registry.NewMachineStorage(storages["machines"])
	if err != nil {
		return nil, err
	}

	machineClaimStorage, err := registry.NewMachineClaimStorage(storages["machineclaims"])
	if err != nil {
		return nil, err
	}

	scheduledEventStorage, err := registry.NewScheduledEventStorage(storages["scheduledevents"])
	if err != nil {
		return nil, err
	}

	accessCodeStorage, err := registry.NewAccessCodeStorage(storages["accesscodes"])
	if err != nil {
		return nil, err
	}

	sessionStorage, err := registry.NewSessionStorage(storages["sessions"])
	if err != nil {
		return nil, err
	}

	courseStorage, err := registry.NewCourseStorage(storages["courses"])
	if err != nil {
		return nil, err
	}

	otacStorage, err := registry.NewOneTimeAccessCodeStorage(storages["onetimeaccesscodes"])
	if err != nil {
		return nil, err
	}

	predefinedServiceStorage, err := registry.NewPredefinedServiceStorage(storages["predefinedservices"])
	if err != nil {
		return nil, err
	}

	progressStorage, err := registry.NewProgressStorage(storages["progresses"])
	if err != nil {
		return nil, err
	}

	scenarioStorage, err := registry.NewScenarioStorage(storages["scenarios"])
	if err != nil {
		return nil, err
	}

	scenarioStepStorage, err := registry.NewScenarioStepStorage(storages["scenariosteps"])
	if err != nil {
		return nil, err
	}

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

	stores := map[string]rest.Storage{
		"providers":          providerStorage,
		"machinetemplates":   machineTemplateStorage,
		"environments":       environmentStorage,
		"machinesets":        machineSetStorage,
		"machines":           machineStorage,
		"machineclaims":      machineClaimStorage,
		"scheduledevents":    scheduledEventStorage,
		"accesscodes":        accessCodeStorage,
		"sessions":           sessionStorage,
		"courses":            courseStorage,
		"onetimeaccesscodes": otacStorage,
		"predefinedservices": predefinedServiceStorage,
		"progresses":         progressStorage,
		"scenarios":          scenarioStorage,
		"scenariosteps":      scenarioStepStorage,
		"scopes":             scopeStorage,
		"settings":           settingStorage,
		"users":              userStorage,
		"serviceaccounts":    serviceAccountStorage,
		"secrets":            secretStorage,
		"configmaps":         configMapStorage,
		"roles":              roleStorage,
		"rolebindings":       roleBindingStorage,
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
