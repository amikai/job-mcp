package ats

import (
	"net/http"
	"net/url"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/amikai/openings-mcp/internal/provider/icims"
)

func testICIMSAdapter(t *testing.T) *ICIMSAdapter {
	t.Helper()
	mock := icims.NewMockServer()
	t.Cleanup(mock.Close)
	a := NewICIMSAdapter(&http.Client{Timeout: 5 * time.Second})
	a.baseURL = func(string) string { return mock.URL }
	return a
}

func TestICIMSRosterBuildsRegistry(t *testing.T) {
	_, err := NewRegistry(NewICIMSAdapter(http.DefaultClient))
	require.NoError(t, err)
}

func TestICIMSRosterReturnsCompanyNames(t *testing.T) {
	a := NewICIMSAdapter(http.DefaultClient)
	roster := a.Roster()
	require.NotEmpty(t, roster)
	found := false
	for _, c := range roster {
		if c.Slug == "careers-peraton.icims.com" {
			found = true
			assert.Equal(t, "Peraton", c.Name)
		}
	}
	assert.True(t, found, "expected careers-peraton.icims.com in roster")
}

func TestICIMSParseCareersURL(t *testing.T) {
	a := NewICIMSAdapter(http.DefaultClient)
	cases := []struct {
		raw  string
		ok   bool
		slug string
	}{
		{"https://careers-peraton.icims.com/jobs/search?ss=1", true, "careers-peraton.icims.com"},
		{"https://uscareers-example.icims.com/jobs/1/x/job", true, "uscareers-example.icims.com"},
		{"https://login.icims.com/", false, ""},
		{"https://boards.greenhouse.io/x", false, ""},
	}
	for _, tc := range cases {
		u, err := url.Parse(tc.raw)
		require.NoError(t, err)
		slug, ok := a.ParseCareersURL(u)
		assert.Equal(t, tc.ok, ok, tc.raw)
		assert.Equal(t, tc.slug, slug, tc.raw)
	}
}

// mockFixtureHost is a roster host used only for mock-backed tests. The
// adapter overrides baseURL to the mock server, so live DNS is never hit.
const mockFixtureHost = "careers-peraton.icims.com"

func TestICIMSSearch(t *testing.T) {
	a := testICIMSAdapter(t)
	res, err := a.Search(t.Context(), mockFixtureHost, SearchParams{})
	require.NoError(t, err)
	assert.Equal(t, 3, res.TotalCount)
	assert.Equal(t, 1, res.Page)
	assert.Len(t, res.Jobs, 3)

	first := res.Jobs[0]
	assert.Equal(t, "1977", first.JobID)
	assert.Equal(t, "Senior Product Manager", first.Title)
	assert.Contains(t, first.Location, "Austin")
	assert.Equal(t, "https://careers-peraton.icims.com/jobs/1977/job/job", first.URL)
}

func TestICIMSSearchKeyword(t *testing.T) {
	a := testICIMSAdapter(t)
	res, err := a.Search(t.Context(), mockFixtureHost, SearchParams{Query: "Product"})
	require.NoError(t, err)
	assert.NotEmpty(t, res.Jobs)
}

func TestICIMSSearchNoResults(t *testing.T) {
	a := testICIMSAdapter(t)
	res, err := a.Search(t.Context(), mockFixtureHost, SearchParams{Query: "zzzznonexistentkeyword12345"})
	require.NoError(t, err)
	assert.Empty(t, res.Jobs)
	assert.Equal(t, 0, res.TotalCount)
}

func TestICIMSFiltersEmpty(t *testing.T) {
	a := testICIMSAdapter(t)
	fs, err := a.Filters(t.Context(), mockFixtureHost)
	require.NoError(t, err)
	assert.Empty(t, fs)
}

func TestICIMSDetail(t *testing.T) {
	a := testICIMSAdapter(t)
	d, err := a.Detail(t.Context(), mockFixtureHost, "1977")
	require.NoError(t, err)
	assert.Equal(t, "1977", d.JobID)
	assert.Equal(t, "Senior Product Manager", d.Title)
	assert.Equal(t, "Peraton", d.Company)
	assert.Contains(t, d.Location, "Austin")
	assert.NotEmpty(t, d.Description)
	assert.Equal(t, "https://careers-peraton.icims.com/jobs/1977/job/job", d.URL)
}

func TestICIMSDetailNotFound(t *testing.T) {
	a := testICIMSAdapter(t)
	_, err := a.Detail(t.Context(), mockFixtureHost, "999999999")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestICIMSDetailBadID(t *testing.T) {
	a := testICIMSAdapter(t)
	_, err := a.Detail(t.Context(), mockFixtureHost, "not-a-number")
	require.Error(t, err)
}

func TestICIMSUnknownSlug(t *testing.T) {
	a := testICIMSAdapter(t)
	_, err := a.Search(t.Context(), "not-a-company", SearchParams{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unknown company")
}
