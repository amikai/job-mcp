package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/amikai/openings-mcp/internal/provider/oracle"
)

func TestParseFilters(t *testing.T) {
	tests := []struct {
		name    string
		values  []string
		want    map[oracle.Facet][]string
		wantErr string
	}{
		{
			name:   "repeat same facet",
			values: []string{"location=1", "location=2", "workplace-type=ORA_REMOTE"},
			want: map[oracle.Facet][]string{
				oracle.FacetLocation:      {"1", "2"},
				oracle.FacetWorkplaceType: {"ORA_REMOTE"},
			},
		},
		{name: "trim whitespace", values: []string{" category = 42 "}, want: map[oracle.Facet][]string{oracle.FacetCategory: {"42"}}},
		{name: "missing equals", values: []string{"location"}, wantErr: "must be name=id"},
		{name: "empty value", values: []string{"location="}, wantErr: "must be name=id"},
		{name: "unknown facet", values: []string{"salary=high"}, wantErr: "unknown facet"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseFilters(tt.values)
			if tt.wantErr != "" {
				require.ErrorContains(t, err, tt.wantErr)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestRunDiscoverEndToEnd(t *testing.T) {
	server := oracle.NewMockServer()
	defer server.Close()
	var out bytes.Buffer

	err := runDiscover(t.Context(), commonFlags{
		careersURL: careersURL(server.URL),
		timeout:    time.Second,
		format:     "json",
	}, commandEnv{httpClient: server.Client(), out: &out})
	require.NoError(t, err)

	var site oracle.Site
	require.NoError(t, json.Unmarshal(out.Bytes(), &site))
	assert.Equal(t, "Mayo-US", site.Site)
	assert.Equal(t, "CX_1", site.SiteNumber)
	assert.Equal(t, server.URL, site.APIBaseURL)
}

func TestRunSearchEndToEnd(t *testing.T) {
	server := oracle.NewMockServer()
	defer server.Close()
	var out bytes.Buffer

	err := runSearch(t.Context(), searchFlags{
		commonFlags: commonFlags{
			careersURL: careersURL(server.URL),
			timeout:    time.Second,
			format:     "json",
		},
		limit: 3,
	}, commandEnv{httpClient: server.Client(), out: &out})
	require.NoError(t, err)

	var result oracle.SearchResult
	require.NoError(t, json.Unmarshal(out.Bytes(), &result))
	assert.Equal(t, 1330, result.Total)
	require.Len(t, result.Jobs, 3)
	assert.Equal(t, "361564", result.Jobs[0].ID)
	assert.Equal(t, "Cardiologist - Echo / Imaging", result.Jobs[0].Title)
}

func TestRunFacetsEndToEnd(t *testing.T) {
	server := oracle.NewMockServer()
	defer server.Close()
	var out bytes.Buffer

	err := runFacets(t.Context(), facetsFlags{
		commonFlags: commonFlags{
			careersURL: careersURL(server.URL),
			timeout:    time.Second,
			format:     "json",
		},
	}, commandEnv{httpClient: server.Client(), out: &out})
	require.NoError(t, err)

	var facets map[oracle.Facet][]oracle.FacetOption
	require.NoError(t, json.Unmarshal(out.Bytes(), &facets))
	assert.Equal(t, "7", facets[oracle.FacetPostingDate][0].ID)
	assert.Equal(t, "300000034547579", facets[oracle.FacetWorkLocation][0].ID)
	assert.Equal(t, "1", facets[oracle.FacetOrganization][0].ID)
}

func TestRunDetailEndToEnd(t *testing.T) {
	server := oracle.NewMockServer()
	defer server.Close()
	var out bytes.Buffer

	err := runDetail(t.Context(), detailFlags{
		commonFlags: commonFlags{
			careersURL: careersURL(server.URL),
			timeout:    time.Second,
			format:     "text",
		},
		jobID: "386920",
	}, commandEnv{httpClient: server.Client(), out: &out})
	require.NoError(t, err)
	assert.Contains(t, out.String(), "Senior Analyst - ATS")
	assert.Contains(t, out.String(), "Access Technologies and Systems")
	assert.Contains(t, out.String(), "/job/386920")
}

func TestRunDetailNotFound(t *testing.T) {
	server := oracle.NewMockServer()
	defer server.Close()
	var out bytes.Buffer

	err := runDetail(t.Context(), detailFlags{
		commonFlags: commonFlags{
			careersURL: careersURL(server.URL),
			timeout:    time.Second,
			format:     "json",
		},
		jobID: "999999999999",
	}, commandEnv{httpClient: server.Client(), out: &out})
	require.Error(t, err)
	assert.True(t, errors.Is(err, oracle.ErrJobNotFound))
}

func TestCommandValidation(t *testing.T) {
	env := commandEnv{out: &bytes.Buffer{}}

	err := runSearch(t.Context(), searchFlags{
		commonFlags: commonFlags{timeout: time.Second},
		limit:       20,
	}, env)
	assert.ErrorContains(t, err, "--url is required")

	err = runDetail(t.Context(), detailFlags{
		commonFlags: commonFlags{careersURL: "https://example.com", timeout: time.Second},
	}, env)
	assert.ErrorContains(t, err, "--id is required")

	err = runDiscover(t.Context(), commonFlags{
		careersURL: "https://example.com",
		timeout:    0,
	}, env)
	assert.ErrorContains(t, err, "--timeout must be greater than zero")
}

func careersURL(baseURL string) string {
	return baseURL + "/hcmUI/CandidateExperience/en/sites/Mayo-US/jobs"
}
