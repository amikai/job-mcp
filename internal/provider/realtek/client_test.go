package realtek

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestListJobs(t *testing.T) {
	srv := NewMockServer()
	defer srv.Close()

	client, err := NewClient(srv.URL)
	require.NoError(t, err)

	res, err := client.ListJobs(t.Context())
	require.NoError(t, err)

	assert.True(t, res.Pass)
	require.NotEmpty(t, res.Data)

	first := res.Data[0]
	assert.Equal(t, "18", first.JobOppId)
	assert.Equal(t, "22", first.JobId)
	assert.Equal(t, "4", first.JobTypeId)
	assert.Equal(t, "1", first.JobLocationId)
	assert.Equal(t, "IC Layout工程師", first.JobTitle)
	assert.Equal(t, "設計技術服務", first.JobType)
	assert.Equal(t, "大學", first.Degree)
	assert.Equal(t, "0", first.Exp)
	assert.Equal(t, "新竹科學園區", first.Location)
	_, ok := first.ModDate.Get()
	assert.False(t, ok, "ModDate has been null on every observed job summary")
}

// TestFilterJobs proves the form-urlencoded keyword filter is a real
// server-side narrowing, matching the site's startJobSearch() semantics.
func TestFilterJobs(t *testing.T) {
	srv := NewMockServer()
	defer srv.Close()

	client, err := NewClient(srv.URL)
	require.NoError(t, err)

	res, err := client.FilterJobs(t.Context(), &FilterJobsReq{
		Keyword: NewOptString("verification"),
		Xp:      NewOptString("-1"),
	})
	require.NoError(t, err)

	assert.True(t, res.Pass)
	// The keyword matches substrings of title/requirement server-side;
	// requirement text isn't exposed on this list endpoint, so not every
	// title visibly contains the keyword, but the result set (43) is
	// narrower than the capped dump (200).
	assert.Len(t, res.Data, 43)
}

func TestListJobTypes(t *testing.T) {
	srv := NewMockServer()
	defer srv.Close()

	client, err := NewClient(srv.URL)
	require.NoError(t, err)

	res, err := client.ListJobTypes(t.Context())
	require.NoError(t, err)

	assert.True(t, res.Pass)
	require.NotEmpty(t, res.Data)
	assert.Equal(t, "3", res.Data[0].JobTypeId)
	assert.Equal(t, "財務、會計", res.Data[0].JobType)
}

func TestListJobLocations(t *testing.T) {
	srv := NewMockServer()
	defer srv.Close()

	client, err := NewClient(srv.URL)
	require.NoError(t, err)

	res, err := client.ListJobLocations(t.Context())
	require.NoError(t, err)

	assert.True(t, res.Pass)
	require.NotEmpty(t, res.Data)
	assert.Equal(t, "1", res.Data[0].JobLocationId)
	assert.Equal(t, "新竹科學園區", res.Data[0].JobLocation)
}

func TestGetVacancyDetail(t *testing.T) {
	srv := NewMockServer()
	defer srv.Close()

	client, err := NewClient(srv.URL)
	require.NoError(t, err)

	res, err := client.GetVacancyDetail(t.Context(), GetVacancyDetailParams{JobOppId: "18"})
	require.NoError(t, err)

	assert.True(t, res.Pass)
	title, ok := res.Data.JobTitle.Get()
	require.True(t, ok, "want JobTitle populated for a known JobOppId")
	assert.Equal(t, "IC Layout工程師", title)

	requirement, ok := res.Data.Requirement.Get()
	require.True(t, ok, "want Requirement populated for a known JobOppId")
	assert.NotEmpty(t, requirement)

	degree, ok := res.Data.Degree.Get()
	require.True(t, ok)
	assert.Equal(t, "大學", degree)
}

// TestGetVacancyDetailNotFound guards the no-404 quirk: an unrecognized
// JobOppId is HTTP 200 with Pass true and an all-null Data object;
// JobTitle == null is the not-found signal.
func TestGetVacancyDetailNotFound(t *testing.T) {
	srv := NewMockServer()
	defer srv.Close()

	client, err := NewClient(srv.URL)
	require.NoError(t, err)

	res, err := client.GetVacancyDetail(t.Context(), GetVacancyDetailParams{JobOppId: "999999"})
	require.NoError(t, err)

	assert.True(t, res.Pass)
	_, ok := res.Data.JobTitle.Get()
	assert.False(t, ok, "want JobTitle null for an unknown JobOppId")
}
