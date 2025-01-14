package costservice

import (
	"github.com/hobbyfarm/gargantua/v3/pkg/apis/hobbyfarm.io/v1"
	"github.com/hobbyfarm/gargantua/v3/pkg/util"
	"reflect"
	"sort"
	"testing"
	"time"
)

func TestCostResourceCalcCost(t *testing.T) {
	tests := []struct {
		name     string
		input    v1.CostResource
		duration time.Duration
		want     float64
	}{
		{
			name: "seconds",
			input: v1.CostResource{
				BasePrice: 1,
				TimeUnit:  util.TimeUnitSeconds,
			},
			duration: 10 * time.Second,
			want:     10,
		},
		{
			name: "minutes",
			input: v1.CostResource{
				BasePrice: 2,
				TimeUnit:  util.TimeUnitMinutes,
			},
			duration: 60 * time.Second,
			want:     2,
		},
		{
			name: "minutes and always round up",
			input: v1.CostResource{
				BasePrice: 2,
				TimeUnit:  util.TimeUnitMinutes,
			},
			duration: 61 * time.Second,
			want:     4,
		},
		{
			name: "hours",
			input: v1.CostResource{
				BasePrice: 2,
				TimeUnit:  util.TimeUnitHours,
			},
			duration: 1 * time.Hour,
			want:     2,
		},
		{
			name: "hours and always round up",
			input: v1.CostResource{
				BasePrice: 2,
				TimeUnit:  util.TimeUnitHours,
			},
			duration: 61 * time.Minute,
			want:     4,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := CostResourceCalcCost(tt.input, tt.duration); got != tt.want {
				t.Errorf("CalcCost() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCostResourceDuration(t *testing.T) {
	defaultDeletion := time.Unix(10, 0)

	tests := []struct {
		name  string
		input v1.CostResource
		want  time.Duration
	}{
		{
			name: "deletion",
			input: v1.CostResource{
				CreationUnixTimestamp: 0,
				DeletionUnixTimestamp: 10,
			},
			want: 10 * time.Second,
		},
		{
			name: "no deletion",
			input: v1.CostResource{
				CreationUnixTimestamp: 0,
			},
			want: 10 * time.Second,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := CostResourceDuration(tt.input, defaultDeletion); got != tt.want {
				t.Errorf("Duration() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGroupCostResourceByKind(t *testing.T) {
	input := []v1.CostResource{
		{Id: "a", Kind: "Pod"},
		{Id: "x", Kind: "Deployment"},
		{Id: "b", Kind: "Pod"},
		{Id: "y", Kind: "Deployment"},
		{Id: "c", Kind: "Pod"},
		{Id: "1", Kind: "VirtualMachine"},
	}
	want := map[string][]v1.CostResource{
		"Pod": {
			{Id: "a", Kind: "Pod"},
			{Id: "b", Kind: "Pod"},
			{Id: "c", Kind: "Pod"},
		},
		"Deployment": {
			{Id: "x", Kind: "Deployment"},
			{Id: "y", Kind: "Deployment"},
		},
		"VirtualMachine": {
			{Id: "1", Kind: "VirtualMachine"},
		},
	}

	got := GroupCostResourceByKind(input)

	// Sort the slices for deterministic comparison
	for k := range got {
		sort.Slice(got[k], func(i, j int) bool {
			return got[k][i].Id < got[k][j].Id
		})
	}
	for k := range want {
		sort.Slice(want[k], func(i, j int) bool {
			return want[k][i].Id < want[k][j].Id
		})
	}

	// Perform the comparison
	if !reflect.DeepEqual(got, want) {
		t.Errorf("groupByKind() = %v, want %v", got, want)
	}
}
