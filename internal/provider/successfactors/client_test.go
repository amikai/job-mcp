package successfactors

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSearch(t *testing.T) {
	srv := NewMockServer()
	defer srv.Close()
	c := NewClient(srv.URL, srv.Client())

	got, err := c.Search(t.Context(), &SearchRequest{Query: "engineer"})
	require.NoError(t, err)

	assert.Equal(t, 633, got.TotalCount)
	assert.Len(t, got.Jobs, 25)
	assert.Equal(t, Job{ID: "1414343333", Title: "Developer Associate", Location: "Bangalore, IN, 560066"}, got.Jobs[0])
}

func TestSearchFiltered(t *testing.T) {
	srv := NewMockServer()
	defer srv.Close()
	c := NewClient(srv.URL, srv.Client())

	got, err := c.Search(t.Context(), &SearchRequest{
		Query:      "engineer",
		Department: "Software-Design and Development",
		Country:    "DE",
	})
	require.NoError(t, err)
	assert.NotEmpty(t, got.Jobs)
}

func TestSearchNoResults(t *testing.T) {
	srv := NewMockServer()
	defer srv.Close()
	c := NewClient(srv.URL, srv.Client())

	got, err := c.Search(t.Context(), &SearchRequest{Query: "zzzznonexistentkeyword12345"})
	require.NoError(t, err)
	assert.Empty(t, got.Jobs)
	assert.Equal(t, 0, got.TotalCount)
}

func TestJobDetail(t *testing.T) {
	srv := NewMockServer()
	defer srv.Close()
	c := NewClient(srv.URL, srv.Client())

	got, err := c.JobDetail(t.Context(), "1414343333")
	require.NoError(t, err)

	assert.Equal(t, "1414343333", got.ID)
	assert.Equal(t, "Developer Associate", got.Title)
	assert.Equal(t, "Bangalore, IN, 560066", got.Location)
	assert.Equal(t, "SAP", got.Employer)
	assert.NotEmpty(t, got.DescriptionHTML)
}

func TestJobDetailNotFound(t *testing.T) {
	srv := NewMockServer()
	defer srv.Close()
	c := NewClient(srv.URL, srv.Client())

	_, err := c.JobDetail(t.Context(), "999999999")
	require.Error(t, err)
}

func TestFacetValues(t *testing.T) {
	srv := NewMockServer()
	defer srv.Close()
	c := NewClient(srv.URL, srv.Client())

	got, err := c.FacetValues(t.Context(), &SearchRequest{Query: "engineer"})
	require.NoError(t, err)

	require.Contains(t, got.Facets, "country")
	require.Contains(t, got.Facets, "department")
	assert.Contains(t, got.Facets["country"], FacetOption{Name: "DE", Translated: "Germany", Count: 192})
	assert.Contains(t, got.Facets["department"], FacetOption{Name: "Software-Design and Development", Translated: "", Count: 208})
}
