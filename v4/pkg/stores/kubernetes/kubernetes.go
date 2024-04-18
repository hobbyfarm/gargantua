package kubernetes

import (
	v1 "github.com/hobbyfarm/gargantua/v3/pkg/apis/hobbyfarm.io/v1"
	"github.com/hobbyfarm/gargantua/v4/pkg/apis/hobbyfarm.io/v4alpha1"
	"github.com/hobbyfarm/gargantua/v4/pkg/scheme"
	"github.com/hobbyfarm/gargantua/v4/pkg/stores/registry"
	"github.com/hobbyfarm/mink/pkg/serializer"
	remote "github.com/hobbyfarm/mink/pkg/strategy/remote"
	v12 "k8s.io/api/core/v1"
	"k8s.io/apiserver/pkg/registry/rest"
	"k8s.io/apiserver/pkg/server"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func NewKubernetesStorage(client client.WithWatch) ([]*server.APIGroupInfo, error) {
	v4alpha1Stores, err := V4Alpha1APIGroups(client)
	if err != nil {
		return nil, err
	}

	corev1stores, err := CoreV1APIGroups(client)
	if err != nil {
		return nil, err
	}

	corev1apigroupinfo := server.NewDefaultAPIGroupInfo(
		"",
		scheme.Scheme,
		scheme.ParameterCodec,
		scheme.Codec,
	)

	apiGroupInfo := server.NewDefaultAPIGroupInfo(
		"hobbyfarm.io",
		scheme.Scheme,
		scheme.ParameterCodec,
		scheme.Codec,
	)

	apiGroupInfo.VersionedResourcesStorageMap = map[string]map[string]rest.Storage{
		"v4alpha1": v4alpha1Stores,
	}
	apiGroupInfo.NegotiatedSerializer = serializer.NewNoProtobufSerializer(apiGroupInfo.NegotiatedSerializer)

	corev1apigroupinfo.VersionedResourcesStorageMap = map[string]map[string]rest.Storage{
		"v1": corev1stores,
	}
	corev1apigroupinfo.NegotiatedSerializer = serializer.NewNoProtobufSerializer(corev1apigroupinfo.NegotiatedSerializer)

	return []*server.APIGroupInfo{&apiGroupInfo, &corev1apigroupinfo}, nil
}

func CoreV1APIGroups(client client.WithWatch) (map[string]rest.Storage, error) {
	secretRemote := remote.NewRemote(&v12.Secret{}, client)
	configMapRemote := remote.NewRemote(&v12.ConfigMap{}, client)

	configMapStorage, err := registry.NewConfigMapStorage(configMapRemote)
	if err != nil {
		return nil, err
	}

	secretStorage, err := registry.NewSecretStorage(secretRemote)
	if err != nil {
		return nil, err
	}

	stores := map[string]rest.Storage{
		"secrets":    secretStorage,
		"configmaps": configMapStorage,
	}

	return stores, nil
}

func V4Alpha1APIGroups(client client.WithWatch) (map[string]rest.Storage, error) {
	providerRemote := remote.NewRemote(&v4alpha1.Provider{}, client)
	machineTemplateRemote := remote.NewRemote(&v4alpha1.MachineTemplate{}, client)
	environmentRemote := remote.NewRemote(&v4alpha1.Environment{}, client)
	machineSetRemote := remote.NewRemote(&v4alpha1.MachineSet{}, client)
	machineRemote := remote.NewRemote(&v4alpha1.Machine{}, client)
	machineClaimRemote := remote.NewRemote(&v4alpha1.MachineClaim{}, client)
	scheduledEventRemote := remote.NewRemote(&v4alpha1.ScheduledEvent{}, client)
	accessCodeRemote := remote.NewRemote(&v1.AccessCode{}, client)
	sessionRemote := remote.NewRemote(&v4alpha1.Session{}, client)
	courseRemote := remote.NewRemote(&v4alpha1.Course{}, client)
	otacRemote := remote.NewRemote(&v4alpha1.OneTimeAccessCode{}, client)
	predefinedServiceRemote := remote.NewRemote(&v4alpha1.PredefinedService{}, client)
	progressRemote := remote.NewRemote(&v4alpha1.Progress{}, client)
	scenarioRemote := remote.NewRemote(&v4alpha1.Scenario{}, client)
	scenarioStepRemote := remote.NewRemote(&v4alpha1.ScenarioStep{}, client)
	scopeRemote := remote.NewRemote(&v4alpha1.Scope{}, client)
	settingRemote := remote.NewRemote(&v4alpha1.Setting{}, client)
	userRemote := remote.NewRemote(&v4alpha1.User{}, client)
	serviceAccountRemote := remote.NewRemote(&v4alpha1.ServiceAccount{}, client)

	providerStorage, err := registry.NewProviderStorage(providerRemote, machineSetRemote, machineRemote, environmentRemote)
	if err != nil {
		return nil, err
	}

	machineTemplateStorage, err := registry.NewMachineTemplateStorage(machineTemplateRemote)
	if err != nil {
		return nil, err
	}

	environmentStorage, err := registry.NewEnvironmentStorage(environmentRemote, providerRemote,
		machineSetRemote, machineRemote, scheduledEventRemote)
	if err != nil {
		return nil, err
	}

	machineSetStorage, err := registry.NewMachineSetStorage(machineSetRemote)
	if err != nil {
		return nil, err
	}

	machineStorage, err := registry.NewMachineStorage(machineRemote)
	if err != nil {
		return nil, err
	}

	machineClaimStorage, err := registry.NewMachineClaimStorage(machineClaimRemote)
	if err != nil {
		return nil, err
	}

	scheduledEventStorage, err := registry.NewScheduledEventStorage(scheduledEventRemote)
	if err != nil {
		return nil, err
	}

	accessCodeStorage, err := registry.NewAccessCodeStorage(accessCodeRemote)
	if err != nil {
		return nil, err
	}

	sessionStorage, err := registry.NewSessionStorage(sessionRemote)
	if err != nil {
		return nil, err
	}

	courseStorage, err := registry.NewCourseStorage(courseRemote)
	if err != nil {
		return nil, err
	}

	otacStorage, err := registry.NewOneTimeAccessCodeStorage(otacRemote)
	if err != nil {
		return nil, err
	}

	predefinedServiceStorage, err := registry.NewPredefinedServiceStorage(predefinedServiceRemote)
	if err != nil {
		return nil, err
	}

	progressStorage, err := registry.NewProgressStorage(progressRemote)
	if err != nil {
		return nil, err
	}

	scenarioStorage, err := registry.NewScenarioStorage(scenarioRemote)
	if err != nil {
		return nil, err
	}

	scenarioStepStorage, err := registry.NewScenarioStepStorage(scenarioStepRemote)
	if err != nil {
		return nil, err
	}

	scopeStorage, err := registry.NewScopeStorage(scopeRemote)
	if err != nil {
		return nil, err
	}

	settingStorage, err := registry.NewSettingStorage(settingRemote)
	if err != nil {
		return nil, err
	}

	userStorage, err := registry.NewUserStorage(userRemote)
	if err != nil {
		return nil, err
	}

	serviceAccountStorage, err := registry.NewServiceAccountStorage(serviceAccountRemote)
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
	}

	return stores, nil
}
