package accesscode

import (
	"github.com/hobbyfarm/gargantua/v4/pkg/apis/hobbyfarm.io/v4alpha1"
	"github.com/hobbyfarm/gargantua/v4/pkg/factoryhelpers"
	"github.com/rancher/lasso/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type accessCodeController struct {
	roleClient client.Client
}

func NewAccessCodeController(factory controller.SharedControllerFactory) error {
	roleClient, err := factoryhelpers.ClientForObject(&v4alpha1.Role{}, factory)
	if err != nil {
		return err
	}

}

func (acc *accessCodeController) RegisterHandlers() {

}
