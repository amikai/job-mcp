package himalayas

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBrowseJobs(t *testing.T) {
	srv := NewMockServer()
	defer srv.Close()

	client, err := NewClient(srv.URL)
	require.NoError(t, err)

	res, err := client.BrowseJobs(t.Context(), BrowseJobsParams{Limit: NewOptInt(3)})
	require.NoError(t, err)

	assert.Equal(t, 0, res.Offset)
	assert.Equal(t, 3, res.Limit)
	assert.Equal(t, 92721, res.TotalCount)
	// updatedAt/pubDate/expiryDate are Unix seconds, not the officially
	// documented milliseconds — see the deviation note in openapi.yaml.
	assert.Equal(t, int64(1784404129), res.UpdatedAt)
	require.Len(t, res.Jobs, 3)

	first := res.Jobs[0]
	assert.Equal(t, "Retail & Merchandising Coach", first.Title)
	assert.Equal(t, "Leland", first.CompanyName)
	assert.Equal(t, "leland", first.CompanySlug)
	assert.Equal(t, JobEmploymentTypeContractor, first.EmploymentType)
	assert.Equal(t, []JobSeniorityItem{JobSeniorityItemMidLevel}, first.Seniority)
	assert.Equal(t, JobSalaryPeriodAnnual, first.SalaryPeriod)
	// No disclosed salary: minSalary/maxSalary and currency are all null.
	assert.True(t, first.MinSalary.Null)
	assert.True(t, first.MaxSalary.Null)
	assert.True(t, first.Currency.Null)
	// locationRestrictions is plain country-name strings, and
	// timezoneRestrictions numeric UTC offsets — not the officially
	// documented object/string shapes.
	assert.Equal(t, []string{"United States"}, first.LocationRestrictions)
	assert.Equal(t, []float64{-10, -9, -8, -7, -6, -5, 14}, first.TimezoneRestrictions)
	assert.Equal(t, int64(1784404129), first.PubDate)
	assert.Equal(t, int64(1789588129), first.ExpiryDate)
	assert.Contains(t, first.Description, "<p")
	// guid doubles as the job's public posting URL and equals
	// applicationLink.
	assert.Equal(t, "https://himalayas.app/companies/leland/jobs/retail-merchandising-coach", first.GUID)
	assert.Equal(t, first.ApplicationLink, first.GUID)

	// The third job discloses a currency without salary bounds.
	third := res.Jobs[2]
	assert.Equal(t, "newfire-global-partners", third.CompanySlug)
	assert.Equal(t, "USD", third.Currency.Value)
	assert.True(t, third.MinSalary.Null)
}

func TestSearchJobs(t *testing.T) {
	srv := NewMockServer()
	defer srv.Close()

	client, err := NewClient(srv.URL)
	require.NoError(t, err)

	res, err := client.SearchJobs(t.Context(), SearchJobsParams{
		Q:    NewOptString("golang"),
		Sort: NewOptSearchJobsSort(SearchJobsSortRecent),
	})
	require.NoError(t, err)

	got, ok := res.(*JobsResponse)
	require.True(t, ok, "want *JobsResponse, got %T", res)

	assert.Equal(t, 97, got.TotalCount)
	require.Len(t, got.Jobs, 20)
	assert.Equal(t, "Staff Software Engineer, Golang", got.Jobs[0].Title)
	assert.Equal(t, "block-labs", got.Jobs[0].CompanySlug)

	// Fractional UTC offsets like -3.5 and 5.5 decode as float64s.
	assert.Contains(t, got.Jobs[1].TimezoneRestrictions, -3.5)
	assert.Contains(t, got.Jobs[1].TimezoneRestrictions, 5.5)
}

// TestSearchJobsFiltered proves country/employment_type/seniority are
// modeled as real server-side filters: the fixture's totalCount narrows
// from 97 (q=golang alone) to 18, and every returned job is Full Time
// at Senior level.
func TestSearchJobsFiltered(t *testing.T) {
	srv := NewMockServer()
	defer srv.Close()

	client, err := NewClient(srv.URL)
	require.NoError(t, err)

	res, err := client.SearchJobs(t.Context(), SearchJobsParams{
		Q:              NewOptString("golang"),
		Country:        NewOptString("US"),
		EmploymentType: NewOptString("Full Time"),
		Seniority:      NewOptString("Senior"),
	})
	require.NoError(t, err)

	got, ok := res.(*JobsResponse)
	require.True(t, ok, "want *JobsResponse, got %T", res)

	assert.Equal(t, 18, got.TotalCount)
	// A page may hold fewer than 20 jobs even when totalCount says more
	// remain — see the pagination quirk in openapi.yaml.
	require.Len(t, got.Jobs, 15)
	for _, j := range got.Jobs {
		assert.Equal(t, JobEmploymentTypeFullTime, j.EmploymentType)
		assert.Contains(t, j.Seniority, JobSeniorityItemSenior)
	}
}

func TestSearchJobsByCompany(t *testing.T) {
	srv := NewMockServer()
	defer srv.Close()

	client, err := NewClient(srv.URL)
	require.NoError(t, err)

	res, err := client.SearchJobs(t.Context(), SearchJobsParams{
		Company: NewOptString("leland"),
	})
	require.NoError(t, err)

	got, ok := res.(*JobsResponse)
	require.True(t, ok, "want *JobsResponse, got %T", res)

	assert.Equal(t, 133, got.TotalCount)
	require.Len(t, got.Jobs, 20)
	for _, j := range got.Jobs {
		assert.Equal(t, "leland", j.CompanySlug)
	}
}

func TestSearchJobsInvalidCountry(t *testing.T) {
	srv := NewMockServer()
	defer srv.Close()

	client, err := NewClient(srv.URL)
	require.NoError(t, err)

	res, err := client.SearchJobs(t.Context(), SearchJobsParams{
		Country: NewOptString("Narnia"),
	})
	require.NoError(t, err)

	got, ok := res.(*SearchError)
	require.True(t, ok, "want *SearchError, got %T", res)

	assert.False(t, got.Ok)
	assert.Equal(t, "Invalid country", got.Errors)
}

// TestSearchJobsUnknownCompany guards the 400-not-empty quirk: an
// unrecognized company slug is rejected as an invalid filter, not answered
// with an empty 200 result.
func TestSearchJobsUnknownCompany(t *testing.T) {
	srv := NewMockServer()
	defer srv.Close()

	client, err := NewClient(srv.URL)
	require.NoError(t, err)

	res, err := client.SearchJobs(t.Context(), SearchJobsParams{
		Company: NewOptString(MockUnknownCompany),
	})
	require.NoError(t, err)

	got, ok := res.(*SearchError)
	require.True(t, ok, "want *SearchError, got %T", res)

	assert.False(t, got.Ok)
	assert.Equal(t, "Invalid company", got.Errors)
}
