package jobicy

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetRemoteJobs(t *testing.T) {
	srv := NewMockServer()
	defer srv.Close()

	client, err := NewClient(srv.URL)
	require.NoError(t, err)

	res, err := client.GetRemoteJobs(t.Context(), GetRemoteJobsParams{Count: NewOptInt(3)})
	require.NoError(t, err)

	sum, ok := res.(*GetRemoteJobsOK)
	require.True(t, ok, "want *GetRemoteJobsOK, got %T", res)
	got, ok := sum.GetJobsResponse()
	require.True(t, ok, "want JobsResponse variant, got %s", sum.Type)

	assert.Equal(t, 3, got.JobCount)
	assert.True(t, got.Success)
	assert.Equal(t, "2026-07-18T09:40:07+00:00", got.LastUpdate)
	assert.Equal(t, AppliedFilters{Count: NewOptInt(3)}, got.AppliedFilters)
	assert.False(t, got.StatusCode.Set)

	require.Len(t, got.Jobs, 3)
	first := got.Jobs[0]
	assert.Equal(t, 144808, first.ID)
	assert.Equal(t, "https://jobicy.com/jobs/144808-senior-cloud-engineer-aws", first.URL)
	assert.Equal(t, "144808-senior-cloud-engineer-aws", first.JobSlug)
	assert.Equal(t, "Senior Cloud Engineer (AWS)", first.JobTitle)
	assert.Equal(t, "Leidos", first.CompanyName)
	// Display names carry HTML entities, the quirk documented on the Job
	// schema.
	assert.Equal(t, []string{"DevOps &amp; Infrastructure"}, first.JobIndustry)
	assert.Equal(t, []string{"Full-Time"}, first.JobType)
	assert.Equal(t, "USA", first.JobGeo)
	assert.Equal(t, "Senior", first.JobLevel)
	assert.Contains(t, first.JobExcerpt, "Senior Cloud Engineer")
	assert.Contains(t, first.JobDescription, "<p>")
	assert.Equal(t, "2026-07-18T09:40:07+00:00", first.PubDate)
	assert.Equal(t, NewOptFloat64(107900), first.SalaryMin)
	assert.Equal(t, NewOptFloat64(195050), first.SalaryMax)
	assert.Equal(t, NewOptString("USD"), first.SalaryCurrency)
	assert.Equal(t, NewOptString("yearly"), first.SalaryPeriod)

	// Salary fields are optional per row within the same response.
	assert.False(t, got.Jobs[2].SalaryMin.Set)
	assert.False(t, got.Jobs[2].SalaryMax.Set)
}

func TestGetRemoteJobsFiltered(t *testing.T) {
	srv := NewMockServer()
	defer srv.Close()

	client, err := NewClient(srv.URL)
	require.NoError(t, err)

	res, err := client.GetRemoteJobs(t.Context(), GetRemoteJobsParams{
		Count:    NewOptInt(3),
		Geo:      NewOptString("usa"),
		Industry: NewOptString("dev"),
		Tag:      NewOptString("golang"),
	})
	require.NoError(t, err)

	sum, ok := res.(*GetRemoteJobsOK)
	require.True(t, ok, "want *GetRemoteJobsOK, got %T", res)
	got, ok := sum.GetJobsResponse()
	require.True(t, ok, "want JobsResponse variant, got %s", sum.Type)

	assert.Equal(t, AppliedFilters{
		Count:    NewOptInt(3),
		Geo:      NewOptString("usa"),
		Industry: NewOptString("dev"),
		Tag:      NewOptString("golang"),
	}, got.AppliedFilters)
	require.Len(t, got.Jobs, 3)
	assert.Equal(t, 145145, got.Jobs[0].ID)
	assert.Equal(t, "MTS Cloud Test", got.Jobs[0].JobTitle)
	assert.Equal(t, "Aviatrix", got.Jobs[0].CompanyName)
}

func TestGetRemoteJobsEmpty(t *testing.T) {
	srv := NewMockServer()
	defer srv.Close()

	client, err := NewClient(srv.URL)
	require.NoError(t, err)

	res, err := client.GetRemoteJobs(t.Context(), GetRemoteJobsParams{
		Count: NewOptInt(3),
		Tag:   NewOptString("zzzznomatchzzz"),
	})
	require.NoError(t, err)

	// Zero matches is HTTP 404 but still the jobs envelope, so the client
	// returns it as a result, not an error.
	got, ok := res.(*JobsResponse)
	require.True(t, ok, "want *JobsResponse, got %T", res)
	assert.False(t, got.Success)
	assert.Equal(t, NewOptInt(404), got.StatusCode)
	assert.Equal(t, NewOptString("Nothing found. Please modify or simplify your request."), got.Message)
	assert.Equal(t, 0, got.JobCount)
	assert.Empty(t, got.Jobs)
	assert.Equal(t, "", got.LastUpdate)
}

func TestGetRemoteJobsInvalidIndustry(t *testing.T) {
	srv := NewMockServer()
	defer srv.Close()

	client, err := NewClient(srv.URL)
	require.NoError(t, err)

	_, err = client.GetRemoteJobs(t.Context(), GetRemoteJobsParams{
		Count:    NewOptInt(3),
		Industry: NewOptString("not-a-real-industry"),
	})
	apiErr, ok := errors.AsType[*ErrorResponseStatusCode](err)
	require.True(t, ok, "want *ErrorResponseStatusCode, got %T", err)
	assert.Equal(t, 400, apiErr.StatusCode)
	assert.False(t, apiErr.Response.Success)
	assert.Contains(t, apiErr.Response.Error, "Invalid 'industry' value")
}

func TestGetLocations(t *testing.T) {
	srv := NewMockServer()
	defer srv.Close()

	client, err := NewClient(srv.URL)
	require.NoError(t, err)

	res, err := client.GetRemoteJobs(t.Context(), GetRemoteJobsParams{
		Get: NewOptGetRemoteJobsGet(GetRemoteJobsGetLocations),
	})
	require.NoError(t, err)

	sum, ok := res.(*GetRemoteJobsOK)
	require.True(t, ok, "want *GetRemoteJobsOK, got %T", res)
	got, ok := sum.GetLocationsResponse()
	require.True(t, ok, "want LocationsResponse variant, got %s", sum.Type)

	require.Len(t, got.Locations, 54)
	assert.Equal(t, Location{GeoID: 446, GeoName: "APAC", GeoSlug: "apac"}, got.Locations[0])
}

func TestGetIndustries(t *testing.T) {
	srv := NewMockServer()
	defer srv.Close()

	client, err := NewClient(srv.URL)
	require.NoError(t, err)

	res, err := client.GetRemoteJobs(t.Context(), GetRemoteJobsParams{
		Get: NewOptGetRemoteJobsGet(GetRemoteJobsGetIndustries),
	})
	require.NoError(t, err)

	sum, ok := res.(*GetRemoteJobsOK)
	require.True(t, ok, "want *GetRemoteJobsOK, got %T", res)
	got, ok := sum.GetIndustriesResponse()
	require.True(t, ok, "want IndustriesResponse variant, got %s", sum.Type)

	require.Len(t, got.Industries, 27)
	assert.Equal(t, Industry{
		IndustryID:   6346,
		IndustryName: "Admin & Virtual Assistant",
		IndustrySlug: "admin-support",
	}, got.Industries[0])
}
