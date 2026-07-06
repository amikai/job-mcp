package lever

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newMockServer wraps the package mock server with per-request assertions
// on the exact query encoding the generated client produces.
func newMockServer(t *testing.T) *httptest.Server {
	t.Helper()
	mux := http.NewServeMux()
	mux.HandleFunc("/v0/postings/{site}", func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodGet, r.Method)
		assert.Equal(t, "leverdemo", r.PathValue("site"))
		q := r.URL.Query()
		assert.Equal(t, "json", q.Get("mode"))
		assert.Equal(t, "0", q.Get("skip"))
		assert.Equal(t, "3", q.Get("limit"))
		assert.Equal(t, []string{"Arlington, TX", "New York, NY"}, q["location"])
		assert.Equal(t, []string{"Customer Success"}, q["department"])
		serveMockJSON(mockPostingsRsp)(w, r)
	})
	mux.HandleFunc("/v0/postings/{site}/{postingId}", func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodGet, r.Method)
		assert.Equal(t, "leverdemo", r.PathValue("site"))
		assert.Equal(t, "33538a2f-d27d-4a96-8f05-fa4b0e4d940e", r.PathValue("postingId"))
		serveMockJSON(mockPostingDetailRsp)(w, r)
	})
	return httptest.NewServer(mux)
}

func TestListPostings(t *testing.T) {
	srv := newMockServer(t)
	defer srv.Close()

	c, err := NewClient(srv.URL, WithClient(srv.Client()))
	require.NoError(t, err)

	got, err := c.ListPostings(t.Context(), ListPostingsParams{
		Site:       "leverdemo",
		Mode:       ListPostingsModeJSON,
		Skip:       NewOptInt(0),
		Limit:      NewOptInt(3),
		Location:   []string{"Arlington, TX", "New York, NY"},
		Department: []string{"Customer Success"},
	})
	require.NoError(t, err)
	require.Len(t, got, 3)

	first := got[0]
	assert.Equal(t, "33538a2f-d27d-4a96-8f05-fa4b0e4d940e", first.ID)
	assert.Equal(t, "AbelsonTaylor Writer", first.Text)
	assert.Equal(t, NewOptNilString("US"), first.Country)
	assert.Equal(t, NewOptInt64(1553186035299), first.CreatedAt)
	assert.Equal(t, NewOptPostingWorkplaceType(PostingWorkplaceTypeHybrid), first.WorkplaceType)
	assert.Equal(t, NewOptString("https://jobs.lever.co/leverdemo/33538a2f-d27d-4a96-8f05-fa4b0e4d940e"), first.HostedUrl)
	assert.Equal(t, NewOptString("https://jobs.lever.co/leverdemo/33538a2f-d27d-4a96-8f05-fa4b0e4d940e/apply"), first.ApplyUrl)

	cats := first.Categories.Value
	assert.Equal(t, NewOptString("Arlington, TX"), cats.Location)
	assert.Equal(t, NewOptString("Regular Full Time (Salary)"), cats.Commitment)
	assert.Equal(t, NewOptString("Professional Services"), cats.Team)
	assert.Equal(t, NewOptString("Customer Success"), cats.Department)
	assert.Equal(t, []string{"Arlington, TX"}, cats.AllLocations)

	require.NotEmpty(t, first.Lists)
	assert.Equal(t, "Qualifications", first.Lists[0].Text)
}

func TestGetPosting(t *testing.T) {
	srv := newMockServer(t)
	defer srv.Close()

	c, err := NewClient(srv.URL, WithClient(srv.Client()))
	require.NoError(t, err)

	got, err := c.GetPosting(t.Context(), GetPostingParams{
		Site:      "leverdemo",
		PostingId: "33538a2f-d27d-4a96-8f05-fa4b0e4d940e",
	})
	require.NoError(t, err)
	assert.Equal(t, "33538a2f-d27d-4a96-8f05-fa4b0e4d940e", got.ID)
	assert.Equal(t, "AbelsonTaylor Writer", got.Text)
}
