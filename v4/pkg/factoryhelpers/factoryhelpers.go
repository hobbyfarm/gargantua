package factoryhelpers

import (
	"github.com/rancher/lasso/pkg/client"
	"github.com/rancher/lasso/pkg/controller"
	"k8s.io/apimachinery/pkg/runtime"
)

func ClientForObject(obj runtime.Object, factory controller.SharedControllerFactory) (*client.Client, error) {
	gvk, err := factory.SharedCacheFactory().SharedClientFactory().GVKForObject(obj)
	if err != nil {
		return nil, err
	}

	return factory.SharedCacheFactory().SharedClientFactory().ForKind(gvk)
}
