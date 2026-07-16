package indeed

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestJobTypesFromAttributes(t *testing.T) {
	attrs := []struct{ Key, Label string }{
		{"EY33Q", "Health insurance"},
		{"CF3CP", "Full-time"},
		{"GX2AS", "Cash register operations"},
		{"5QWDV", "Permanent"},
		{"75GKK", "Part-time"},
		{"NJXCK", "Contract"},
		{"VDTG7", "Internship"},
		{"4HKF7", "Temporary"},
		{"Y4JG9", "Entry level"},
	}
	assert.Equal(t, []string{
		"Full-time", "Permanent", "Part-time", "Contract", "Internship", "Temporary",
	}, jobTypesFromAttributes(attrs))
}

func TestJobTypesFromAttributesEmpty(t *testing.T) {
	assert.Nil(t, jobTypesFromAttributes(nil))
	assert.Nil(t, jobTypesFromAttributes([]struct{ Key, Label string }{
		{"EY33Q", "Health insurance"},
	}))
}
