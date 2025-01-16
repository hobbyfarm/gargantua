package util

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestParseTimeUnit(t *testing.T) {
	tests := []struct {
		name    string
		input   []string
		want    TimeUnit
		wantErr bool
	}{
		{
			name:    "ok second",
			input:   []string{"seconds", "second", "sec", "s", "SECONDS", "SECOND", "SEC", "S"},
			want:    TimeUnitSeconds,
			wantErr: false,
		},
		{
			name:    "ok minute",
			input:   []string{"minutes", "minute", "min", "m", "MINUTES", "MINUTE", "MIN", "M"},
			want:    TimeUnitMinutes,
			wantErr: false,
		},
		{
			name:    "ok hour",
			input:   []string{"hours", "hour", "h", "HOURS", "HOUR", "H"},
			want:    TimeUnitHours,
			wantErr: false,
		},
		{
			name:    "nok",
			input:   []string{"", " ", "idk"},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		for _, input := range tt.input {
			t.Run(fmt.Sprintf("%s %s", tt.name, input), func(t *testing.T) {
				actual, err := ParseTimeUnit(input)
				if tt.wantErr {
					require.Errorf(t, err, "error expected ParseTimeUnit(%v)", input)
				} else {
					require.NoErrorf(t, err, "no error expected ParseTimeUnit(%v)", input)
					assert.Equalf(t, tt.want, actual, "ParseTimeUnit(%v)", input)
				}
			})
		}
	}
}
