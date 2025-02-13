package otac

import (
	"context"
	"fmt"
	"github.com/hobbyfarm/gargantua/v4/pkg/apis/hobbyfarm.io/v4alpha1"
	"github.com/hobbyfarm/gargantua/v4/pkg/eventbuilder"
	"github.com/hobbyfarm/gargantua/v4/pkg/labels"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

const (
	OtacSetScaleControllerName = "otacset-scale-controller"
)

type otacSetScaleController struct {
	kclient client.Client
}

func (cx otacSetScaleController) Reconcile(ctx context.Context, request reconcile.Request) (*reconcile.Result, error) {
	set := &v4alpha1.OneTimeAccessCodeSet{}
	if err := cx.kclient.Get(ctx, request.NamespacedName, set); err != nil {
		return nil, client.IgnoreNotFound(err)
	}

	otacList := &v4alpha1.OneTimeAccessCodeList{}
	if err := cx.kclient.List(ctx, otacList); err != nil {
		return nil, client.IgnoreNotFound(err)
	}

	// how many we got?
	set.Status.Created = len(otacList.Items)
	if err := cx.kclient.Status().Update(ctx, set); err != nil {
		return nil, err
	}

	// ... not enough
	if set.Status.Created < set.Spec.Count {
		// issue an event
		eventbuilder.
			Info().
			For(set).
			By(OtacSetScaleControllerName, "").
			Reason(fmt.Sprintf("creating %d OTACs", set.Spec.Count-set.Status.Created)).
			WriteOrLog()

		// create some new ones
		for range len(otacList.Items) - set.Status.Created {
			newOtac := &v4alpha1.OneTimeAccessCode{
				ObjectMeta: metav1.ObjectMeta{
					GenerateName: "otacset-",
					Labels: map[string]string{
						labels.OneTimeAccessCodeSetLabel: set.GetName(),
					},
				},
				Spec: set.Spec.Template,
			}
		}
	}
}
