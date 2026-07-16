package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseProvidersIncludesTeamtailor(t *testing.T) {
	got, err := parseProviders("workday,teamtailor")
	require.NoError(t, err)
	assert.Equal(t, []string{"teamtailor", "workday"}, got)
}

func TestBuildTeamtailorAdapter(t *testing.T) {
	adapters, err := buildAdapters([]string{"teamtailor"})
	require.NoError(t, err)
	require.Len(t, adapters, 1)
	assert.Equal(t, "teamtailor", adapters[0].Name())
}

func TestParseProvidersIncludesOracle(t *testing.T) {
	got, err := parseProviders("workday,oracle")
	require.NoError(t, err)
	assert.Equal(t, []string{"oracle", "workday"}, got)
}

func TestBuildOracleAdapter(t *testing.T) {
	adapters, err := buildAdapters([]string{"oracle"})
	require.NoError(t, err)
	require.Len(t, adapters, 1)
	assert.Equal(t, "oracle", adapters[0].Name())
}
