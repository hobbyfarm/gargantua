package serviceaccount

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"github.com/hobbyfarm/gargantua/v4/pkg/apis/hobbyfarm.io/v4alpha1"
	"github.com/hobbyfarm/gargantua/v4/pkg/factoryhelpers"
	"github.com/pkg/errors"
	"github.com/rancher/lasso/pkg/controller"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

func RegisterHandlers(factory controller.SharedControllerFactory) error {
	serviceAccountController, err := factory.ForObject(&v4alpha1.ServiceAccount{})
	if err != nil {
		return err
	}

	secretClient, err := factoryhelpers.ClientForObject(&v4alpha1.Secret{}, factory)
	if err != nil {
		return err
	}

	serviceAccountController.RegisterHandler(context.TODO(), "ensure-sa-token", controller.SharedControllerHandlerFunc(func(key string, obj runtime.Object) (runtime.Object, error) {
		sa, ok := obj.(*v4alpha1.ServiceAccount)
		if !ok {
			return nil, errors.New("invalid object")
		}

		if len(sa.Secrets) == 0 {
			pw, err := newRandomKey()
			if err != nil {
				return nil, fmt.Errorf("error while generating crypto secure rand key: %s", err.Error())
			}

			secret := &v4alpha1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					GenerateName: "sec-",
				},
				Type: "Opaque",
				Data: map[string][]byte{
					"password": []byte(pw),
				},
			}

			secretResult := &v4alpha1.Secret{}
			if err := secretClient.Create(context.TODO(), "", secret, secretResult, metav1.CreateOptions{}); err != nil {
				return nil, fmt.Errorf("error creating secret for serviceaccount %s: %s", key, err.Error())
			}

			sa.Secrets = []string{secretResult.Name}

			saResult := &v4alpha1.ServiceAccount{}
			if err := serviceAccountController.Client().Update(context.TODO(), "", sa, saResult, metav1.UpdateOptions{}); err != nil {
				return nil, fmt.Errorf("error updating serviceaccount %s: %s", key, err.Error())
			}
		}

		return sa, nil
	}))

	return nil
}

func newRandomKey() (string, error) {
	b := make([]byte, 16)
	_, err := rand.Read(b)
	if err != nil {
		return "", err
	}

	return base64.StdEncoding.EncodeToString(b), nil
}
