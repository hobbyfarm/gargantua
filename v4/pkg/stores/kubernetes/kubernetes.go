package kubernetes

import (
	"github.com/hobbyfarm/gargantua/v4/pkg/apis/hobbyfarm.io/v4alpha1"
	"github.com/hobbyfarm/gargantua/v4/pkg/scheme"
	"github.com/hobbyfarm/gargantua/v4/pkg/stores/registry"
	"github.com/hobbyfarm/mink/pkg/serializer"
	remote "github.com/hobbyfarm/mink/pkg/strategy/remote"
	"k8s.io/apiserver/pkg/registry/rest"
	"k8s.io/apiserver/pkg/server"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func NewKubernetesStorage(client client.WithWatch) (*server.APIGroupInfo, error) {
	stores, err := APIGroups(client)
	if err != nil {
		return nil, err
	}

	apiGroupInfo := server.NewDefaultAPIGroupInfo(
		"hobbyfarm.io",
		scheme.Scheme,
		scheme.ParameterCodec,
		scheme.Codec,
	)

	apiGroupInfo.VersionedResourcesStorageMap = map[string]map[string]rest.Storage{
		"v4alpha1": stores,
	}
	apiGroupInfo.NegotiatedSerializer = serializer.NewNoProtobufSerializer(apiGroupInfo.NegotiatedSerializer)

	return &apiGroupInfo, nil
}

func APIGroups(client client.WithWatch) (map[string]rest.Storage, error) {
	providerRemote := remote.NewRemote(&v4alpha1.Provider{}, client)
	machineTemplateRemote := remote.NewRemote(&v4alpha1.MachineTemplate{}, client)
	environmentRemote := remote.NewRemote(&v4alpha1.Environment{}, client)
	machineSetRemote := remote.NewRemote(&v4alpha1.MachineSet{}, client)
	machineRemote := remote.NewRemote(&v4alpha1.Machine{}, client)
	machineClaimRemote := remote.NewRemote(&v4alpha1.MachineClaim{}, client)
	scheduledEventRemote := remote.NewRemote(&v4alpha1.ScheduledEvent{}, client)
	accessCodeRemote := remote.NewRemote(&v4alpha1.AccessCode{}, client)
	sessionRemote := remote.NewRemote(&v4alpha1.Session{}, client)

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

	stores := map[string]rest.Storage{
		"providers":        providerStorage,
		"machinetemplates": machineTemplateStorage,
		"environments":     environmentStorage,
		"machinesets":      machineSetStorage,
		"machines":         machineStorage,
		"machineclaims":    machineClaimStorage,
		"scheduledevents":  scheduledEventStorage,
		"accesscodes":      accessCodeStorage,
		"sessions":         sessionStorage,
	}

	return stores, nil
}
