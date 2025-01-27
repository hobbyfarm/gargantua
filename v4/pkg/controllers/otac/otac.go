package otac

import (
	"context"
	"fmt"
	"github.com/hobbyfarm/gargantua/v4/pkg/apis/hobbyfarm.io/v4alpha1"
	"github.com/hobbyfarm/gargantua/v4/pkg/factoryhelpers"
	"github.com/hobbyfarm/gargantua/v4/pkg/genericcondition"
	labels2 "github.com/hobbyfarm/gargantua/v4/pkg/labels"
	"github.com/rancher/lasso/pkg/client"
	"github.com/rancher/lasso/pkg/controller"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/utils/ptr"
)

type otacController struct {
	otacClient        *client.Client
	otacSetController controller.SharedController
}

func RegisterHandlers(factory controller.SharedControllerFactory) error {
	otacSetController, err := factory.ForObject(&v4alpha1.OneTimeAccessCodeSet{})
	if err != nil {
		return err
	}

	otacClient, err := factoryhelpers.ClientForObject(&v4alpha1.OneTimeAccessCode{}, factory)
	if err != nil {
		return err
	}

	oc := &otacController{
		otacClient: otacClient,
	}

	otacSetController.RegisterHandler(context.TODO(), "otac-set-handler",
		controller.SharedControllerHandlerFunc(oc.otacSetHandler))

	return nil
}

func (oc *otacController) otacSetHandler(key string, obj runtime.Object) (runtime.Object, error) {
	otacSet, ok := obj.(*v4alpha1.OneTimeAccessCodeSet)
	if !ok {
		return obj, nil
	}

	if otacSet.DeletionTimestamp != nil {
		return oc.handleDelete(context.Background(), otacSet)
	}

	genericcondition.CreateIfNot(obj, v4alpha1.DesiredOTACAmount)
	if err := oc.otacClient.UpdateStatus(context.TODO(), "", otacSet, otacSet, v1.UpdateOptions{}); err != nil {
		return obj, err
	}

	return oc.reconcile(context.Background(), otacSet)
}

func (oc *otacController) handleDelete(ctx context.Context, otacSet *v4alpha1.OneTimeAccessCodeSet) (runtime.Object, error) {
	// OTACset is to be removed, we should clean up the OTACs

	if err := oc.otacClient.DeleteCollection(ctx, "", v1.DeleteOptions{
		PropagationPolicy: ptr.To(v1.DeletePropagationBackground),
	}, v1.ListOptions{
		LabelSelector: fmt.Sprintf("%s=%s", labels2.OneTimeAccessCodeSetLabel, otacSet.GetName()),
	}); err != nil {
		return nil, err
	}

	return otacSet, nil
}

func (oc *otacController) reconcile(ctx context.Context, otacSet *v4alpha1.OneTimeAccessCodeSet) (runtime.Object, error) {
	// list existing OTACs
	var otacList = &v4alpha1.OneTimeAccessCodeList{}
	if err := oc.otacClient.List(ctx, "", otacList, v1.ListOptions{
		LabelSelector: fmt.Sprintf("%s=%s", labels2.OneTimeAccessCodeSetLabel, otacSet.GetName()),
	}); err != nil {
		return nil, err
	}

	// how many we got?
	otacSet.Status.Created = len(otacList.Items)
	if err := oc.otacClient.UpdateStatus(ctx, "", otacSet, otacSet, v1.UpdateOptions{}); err != nil {
		return nil, err
	}

	// just enough
	if otacSet.Status.Created == otacSet.Spec.Count {
		genericcondition.Update(otacSet, v4alpha1.DesiredOTACAmount, corev1.ConditionTrue,
			"otacs created", "")
		return otacSet, nil
	}

	// ... not enough
	if otacSet.Status.Created < otacSet.Spec.Count {
		// create some new ones
		for range len(otacList.Items) - otacSet.Status.Created {
			newOtac := &v4alpha1.OneTimeAccessCode{
				ObjectMeta: v1.ObjectMeta{
					GenerateName: "otacset-",
					Labels: map[string]string{
						labels2.OneTimeAccessCodeSetLabel: otacSet.GetName(),
					},
				},
				Spec: otacSet.Spec.Template,
			}

			err := oc.otacClient.Create(ctx, "", newOtac, newOtac, v1.CreateOptions{})
			if err != nil {
				genericcondition.Update(otacSet, v4alpha1.DesiredOTACAmount, corev1.ConditionFalse,
					"error creating OTACs", err.Error())
				if upErr := oc.otacClient.UpdateStatus(ctx, "", otacSet, otacSet, v1.UpdateOptions{}); upErr != nil {
					return nil, upErr
				}
				return nil, err
			}
		}

		return otacSet, nil
	}

	// too many
	if otacSet.Status.Created > otacSet.Spec.Count {
		// let's attempt to delete some candidates
		if err := oc.otacClient.DeleteCollection(ctx, "", v1.DeleteOptions{}, v1.ListOptions{
			LabelSelector: fmt.Sprintf("%s=%s,%s=%s", labels2.OneTimeAccessCodeSetLabel, otacSet.GetName(),
				labels2.OneTimeAccessCodeRedeemedLabel, "False"),
			Limit: int64(otacSet.Spec.Count - otacSet.Status.Created),
		}); err != nil {
			genericcondition.Update(otacSet, v4alpha1.DesiredOTACAmount, corev1.ConditionFalse,
				"error deleting OTACs", err.Error())
		}

		// did we delete enough?
		// list again
		if err := oc.otacClient.List(ctx, "", otacList, v1.ListOptions{
			LabelSelector: fmt.Sprintf("%s=%s", labels2.OneTimeAccessCodeSetLabel, otacSet.GetName()),
		}); err != nil {
			return nil, err
		}

		if len(otacList.Items) > otacSet.Spec.Count {
			// nope we did not, let the user know
			genericcondition.Update(otacSet, v4alpha1.DesiredOTACAmount, corev1.ConditionFalse,
				"could not delete enough OTACs", "attempted to delete OTACs to meet spec.count, but could not "+
					"delete enough. will not delete redeemed OTACs.")
			if err := oc.otacClient.UpdateStatus(ctx, "", otacSet, otacSet, v1.UpdateOptions{}); err != nil {
				return nil, err
			}
		} else {
			genericcondition.Update(otacSet, v4alpha1.DesiredOTACAmount, corev1.ConditionTrue,
				"OTACs at desired count", "")
			if err := oc.otacClient.UpdateStatus(ctx, "", otacSet, otacSet, v1.UpdateOptions{}); err != nil {
				return nil, err
			}
		}

		return otacSet, nil
	}

	return nil, nil
}
