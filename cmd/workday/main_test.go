package main

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestRunFacetsMissingTenant(t *testing.T) {
	err := runFacets(context.Background(), "", time.Second, "", nil, "text")
	assert.ErrorContains(t, err, "--tenant is required")
}

func TestRunFacetsUnknownTenant(t *testing.T) {
	err := runFacets(context.Background(), "doesnotexist-tenant-xyz", time.Second, "", nil, "text")
	assert.ErrorContains(t, err, `tenant "doesnotexist-tenant-xyz" not found`)
	assert.ErrorContains(t, err, "workday companies")
}

func TestRunSearchMissingTenant(t *testing.T) {
	err := runSearch(context.Background(), "", time.Second, "", 20, 0, nil, "text")
	assert.ErrorContains(t, err, "--tenant is required")
}

func TestRunSearchUnknownTenant(t *testing.T) {
	err := runSearch(context.Background(), "doesnotexist-tenant-xyz", time.Second, "", 20, 0, nil, "text")
	assert.ErrorContains(t, err, `tenant "doesnotexist-tenant-xyz" not found`)
	assert.ErrorContains(t, err, "workday companies")
}
