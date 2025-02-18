package otac

import (
	"context"
	"github.com/hobbyfarm/gargantua/v4/pkg/apis/hobbyfarm.io/v4alpha1"
	"github.com/hobbyfarm/gargantua/v4/pkg/eventbuilder"
	"github.com/hobbyfarm/gargantua/v4/pkg/labels"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	labels2 "k8s.io/apimachinery/pkg/labels"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

const (
	OtacSetScaleControllerName = "otacset-scale-controller"
)

type otacSetScaleController struct {
	kclient client.Client
}

func (cx otacSetScaleController) Reconcile(ctx context.Context, request reconcile.Request) (reconcile.Result, error) {
	set := &v4alpha1.OneTimeAccessCodeSet{}
	if err := cx.kclient.Get(ctx, request.NamespacedName, set); err != nil {
		return reconcile.Result{}, client.IgnoreNotFound(err)
	}

	otacList := &v4alpha1.OneTimeAccessCodeList{}
	if err := cx.kclient.List(ctx, otacList); err != nil {
		return reconcile.Result{}, client.IgnoreNotFound(err)
	}

	// how many we got?
	set.Status.Created = len(otacList.Items)
	if err := cx.kclient.Status().Update(ctx, set); err != nil {
		return reconcile.Result{}, err
	}

	var reQ = false

	// ... not enough
	if set.Status.Created < set.Spec.Count {
		// issue an event
		eventbuilder.
			Info().
			For(set).
			By(OtacSetScaleControllerName, "").
			Reason("attempting to scale up otacset").
			WriteOrLog(cx.kclient)

		// create some new ones
		for range set.Spec.Count - set.Status.Created {
			newOtac := &v4alpha1.OneTimeAccessCode{
				ObjectMeta: metav1.ObjectMeta{
					GenerateName: "otacset-",
					Labels: map[string]string{
						labels.OneTimeAccessCodeSetLabel:      set.GetName(),
						labels.OneTimeAccessCodeRedeemedLabel: "false",
					},
				},
				Spec: set.Spec.Template,
			}

			if err := controllerutil.SetControllerReference(set, newOtac, cx.kclient.Scheme()); err != nil {
				return reconcile.Result{}, err
			}

			err := cx.kclient.Create(ctx, newOtac)
			if err != nil {
				eventbuilder.Error().For(set).By(OtacSetScaleControllerName, "").
					Reason("error creating otac").Note(err.Error()).WriteOrLog(cx.kclient)
			}
		}

		reQ = true
	}

	// ... too many
	if set.Status.Created > set.Spec.Count {
		eventbuilder.Info().For(set).By(OtacSetScaleControllerName, "").
			Reason("attempting to scale down otacset").WriteOrLog(cx.kclient)

		// list the number of objects that meet the requirement
		list := &v4alpha1.OneTimeAccessCodeList{}
		if err := cx.kclient.List(ctx, list, &client.ListOptions{
			Limit: int64(set.Status.Created - set.Spec.Count),
			LabelSelector: labels2.SelectorFromSet(map[string]string{
				labels.OneTimeAccessCodeRedeemedLabel: "false",
			}),
		}); err != nil {
			return reconcile.Result{}, err
		}

		// attempt to delete them
		for _, i := range list.Items {
			if err := cx.kclient.Delete(ctx, &i); err != nil {
				eventbuilder.Error().For(set).By(OtacSetScaleControllerName, "").
					Reason("error deleting otac").For(set).Note(err.Error()).WriteOrLog(cx.kclient)
			}
		}

		reQ = true
	}

	// ... just right, do nothing
	return reconcile.Result{Requeue: reQ}, nil
}
