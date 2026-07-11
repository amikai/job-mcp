package nvidia

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSplitExternalPath(t *testing.T) {
	tests := []struct {
		name         string
		path         string
		wantLocation string
		wantTitle    string
		wantOK       bool
	}{
		{
			name:         "valid path",
			path:         "/job/US-CA-Remote/Software-Engineer--CUDA-Q_JR2011649",
			wantLocation: "US-CA-Remote",
			wantTitle:    "Software-Engineer--CUDA-Q_JR2011649",
			wantOK:       true,
		},
		{name: "missing /job/ prefix", path: "US/T", wantOK: false},
		{name: "empty location", path: "/job//T", wantOK: false},
		{name: "empty title", path: "/job/US/", wantOK: false},
		{name: "extra segment", path: "/job/L/T/extra", wantOK: false},
		{name: "empty input", path: "", wantOK: false},
		{name: "prefix only", path: "/job/", wantOK: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			location, title, ok := SplitExternalPath(tt.path)
			assert.Equal(t, tt.wantOK, ok)
			if tt.wantOK {
				assert.Equal(t, tt.wantLocation, location)
				assert.Equal(t, tt.wantTitle, title)
			}
		})
	}
}
