package remotive

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

	// The dump is the whole free-tier board: job-count == total-job-count
	// == len(jobs). See the full-dump note in openapi.yaml.
	assert.Equal(t, 42, res.JobCount)
	assert.Equal(t, NewOptInt(42), res.TotalJobCount)
	require.Len(t, res.Jobs, 42)
	assert.Contains(t, res.LegalNotice.Value, "Remotive's API")
	assert.Contains(t, res.Warning.Value, "remotive.com")

	first := res.Jobs[0]
	assert.Equal(t, 2091069, first.ID)
	assert.Equal(t, "Patient Care Specialist", first.Title)
	assert.Equal(t, "STATLINX", first.CompanyName)
	assert.Equal(t, "https://remotive.com/remote-jobs/medical/patient-care-specialist-2091069", first.URL)
	assert.Equal(t, NewOptString("https://remotive.com/job/2091069/logo"), first.CompanyLogo)
	assert.Equal(t, "Medical", first.Category)
	assert.Equal(t, []string{"research", "insurance"}, first.Tags)
	assert.Equal(t, "full_time", first.JobType)
	// No timezone offset — plain string on purpose, see openapi.yaml.
	assert.Equal(t, "2026-07-16T13:28:02", first.PublicationDate)
	assert.Equal(t, "USA", first.CandidateRequiredLocation)
	assert.Equal(t, "$36k", first.Salary)
	assert.Contains(t, first.Description, "STATLINX")
	// company_logo_url appears on only some jobs (21 of 42 in the fixture):
	// absent on the first, present on the second.
	assert.False(t, first.CompanyLogoURL.Set)
	assert.Equal(t, NewOptString("https://remotive.com/job/1919265/logo"), res.Jobs[1].CompanyLogoURL)

	// salary is always present but sometimes empty.
	assert.Equal(t, "Senior DevOps Engineer", res.Jobs[5].Title)
	assert.Empty(t, res.Jobs[5].Salary)
}

// TestListCategories guards the envelope-reuse quirk: the category list
// arrives under the `jobs` key and job-count holds the category count.
func TestListCategories(t *testing.T) {
	srv := NewMockServer()
	defer srv.Close()

	client, err := NewClient(srv.URL)
	require.NoError(t, err)

	res, err := client.ListCategories(t.Context())
	require.NoError(t, err)

	assert.Equal(t, NewOptInt(30), res.JobCount)
	require.Len(t, res.Jobs, 30)
	assert.Equal(t, Category{ID: 19, Name: "Software Development", Slug: "software-development"}, res.Jobs[0])
	assert.Equal(t, Category{ID: 18, Name: "Customer Service", Slug: "customer-service"}, res.Jobs[1])
}
