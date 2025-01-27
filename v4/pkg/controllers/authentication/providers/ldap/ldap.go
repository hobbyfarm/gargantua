package ldap

import (
	"context"
	"fmt"
	"github.com/go-ldap/ldap/v3"
	"github.com/hobbyfarm/gargantua/v4/pkg/apis/hobbyfarm.io/v4alpha1"
	"github.com/hobbyfarm/gargantua/v4/pkg/factoryhelpers"
	"github.com/hobbyfarm/gargantua/v4/pkg/genericcondition"
	"github.com/rancher/lasso/pkg/controller"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"log/slog"
)

func RegisterHandlers(factory controller.SharedControllerFactory) error {
	ldapConfigController, err := factory.ForObject(&v4alpha1.LdapConfig{})
	if err != nil {
		return err
	}

	secretClient, err := factoryhelpers.ClientForObject(&v4alpha1.Secret{}, factory)
	if err != nil {
		return err
	}

	ldapConfigController.RegisterHandler(context.TODO(), "process-ldap-configs", controller.SharedControllerHandlerFunc(func(key string, obj runtime.Object) (runtime.Object, error) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		lc, ok := obj.(*v4alpha1.LdapConfig)
		if !ok {
			// TODO - Add observability here
			return nil, fmt.Errorf("invalid object")
		}

		if len(lc.Status.Conditions) == 0 {
			lc.Status.Conditions = make(map[string]genericcondition.GenericCondition, 1)
		}

		var cond genericcondition.GenericCondition
		if cond, ok = lc.Status.Conditions[v4alpha1.ConditionBindSuccessful]; !ok {
			cond = genericcondition.GenericCondition{}
		}

		// attempt dialing
		conn, err := ldap.DialURL("ldap://" + lc.Spec.LdapHost)
		if err != nil {
			cond.ChangeCondition(corev1.ConditionFalse, "ldap dial failed", err.Error())
			lc.Status.Conditions[v4alpha1.ConditionBindSuccessful] = cond
			err = ldapConfigController.Client().UpdateStatus(ctx, lc.Namespace, lc, lc, metav1.UpdateOptions{})
			if err != nil {
				slog.Error("writing updated status for ldapConfig",
					"ldapConfig", lc.Name, "error", err.Error())
				return nil, err
			}
			return lc, nil
		}

		// conn successful, check bind
		// first get secret
		var sec = &v4alpha1.Secret{}
		if err = secretClient.Get(ctx, lc.Namespace, lc.Spec.BindPasswordSecret,
			sec, metav1.GetOptions{}); err != nil {
			cond.ChangeCondition(corev1.ConditionFalse, "could not retrieve bind password secret", err.Error())
			lc.Status.Conditions[v4alpha1.ConditionBindSuccessful] = cond
			err = ldapConfigController.Client().UpdateStatus(ctx, lc.Namespace, lc, lc, metav1.UpdateOptions{})
			if err != nil {
				slog.Error("writing updated status for ldapConfig",
					"ldapConfig", lc.Name, "error", err.Error())
				return nil, err
			}
			return lc, nil
		}

		// attempt bind
		var pw = string(sec.Data["password"])
		if err = conn.Bind(lc.Spec.BindUsername, pw); err != nil {
			cond.ChangeCondition(corev1.ConditionFalse, "ldap bind failed", err.Error())
			lc.Status.Conditions[v4alpha1.ConditionBindSuccessful] = cond
			err = ldapConfigController.Client().UpdateStatus(ctx, lc.Namespace, lc, lc, metav1.UpdateOptions{})
			if err != nil {
				slog.Error("writing updated status for ldapConfig",
					"ldapConfig", lc.Name, "error", err.Error())
				return nil, err
			}
			return lc, nil
		}

		// TODO - Should there be more testing here? e.g. ldap search, etc
		// success
		cond.ChangeCondition(corev1.ConditionTrue, "ldap bind succeeded", "success")
		lc.Status.Conditions[v4alpha1.ConditionBindSuccessful] = cond
		if err = ldapConfigController.Client().UpdateStatus(ctx, lc.Namespace, lc, lc, metav1.UpdateOptions{}); err != nil {
			slog.Error("writing updated status for ldapConfig",
				"ldapConfig", lc.Name, "error", err.Error())
			return nil, err
		}

		return lc, nil
	}))

	return nil
}
