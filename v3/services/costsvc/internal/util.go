package costservice

import (
	v1 "github.com/hobbyfarm/gargantua/v3/pkg/apis/hobbyfarm.io/v1"
	"github.com/hobbyfarm/gargantua/v3/pkg/util"
	costpb "github.com/hobbyfarm/gargantua/v3/protos/cost"
	"math"
	"time"
)

func CostResourceCalcCost(cr v1.CostResource, duration time.Duration) float64 {
	var durationInTimeUnit float64

	switch cr.TimeUnit {
	case util.TimeUnitSeconds:
		durationInTimeUnit = math.Ceil(duration.Seconds())
	case util.TimeUnitMinutes:
		durationInTimeUnit = math.Ceil(duration.Minutes())
	case util.TimeUnitHours:
		durationInTimeUnit = math.Ceil(duration.Hours())
	default:
		durationInTimeUnit = 0
	}

	return durationInTimeUnit * cr.BasePrice
}

func CostResourceDuration(cr v1.CostResource, defaultDeletion time.Time) time.Duration {
	creation := time.Unix(cr.CreationUnixTimestamp, 0)

	deletion := defaultDeletion
	if cr.DeletionUnixTimestamp != 0 {
		deletion = time.Unix(cr.DeletionUnixTimestamp, 0)
	}

	return deletion.Sub(creation)
}

func GroupCostResourceByKind(resources []v1.CostResource) map[string][]v1.CostResource {
	grouped := make(map[string][]v1.CostResource)

	for _, resource := range resources {
		grouped[resource.Kind] = append(grouped[resource.Kind], resource)
	}
	return grouped
}

type CostBuilder struct {
	cost   *v1.Cost
	filter func(cr v1.CostResource) bool
}

func NewCostBuilder(cost *v1.Cost) *CostBuilder {
	cb := &CostBuilder{cost: cost}
	return cb.WithAllCosts()
}

func (cb *CostBuilder) WithAllCosts() *CostBuilder {
	cb.filter = func(_ v1.CostResource) bool { return false }
	return cb
}

func (cb *CostBuilder) WithPresentCosts() *CostBuilder {
	cb.filter = func(cr v1.CostResource) bool {
		// historic costs have a deletion timestamp
		return cr.DeletionUnixTimestamp != 0
	}
	return cb
}

func (cb *CostBuilder) WithHistoricCosts() *CostBuilder {
	cb.filter = func(cr v1.CostResource) bool {
		// present costs have NO deletion timestamp
		return cr.DeletionUnixTimestamp == 0
	}
	return cb
}

func (cb *CostBuilder) Build(now time.Time) *costpb.Cost {
	var costSources []*costpb.CostSource
	var totalCost float64

	for kind, resources := range GroupCostResourceByKind(cb.cost.Spec.Resources) {
		var costForKind float64
		var count uint64

		for _, resource := range resources {
			if cb.filter(resource) {
				continue
			}
			duration := CostResourceDuration(resource, now)
			costForKind += CostResourceCalcCost(resource, duration)
			count += 1
		}

		totalCost += costForKind
		costSources = append(costSources, &costpb.CostSource{
			Kind:  kind,
			Cost:  costForKind,
			Count: count,
		})
	}

	return &costpb.Cost{
		CostGroup: cb.cost.Name,
		Total:     totalCost,
		Source:    costSources,
	}
}
