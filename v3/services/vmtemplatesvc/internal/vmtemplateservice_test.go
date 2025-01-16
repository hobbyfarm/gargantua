package vmtemplateservice

import (
	"github.com/hobbyfarm/gargantua/v3/pkg/util"
	"reflect"
	"testing"
)

func Test_normalizeCost(t *testing.T) {
	type input struct {
		basePrice string
		timeUnit  string
	}
	tests := []struct {
		name          string
		input         input
		wantBasePrice *string
		wantTimeUnit  *string
		wantErr       bool
	}{
		{
			name: "ok non set",
			input: input{
				basePrice: "",
				timeUnit:  "",
			},
			wantBasePrice: nil,
			wantTimeUnit:  nil,
			wantErr:       false,
		},
		{
			name: "ok both set",
			input: input{
				basePrice: "10",
				timeUnit:  "sec",
			},
			wantBasePrice: util.RefOrNil("10"),
			wantTimeUnit:  util.RefOrNil(util.TimeUnitSeconds),
			wantErr:       false,
		},
		{
			name: "nok only basePrice",
			input: input{
				basePrice: "10",
				timeUnit:  "",
			},
			wantErr: true,
		},
		{
			name: "nok only timeUnit",
			input: input{
				basePrice: "",
				timeUnit:  "sec",
			},
			wantErr: true,
		},
		{
			name: "nok invalid basePrice",
			input: input{
				basePrice: "boom",
				timeUnit:  util.TimeUnitSeconds,
			},
			wantErr: true,
		},
		{
			name: "nok invalid timeUnit",
			input: input{
				basePrice: "10",
				timeUnit:  "boom",
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotBasePrice, gotTimeUnit, err := normalizeCost(tt.input.basePrice, tt.input.timeUnit)
			if (err != nil) != tt.wantErr {
				t.Errorf("normalizeCost() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(gotBasePrice, tt.wantBasePrice) {
				t.Errorf("normalizeCost() gotBasePrice = %v, want %v", gotBasePrice, tt.wantBasePrice)
			}
			if !reflect.DeepEqual(gotTimeUnit, tt.wantTimeUnit) {
				t.Errorf("normalizeCost() gotTimeUnit = %v, want %v", gotTimeUnit, tt.wantTimeUnit)
			}
		})
	}
}
