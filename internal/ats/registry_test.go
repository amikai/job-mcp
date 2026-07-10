package ats

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fakeAdapter satisfies Adapter with a canned roster; search methods are
// never reached by registry tests.
type fakeAdapter struct {
	name   string
	roster []CompanyInfo
}

func (f *fakeAdapter) Name() string          { return f.name }
func (f *fakeAdapter) Roster() []CompanyInfo { return f.roster }
func (f *fakeAdapter) Search(context.Context, string, SearchParams) (*SearchResult, error) {
	return nil, nil
}
func (f *fakeAdapter) Filters(context.Context, string) (FilterSet, error) { return nil, nil }
func (f *fakeAdapter) Detail(context.Context, string, string) (*JobDetail, error) {
	return nil, nil
}

func testRegistry(t *testing.T) *Registry {
	t.Helper()
	r, err := NewRegistry(
		&fakeAdapter{name: "workday", roster: []CompanyInfo{
			{Slug: "nvidia", Name: "NVIDIA Corp"},
			{Slug: "workday", Name: "Workday, Inc."},
		}},
		&fakeAdapter{name: "lever", roster: []CompanyInfo{
			{Slug: "palantir", Name: "Palantir Technologies"},
		}},
	)
	require.NoError(t, err)
	return r
}

func TestResolveBySlug(t *testing.T) {
	r := testRegistry(t)
	a, slug, err := r.Resolve("nvidia")
	require.NoError(t, err)
	assert.Equal(t, "workday", a.Name())
	assert.Equal(t, "nvidia", slug)
}

func TestResolveByDisplayName(t *testing.T) {
	r := testRegistry(t)
	// Case, punctuation, and spaces must not matter.
	for _, input := range []string{"NVIDIA Corp", "nvidia corp", "Workday, Inc.", "workday inc"} {
		_, _, err := r.Resolve(input)
		assert.NoErrorf(t, err, "Resolve(%q)", input)
	}
}

func TestResolveUnknownTeaches(t *testing.T) {
	r := testRegistry(t)
	_, _, err := r.Resolve("palantir tech")
	require.ErrorContains(t, err, "palantir", "suggestions should contain the input")
	assert.ErrorContains(t, err, "3 companies", "error should state supported count")
}

func TestResolveEmpty(t *testing.T) {
	r := testRegistry(t)
	_, _, err := r.Resolve("  ")
	assert.Error(t, err, "want error for empty company")
}

func TestNewRegistryRejectsDuplicateSlug(t *testing.T) {
	_, err := NewRegistry(
		&fakeAdapter{name: "workday", roster: []CompanyInfo{{Slug: "acme", Name: "Acme (Workday)"}}},
		&fakeAdapter{name: "lever", roster: []CompanyInfo{{Slug: "acme", Name: "Acme (Lever)"}}},
	)
	assert.Error(t, err, "want error for duplicate slug across adapters")
}
