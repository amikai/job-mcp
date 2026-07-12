package ats

import (
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/amikai/openings-mcp/internal/provider/smartrecruiters"
)

// recordingQueryProxy forwards every request to inner and records each
// request's path+query, so tests can assert what the adapter sent
// upstream (the workday tests' recordingProxy records POST bodies; the
// SmartRecruiters API is GET-only, so the URL is the whole request).
func recordingQueryProxy(t *testing.T, inner string) (*httptest.Server, *[]string) {
	t.Helper()
	var urls []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		urls = append(urls, r.URL.String())
		req, err := http.NewRequestWithContext(r.Context(), r.Method, inner+r.URL.String(), nil)
		if !assert.NoError(t, err, "proxy") {
			return
		}
		rsp, err := http.DefaultClient.Do(req)
		if !assert.NoError(t, err, "proxy") {
			return
		}
		defer rsp.Body.Close()
		w.Header().Set("Content-Type", rsp.Header.Get("Content-Type"))
		w.WriteHeader(rsp.StatusCode)
		io.Copy(w, rsp.Body)
	}))
	t.Cleanup(srv.Close)
	return srv, &urls
}

func testSmartRecruitersAdapter(t *testing.T) (*SmartRecruitersAdapter, *[]string) {
	t.Helper()
	mock := smartrecruiters.NewMockServer()
	t.Cleanup(mock.Close)
	proxy, urls := recordingQueryProxy(t, mock.URL)
	a, err := NewSmartRecruitersAdapter(proxy.URL, &http.Client{Timeout: 5 * time.Second})
	require.NoError(t, err)
	return a, urls
}

// lastQueryParams parses the query string of the most recent upstream call.
func lastQueryParams(t *testing.T, urls []string) url.Values {
	t.Helper()
	require.NotEmpty(t, urls)
	u, err := url.Parse(urls[len(urls)-1])
	require.NoError(t, err)
	return u.Query()
}

func TestSmartRecruitersFilters(t *testing.T) {
	a, _ := testSmartRecruitersAdapter(t)
	fs, err := a.Filters(t.Context(), "equinox")
	require.NoError(t, err)

	assert.Equal(t, []string{"Hybrid", "Onsite", "Remote"}, fs["location_type"])

	deps := fs["department"]
	// 58 departments in the fixture, exactly one archived.
	assert.Len(t, deps, 57)
	assert.Contains(t, deps, "Club - Staff")
	assert.Contains(t, deps, "Club - Sales")
	assert.NotContains(t, deps, "Club - Pilot PT", "archived departments must be excluded")
	assert.True(t, slices.IsSorted(deps), "department labels must be sorted")
}

func TestSmartRecruitersRosterMirrorsProviderRoster(t *testing.T) {
	a, err := NewSmartRecruitersAdapter("https://api.smartrecruiters.com", http.DefaultClient)
	require.NoError(t, err)
	roster := a.Roster()
	require.Len(t, roster, len(smartrecruiters.Companies))
	seen := map[string]bool{}
	for _, c := range roster {
		assert.Equal(t, strings.ToLower(c.Slug), c.Slug, "slug %q must be lowercase", c.Slug)
		require.Falsef(t, seen[c.Slug], "duplicate slug %q in roster", c.Slug)
		seen[c.Slug] = true
	}
	assert.True(t, seen["equinox"], "expected equinox in roster")
}

func TestSmartRecruitersRosterBuildsRegistry(t *testing.T) {
	a, err := NewSmartRecruitersAdapter("https://api.smartrecruiters.com", http.DefaultClient)
	require.NoError(t, err)
	_, err = NewRegistry(a)
	require.NoError(t, err)
}

func TestSmartRecruitersParseCareersURL(t *testing.T) {
	a, err := NewSmartRecruitersAdapter("https://api.smartrecruiters.com", http.DefaultClient)
	require.NoError(t, err)
	tests := []struct {
		name string
		url  string
		slug string
		ok   bool
	}{
		{"roster company", "https://jobs.smartrecruiters.com/Equinox", "equinox", true},
		{"posting page", "https://jobs.smartrecruiters.com/Equinox/744000137225639-female-locker-room-associate-houston", "equinox", true},
		{"non-roster company", "https://jobs.smartrecruiters.com/SomeUnknownCo", "someunknownco", true},
		{"host only", "https://jobs.smartrecruiters.com/", "", false},
		{"other ats", "https://jobs.lever.co/acme", "", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			u, err := url.Parse(tt.url)
			require.NoError(t, err)
			slug, ok := a.ParseCareersURL(u)
			assert.Equal(t, tt.ok, ok)
			assert.Equal(t, tt.slug, slug)
		})
	}
}
