package accesscode

import (
	"context"
	"github.com/hobbyfarm/gargantua/v4/pkg/apis/hobbyfarm.io/v4alpha1"
	"github.com/hobbyfarm/gargantua/v4/pkg/factoryhelpers"
	"github.com/rancher/lasso/pkg/client"
	"github.com/rancher/lasso/pkg/controller"
)

type accessCodeController struct {
	roleClient *client.Client

	accessCodeController controller.SharedController
	otacController       controller.SharedController
}

func RegisterHandlers(factory controller.SharedControllerFactory) error {
	roleClient, err := factoryhelpers.ClientForObject(&v4alpha1.Role{}, factory)
	if err != nil {
		return err
	}

	acController, err := factory.ForObject(&v4alpha1.AccessCode{})
	if err != nil {
		return err
	}

	otacController, err := factory.ForObject(&v4alpha1.OneTimeAccessCode{})
	if err != nil {
		return err
	}

	acc := &accessCodeController{
		roleClient:           roleClient,
		accessCodeController: acController,
		otacController:       otacController,
	}

	acController.RegisterHandler(context.TODO(), "access-code-ensure-role",
		controller.SharedControllerHandlerFunc(acc.ensureRole))

	otacController.RegisterHandler(context.TODO(), "otac-ensure-role",
		controller.SharedControllerHandlerFunc(acc.ensureRole))

	return nil
}
