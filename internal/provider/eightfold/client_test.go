package eightfold

import (
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSearch(t *testing.T) {
	srv := NewMockServer()
	defer srv.Close()

	client, err := NewClient(srv.URL)
	require.NoError(t, err)

	res, err := client.Search(t.Context(), SearchParams{Domain: MockDomain})
	require.NoError(t, err)

	assert.Equal(t, 1330, res.Data.Count)
	require.Len(t, res.Data.Positions, 10)

	first := res.Data.Positions[0]
	assert.Equal(t, int64(MockPositionID), first.ID)
	assert.Equal(t, "Vice President - Prin Software Eng", first.Name)
	assert.Equal(t, []string{"Bengaluru, Karnataka, India"}, first.Locations)
	assert.Equal(t, "Software Engineering", first.Department.Value)
	assert.Equal(t, "hybrid", first.WorkLocationOption.Value)
	assert.Equal(t, "/careers/job/549798858854", first.PositionUrl)

	// Facet dimensions come back on every search response, not just a
	// dedicated filters endpoint — Filters() reads them from here.
	names := make([]string, 0, len(res.Data.FilterDef.SmartFilters))
	for _, f := range res.Data.FilterDef.SmartFilters {
		names = append(names, f.FilterName)
	}
	assert.Contains(t, names, "businessarea")
	assert.Contains(t, names, "employmenttype")
	assert.Contains(t, names, "city")
}

// TestSearchQueryLocation proves query and location are real server-side
// filters: count narrows from 1330 to 43 and every result matches both.
func TestSearchQueryLocation(t *testing.T) {
	srv := NewMockServer()
	defer srv.Close()

	client, err := NewClient(srv.URL)
	require.NoError(t, err)

	res, err := client.Search(t.Context(), SearchParams{
		Domain:   MockDomain,
		Query:    NewOptString("engineer"),
		Location: NewOptString("New York"),
	})
	require.NoError(t, err)

	assert.Equal(t, 43, res.Data.Count)
	require.Len(t, res.Data.Positions, 10)
	for _, p := range res.Data.Positions {
		assert.Contains(t, p.Locations[0], "New York")
	}
}

// TestSearchFiltered proves the hand-built filter_<facetName> query param
// (undocumented, recovered from live browser traffic — see
// testdata/search_filter_req.hurl) actually narrows results server-side.
func TestSearchFiltered(t *testing.T) {
	srv := NewMockServer()
	defer srv.Close()

	res, err := SearchFiltered(t.Context(), FilteredSearch{
		HTTPClient: srv.Client(),
		BaseURL:    srv.URL,
		Params:     SearchParams{Domain: MockDomain},
		Filters:    map[string][]string{"businessarea": {"technology"}},
	})
	require.NoError(t, err)

	assert.Equal(t, 112, res.Data.Count)
}

func TestPositionDetails(t *testing.T) {
	srv := NewMockServer()
	defer srv.Close()

	client, err := NewClient(srv.URL)
	require.NoError(t, err)

	res, err := client.PositionDetails(t.Context(), PositionDetailsParams{
		PositionID: MockPositionID,
		Domain:     MockDomain,
	})
	require.NoError(t, err)

	got, ok := res.(*PositionDetailsResponse)
	require.True(t, ok, "want *PositionDetailsResponse, got %T", res)

	assert.Equal(t, int64(MockPositionID), got.Data.ID)
	assert.Equal(t, "Vice President - Prin Software Eng", got.Data.Name)
	publicURL, ok := got.Data.PublicUrl.Get()
	require.True(t, ok)
	assert.Equal(t, "https://morganstanley.eightfold.ai/careers/job/"+strconv.Itoa(MockPositionID), publicURL)
	assert.Contains(t, got.Data.JobDescription, "<")
}

func TestPositionDetailsNullPublicURL(t *testing.T) {
	// Some tenants (NetApp, Houston ISD) send publicUrl: null while still
	// filling positionUrl — decode must not fail.
	const body = `{
	  "status": 200,
	  "data": {
	    "id": 42830280,
	    "name": "Global Supply Manager",
	    "locations": ["Taipei"],
	    "postedTs": 1782864873,
	    "positionUrl": "/careers/job/42830280",
	    "jobDescription": "<p>Summary</p>",
	    "publicUrl": null
	  }
	}`
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(body))
	}))
	defer srv.Close()

	client, err := NewClient(srv.URL)
	require.NoError(t, err)

	res, err := client.PositionDetails(t.Context(), PositionDetailsParams{
		PositionID: 42830280,
		Domain:     "netapp.com",
	})
	require.NoError(t, err)

	got, ok := res.(*PositionDetailsResponse)
	require.True(t, ok, "want *PositionDetailsResponse, got %T", res)
	assert.True(t, got.Data.PublicUrl.IsNull())
	assert.Equal(t, "/careers/job/42830280", got.Data.PositionUrl)
	assert.Equal(t, "Global Supply Manager", got.Data.Name)
}

func TestPositionDetailsNotFound(t *testing.T) {
	srv := NewMockServer()
	defer srv.Close()

	client, err := NewClient(srv.URL)
	require.NoError(t, err)

	res, err := client.PositionDetails(t.Context(), PositionDetailsParams{
		PositionID: 1,
		Domain:     MockDomain,
	})
	require.NoError(t, err)

	got, ok := res.(*PositionNotFoundResponse)
	require.True(t, ok, "want *PositionNotFoundResponse, got %T", res)
	assert.Equal(t, 404, got.Status)
	assert.Equal(t, "Position not found", got.Error.Message)
}
