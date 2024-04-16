package scheduledevent

import (
	"context"
	"fmt"
	v1 "github.com/hobbyfarm/gargantua/v3/pkg/apis/hobbyfarm.io/v1"
	util2 "github.com/hobbyfarm/gargantua/v3/pkg/util"
	"github.com/hobbyfarm/mink/pkg/strategy"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/validation/field"
	"k8s.io/apiserver/pkg/registry/rest"
)

type v1Validator struct {
}

func NewV1Storage(scheduledeventStorage strategy.CompleteStrategy) (rest.Storage, error) {

}

func (v1v v1Validator) Validate(ctx context.Context, obj runtime.Object) (result field.ErrorList) {
	se := obj.(*v1.ScheduledEvent)

	if len(se.Spec.Scenarios) == 0 && len(se.Spec.Courses) == 0 {
		result = append(result,
			field.TypeInvalid(field.NewPath("spec"), se.Spec, "must specify at least one scenario or course"))
	}

	random := util2.RandStringRunes(16)
	se.Name = "se-" + util2.GenerateResourceName("se", random, 10)
	se.Spec.Creator = fmt.Sprintf("%v", ctx.Value("user"))
}
