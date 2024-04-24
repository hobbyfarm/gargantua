package kubernetes

import (
	"github.com/hobbyfarm/gargantua/v4/pkg/apis/hobbyfarm.io/v4alpha1"
	"github.com/hobbyfarm/gargantua/v4/pkg/stores/remote"
	"github.com/hobbyfarm/gargantua/v4/pkg/stores/translators"
	"github.com/hobbyfarm/mink/pkg/strategy"
	"github.com/hobbyfarm/mink/pkg/strategy/translation"
	v1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func V4Alpha1Storages(client client.WithWatch, namespace string) map[string]strategy.CompleteStrategy {
	providerRemote := remote.NewNamespaceScopedRemote(&v4alpha1.Provider{}, client, namespace)
	machineTemplateRemote := remote.NewNamespaceScopedRemote(&v4alpha1.MachineTemplate{}, client, namespace)
	environmentRemote := remote.NewNamespaceScopedRemote(&v4alpha1.Environment{}, client, namespace)
	machineSetRemote := remote.NewNamespaceScopedRemote(&v4alpha1.MachineSet{}, client, namespace)
	machineRemote := remote.NewNamespaceScopedRemote(&v4alpha1.Machine{}, client, namespace)
	machineClaimRemote := remote.NewNamespaceScopedRemote(&v4alpha1.MachineClaim{}, client, namespace)
	scheduledEventRemote := remote.NewNamespaceScopedRemote(&v4alpha1.ScheduledEvent{}, client, namespace)
	accessCodeRemote := remote.NewNamespaceScopedRemote(&v4alpha1.AccessCode{}, client, namespace)
	sessionRemote := remote.NewNamespaceScopedRemote(&v4alpha1.Session{}, client, namespace)
	courseRemote := remote.NewNamespaceScopedRemote(&v4alpha1.Course{}, client, namespace)
	otacRemote := remote.NewNamespaceScopedRemote(&v4alpha1.OneTimeAccessCode{}, client, namespace)
	predefinedServiceRemote := remote.NewNamespaceScopedRemote(&v4alpha1.PredefinedService{}, client, namespace)
	progressRemote := remote.NewNamespaceScopedRemote(&v4alpha1.Progress{}, client, namespace)
	scenarioRemote := remote.NewNamespaceScopedRemote(&v4alpha1.Scenario{}, client, namespace)
	scenarioStepRemote := remote.NewNamespaceScopedRemote(&v4alpha1.ScenarioStep{}, client, namespace)
	scopeRemote := remote.NewNamespaceScopedRemote(&v4alpha1.Scope{}, client, namespace)
	settingRemote := remote.NewNamespaceScopedRemote(&v4alpha1.Setting{}, client, namespace)
	userRemote := remote.NewNamespaceScopedRemote(&v4alpha1.User{}, client, namespace)
	serviceAccountRemote := remote.NewNamespaceScopedRemote(&v4alpha1.ServiceAccount{}, client, namespace)
	roleRemote := remote.NewNamespaceScopedRemote(&v4alpha1.Role{}, client, namespace)
	roleBindingRemote := remote.NewNamespaceScopedRemote(&v4alpha1.RoleBinding{}, client, namespace)

	configMapTranslator := translators.ConfigMapTranslator{Namespace: namespace}
	configMapRemote := translation.NewSimpleTranslationStrategy(
		configMapTranslator,
		remote.NewNamespaceScopedRemote(&v1.ConfigMap{}, client, namespace))

	secretTranslator := translators.SecretTranslator{Namespace: namespace}
	secretRemote := translation.NewSimpleTranslationStrategy(
		secretTranslator,
		remote.NewNamespaceScopedRemote(&v1.Secret{}, client, namespace))

	return map[string]strategy.CompleteStrategy{
		"providers":          providerRemote,
		"machinetemplates":   machineTemplateRemote,
		"environments":       environmentRemote,
		"machinesets":        machineSetRemote,
		"machines":           machineRemote,
		"machineclaims":      machineClaimRemote,
		"scheduledevents":    scheduledEventRemote,
		"accesscodes":        accessCodeRemote,
		"sessions":           sessionRemote,
		"courses":            courseRemote,
		"onetimeaccesscodes": otacRemote,
		"predefinedservices": predefinedServiceRemote,
		"progresses":         progressRemote,
		"scenarios":          scenarioRemote,
		"scenariosteps":      scenarioStepRemote,
		"scopes":             scopeRemote,
		"settings":           settingRemote,
		"users":              userRemote,
		"serviceaccounts":    serviceAccountRemote,
		"configmaps":         configMapRemote,
		"secrets":            secretRemote,
		"roles":              roleRemote,
		"rolebindings":       roleBindingRemote,
	}
}
