package costservice

import (
	"github.com/hobbyfarm/gargantua/v3/pkg/labels"
	"github.com/hobbyfarm/gargantua/v3/pkg/util"
	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"testing"
	"time"
)

func Test_newCostGroup(t *testing.T) {
	creationUnixTimestamp := int64(100)
	creation := time.Unix(creationUnixTimestamp, 0)

	tests := []struct {
		name    string
		input   *unstructured.Unstructured
		want    *costGroup
		wantErr assert.ErrorAssertionFunc
	}{
		{
			name: "ok",
			input: &unstructured.Unstructured{
				Object: map[string]interface{}{
					"kind": "VirtualMachine",
					"metadata": map[string]interface{}{
						"name":              "vm-test",
						"creationTimestamp": creation.Format(time.RFC3339),
						"labels": map[string]interface{}{
							labels.CostGroup:     "my-cost-group",
							labels.CostBasePrice: "10.01",
							labels.CostTimeUnit:  util.TimeUnitSeconds,
						},
					},
				},
			},
			want: &costGroup{
				Id:                "vm-test",
				Kind:              "VirtualMachine",
				CostGroup:         "my-cost-group",
				BasePrice:         10.01,
				TimeUnit:          util.TimeUnitSeconds,
				CreationTimestamp: creationUnixTimestamp,
			},
		},
		{
			name: "no cost group",
			input: &unstructured.Unstructured{
				Object: map[string]interface{}{
					"kind": "VirtualMachine",
					"metadata": map[string]interface{}{
						"name":              "vm-test",
						"creationTimestamp": creation.Format(time.RFC3339),
						"labels": map[string]interface{}{
							labels.CostBasePrice: "10.01",
							labels.CostTimeUnit:  util.TimeUnitSeconds,
						},
					},
				},
			},
			wantErr: func(t assert.TestingT, err error, i ...interface{}) bool {
				return assert.ErrorContains(t, err, labels.CostGroup)
			},
		},
		{
			name: "no base price",
			input: &unstructured.Unstructured{
				Object: map[string]interface{}{
					"kind": "VirtualMachine",
					"metadata": map[string]interface{}{
						"name":              "vm-test",
						"creationTimestamp": creation.Format(time.RFC3339),
						"labels": map[string]interface{}{
							labels.CostGroup:    "my-cost-group",
							labels.CostTimeUnit: util.TimeUnitSeconds,
						},
					},
				},
			},
			wantErr: func(t assert.TestingT, err error, i ...interface{}) bool {
				return assert.ErrorContains(t, err, labels.CostBasePrice)
			},
		},
		{
			name: "invalid base price",
			input: &unstructured.Unstructured{
				Object: map[string]interface{}{
					"kind": "VirtualMachine",
					"metadata": map[string]interface{}{
						"name":              "vm-test",
						"creationTimestamp": creation.Format(time.RFC3339),
						"labels": map[string]interface{}{
							labels.CostGroup:     "my-cost-group",
							labels.CostBasePrice: "invalid",
							labels.CostTimeUnit:  util.TimeUnitSeconds,
						},
					},
				},
			},
			wantErr: func(t assert.TestingT, err error, i ...interface{}) bool {
				return assert.ErrorContains(t, err, labels.CostBasePrice)
			},
		},

		{
			name: "no time unit",
			input: &unstructured.Unstructured{
				Object: map[string]interface{}{
					"kind": "VirtualMachine",
					"metadata": map[string]interface{}{
						"name":              "vm-test",
						"creationTimestamp": creation.Format(time.RFC3339),
						"labels": map[string]interface{}{
							labels.CostGroup:     "my-cost-group",
							labels.CostBasePrice: "10.01",
						},
					},
				},
			},
			wantErr: func(t assert.TestingT, err error, i ...interface{}) bool {
				return assert.ErrorContains(t, err, labels.CostTimeUnit)
			},
		},
		{
			name: "invalid base price",
			input: &unstructured.Unstructured{
				Object: map[string]interface{}{
					"kind": "VirtualMachine",
					"metadata": map[string]interface{}{
						"name":              "vm-test",
						"creationTimestamp": creation.Format(time.RFC3339),
						"labels": map[string]interface{}{
							labels.CostGroup:     "my-cost-group",
							labels.CostBasePrice: "10.01",
							labels.CostTimeUnit:  "invalid",
						},
					},
				},
			},
			wantErr: func(t assert.TestingT, err error, i ...interface{}) bool {
				return assert.ErrorContains(t, err, labels.CostTimeUnit)
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := newCostGroup(tt.input)
			if tt.wantErr != nil {
				tt.wantErr(t, err)
			} else {
				assert.NoErrorf(t, err, "newCostGroup() error = %v", err)
				assert.Equal(t, tt.want, got)
			}
		})
	}
}
