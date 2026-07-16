package main

import (
	"testing"

	"github.com/amikai/openings-mcp/internal/provider/indeed"
	"github.com/stretchr/testify/assert"
)

func TestFormatCompensation(t *testing.T) {
	tests := []struct {
		name string
		comp *indeed.Compensation
		want string
	}{
		{
			name: "range",
			comp: &indeed.Compensation{MinAmount: 22.5, MaxAmount: 27.5, Currency: "USD", Interval: "HOUR"},
			want: "22.5-27.5 USD (HOUR)",
		},
		{
			name: "at least",
			comp: &indeed.Compensation{MinAmount: 20, Currency: "USD", Interval: "HOUR"},
			want: ">= 20 USD (HOUR)",
		},
		{
			name: "at most",
			comp: &indeed.Compensation{MaxAmount: 30, Currency: "USD", Interval: "HOUR"},
			want: "<= 30 USD (HOUR)",
		},
		{
			name: "exactly",
			comp: &indeed.Compensation{MinAmount: 17.5, MaxAmount: 17.5, Currency: "USD", Interval: "HOUR"},
			want: "17.5 USD (HOUR)",
		},
		{
			name: "undisclosed amounts",
			comp: &indeed.Compensation{Currency: "USD", Interval: "YEAR"},
			want: "undisclosed USD (YEAR)",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, formatCompensation(tt.comp))
		})
	}
}
