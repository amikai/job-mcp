package icims

import (
	"strings"
	"testing"

	"github.com/PuerkitoBio/goquery"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseSearchHTMLFixture(t *testing.T) {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(string(mockSearchRsp)))
	require.NoError(t, err)

	jobs, totalPages, pageSize, locs, err := parseSearchHTML(doc)
	require.NoError(t, err)
	assert.Equal(t, 1, totalPages)
	assert.Equal(t, 3, pageSize)
	require.Len(t, jobs, 3)
	assert.Equal(t, Job{
		ID:       "1977",
		Slug:     "senior-product-manager",
		Title:    "Senior Product Manager",
		Location: "US-TX-Austin",
	}, jobs[0])
	require.NotEmpty(t, locs)
	assert.Equal(t, "12781-12827-Austin", locs[0].Value)
	assert.Contains(t, locs[0].Label, "Austin")
}

func TestMatchLocationOptions(t *testing.T) {
	opts := []LocationOption{
		{Value: "12781-12827-Austin", Label: "TX Austin US"},
		{Value: "12781-12830-Lorton", Label: "VA Lorton US"},
	}
	assert.Equal(t, []string{"12781-12827-Austin"}, MatchLocationOptions(opts, "Austin"))
	assert.Equal(t, []string{"12781-12830-Lorton"}, MatchLocationOptions(opts, "12781-12830-Lorton"))
	// Broad country match must retain every US option, not collapse to one city.
	assert.Equal(t, []string{"12781-12827-Austin", "12781-12830-Lorton"}, MatchLocationOptions(opts, "US"))
	assert.Empty(t, MatchLocationOptions(opts, "Seattle"))

	// Substring trap: "us" is embedded in "Austin" but must not match the
	// value alone when the label has no separate US token.
	assert.Empty(t, MatchLocationOptions([]LocationOption{
		{Value: "1-1-Austin", Label: "Austin TX"},
	}, "US"))
	// State code must be a full token, not a substring of a city name.
	assert.Empty(t, MatchLocationOptions([]LocationOption{
		{Value: "1-1-Orlando", Label: "Orlando FL"},
	}, "OR"))
}

func TestLocationTextMatches(t *testing.T) {
	assert.True(t, locationTextMatches("US-TX-Austin", "US"))
	assert.True(t, locationTextMatches("US-TX-Austin", "Austin"))
	assert.True(t, locationTextMatches("US-VA-Lorton", "Lorton"))
	assert.False(t, locationTextMatches("US-TX-Austin", "Seattle"))
	// "us" inside "Austin" is not a token match against a city-only string
	// that lacks a US token — but "US-TX-Austin" has an explicit US token.
	assert.False(t, locationTextMatches("Austin", "US"))
}

func TestMatchLocationOption(t *testing.T) {
	opts := []LocationOption{
		{Value: "12781-12827-Austin", Label: "TX Austin US"},
		{Value: "12781-12830-Lorton", Label: "VA Lorton US"},
	}
	v, ok := MatchLocationOption(opts, "Austin")
	require.True(t, ok)
	assert.Equal(t, "12781-12827-Austin", v)

	// Multi-match convenience returns the first hit only.
	v, ok = MatchLocationOption(opts, "US")
	require.True(t, ok)
	assert.Equal(t, "12781-12827-Austin", v)

	_, ok = MatchLocationOption(opts, "Seattle")
	assert.False(t, ok)
}

func TestLooksLikeLocationValue(t *testing.T) {
	assert.True(t, LooksLikeLocationValue("12781-12827-Austin"))
	assert.False(t, LooksLikeLocationValue("Austin"))
	assert.False(t, LooksLikeLocationValue("TX Austin US"))
}

func TestParseSearchNoResults(t *testing.T) {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(string(mockSearchNoResultsRsp)))
	require.NoError(t, err)

	jobs, totalPages, pageSize, _, err := parseSearchHTML(doc)
	require.NoError(t, err)
	assert.Empty(t, jobs)
	assert.Equal(t, 0, pageSize)
	assert.Equal(t, 1, totalPages)
}

func TestParseJobDetailFixture(t *testing.T) {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(string(mockJobDetailRsp)))
	require.NoError(t, err)

	d, ok := parseJobDetailHTML(doc, "1977")
	require.True(t, ok)
	assert.Equal(t, "Senior Product Manager", d.Title)
	assert.Contains(t, d.Location, "Austin")
	assert.Contains(t, d.DescriptionHTML, "Overview")
	assert.Equal(t, "FULL_TIME", d.EmploymentType)
}

func TestParseJobDetailNotFoundBody(t *testing.T) {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(string(mockJobDetailNotFoundRsp)))
	require.NoError(t, err)

	_, ok := parseJobDetailHTML(doc, "999999999")
	assert.False(t, ok)
	assert.True(t, isSearchLikeDetailBody(doc))
}

func TestJobIDAndSlugFromHref(t *testing.T) {
	id, slug := jobIDAndSlugFromHref("https://careers-buspatrol.icims.com/jobs/1977/senior-product-manager/job?in_iframe=1")
	assert.Equal(t, "1977", id)
	assert.Equal(t, "senior-product-manager", slug)
}
