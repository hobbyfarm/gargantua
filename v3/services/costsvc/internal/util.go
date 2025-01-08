package costservice

import (
	"fmt"
	v1 "github.com/hobbyfarm/gargantua/v3/pkg/apis/hobbyfarm.io/v1"
	costpb "github.com/hobbyfarm/gargantua/v3/protos/cost"
	"math"
	"strings"
	"time"
)

type TimeUnit = string

const (
	TimeUnitSeconds TimeUnit = "seconds"
	TimeUnitMinutes TimeUnit = "minutes"
	TimeUnitHours   TimeUnit = "hours"
)

func ParseTimeUnit(s string) (TimeUnit, error) {
	lower := strings.ToLower(s)
	switch lower {
	case "seconds", "second", "sec", "s":
		return TimeUnitSeconds, nil
	case "minutes", "minute", "min", "m":
		return TimeUnitMinutes, nil
	case "hours", "hour", "h":
		return TimeUnitHours, nil
	default:
		return TimeUnitSeconds, fmt.Errorf("%s is not a valid time unit", s)
	}
}

func CostResourceCalcCost(cr v1.CostResource, duration time.Duration) uint64 {
	var durationInTimeUnit uint64

	switch cr.TimeUnit {
	case TimeUnitSeconds:
		durationInTimeUnit = uint64(math.Ceil(duration.Seconds()))
	case TimeUnitMinutes:
		durationInTimeUnit = uint64(math.Ceil(duration.Minutes()))
	case TimeUnitHours:
		durationInTimeUnit = uint64(math.Ceil(duration.Hours()))
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
	var totalCost uint64

	for kind, resources := range GroupCostResourceByKind(cb.cost.Spec.Resources) {
		var costForKind uint64
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
