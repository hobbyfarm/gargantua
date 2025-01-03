package v1

import (
	"testing"
	"time"
)

func TestCostResource_Duration(t *testing.T) {
	defaultDeletion := time.Unix(10, 0)

	tests := []struct {
		name  string
		input CostResource
		want  time.Duration
	}{
		{
			name: "deletion",
			input: CostResource{
				CreationUnixTimestamp: 0,
				DeletionUnixTimestamp: 10,
			},
			want: 10 * time.Second,
		},
		{
			name: "no deletion",
			input: CostResource{
				CreationUnixTimestamp: 0,
			},
			want: 10 * time.Second,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.input.Duration(defaultDeletion); got != tt.want {
				t.Errorf("Duration() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCostResource_CalcCost(t *testing.T) {
	tests := []struct {
		name     string
		input    CostResource
		duration time.Duration
		want     uint64
	}{
		{
			name: "seconds",
			input: CostResource{
				BasePrice: 1,
				TimeUnit:  TimeUnitSeconds,
			},
			duration: 10 * time.Second,
			want:     10,
		},
		{
			name: "minutes",
			input: CostResource{
				BasePrice: 2,
				TimeUnit:  TimeUnitMinutes,
			},
			duration: 60 * time.Second,
			want:     2,
		},
		{
			name: "minutes and always round up",
			input: CostResource{
				BasePrice: 2,
				TimeUnit:  TimeUnitMinutes,
			},
			duration: 61 * time.Second,
			want:     4,
		},
		{
			name: "hours",
			input: CostResource{
				BasePrice: 2,
				TimeUnit:  TimeUnitHours,
			},
			duration: 1 * time.Hour,
			want:     2,
		},
		{
			name: "hours and always round up",
			input: CostResource{
				BasePrice: 2,
				TimeUnit:  TimeUnitHours,
			},
			duration: 61 * time.Minute,
			want:     4,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.input.CalcCost(tt.duration); got != tt.want {
				t.Errorf("CalcCost() = %v, want %v", got, tt.want)
			}
		})
	}
}
