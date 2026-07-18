package remotive

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fixtureJobs decodes the captured dump once per test via the mock server,
// so filter expectations stay pinned to real data.
func fixtureJobs(t *testing.T) []Job {
	t.Helper()
	srv := NewMockServer()
	defer srv.Close()

	client, err := NewClient(srv.URL)
	require.NoError(t, err)
	res, err := client.ListJobs(t.Context())
	require.NoError(t, err)
	return res.Jobs
}

func TestFilterJobs(t *testing.T) {
	jobs := fixtureJobs(t)

	t.Run("zero options return everything", func(t *testing.T) {
		assert.Len(t, FilterJobs(jobs, FilterOptions{}), 42)
	})

	t.Run("keyword searches title, tags, and description", func(t *testing.T) {
		got := FilterJobs(jobs, FilterOptions{Keyword: "devops"})
		require.Len(t, got, 3)
		// Only the first match carries it in the title; the others match
		// via tags/description.
		assert.Equal(t, "Senior DevOps Engineer", got[0].Title)
	})

	t.Run("category matches display name", func(t *testing.T) {
		assert.Len(t, FilterJobs(jobs, FilterOptions{Category: "Software Development"}), 10)
	})

	t.Run("category matches slug too", func(t *testing.T) {
		assert.Len(t, FilterJobs(jobs, FilterOptions{Category: "software-development"}), 10)
	})

	t.Run("company is a case-insensitive substring", func(t *testing.T) {
		got := FilterJobs(jobs, FilterOptions{Company: "a.team"})
		require.Len(t, got, 2)
		for _, j := range got {
			assert.Equal(t, "A.Team", j.CompanyName)
		}
	})

	t.Run("job type is exact", func(t *testing.T) {
		assert.Len(t, FilterJobs(jobs, FilterOptions{JobType: "contract"}), 3)
		assert.Empty(t, FilterJobs(jobs, FilterOptions{JobType: "contractor"}))
	})

	t.Run("location substring", func(t *testing.T) {
		assert.Len(t, FilterJobs(jobs, FilterOptions{Location: "usa"}), 7)
	})

	t.Run("fields AND together", func(t *testing.T) {
		// Keyword alone matches 3 jobs across 3 categories; the category
		// filter narrows those to 1.
		got := FilterJobs(jobs, FilterOptions{Keyword: "devops", Category: "artificial-intelligence"})
		require.Len(t, got, 1)
		assert.Equal(t, "Senior AI Engineer", got[0].Title)
	})
}
