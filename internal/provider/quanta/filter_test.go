package quanta

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
	return res.GetJobResult()
}

func TestFilterJobs(t *testing.T) {
	jobs := fixtureJobs(t)

	t.Run("zero options return everything", func(t *testing.T) {
		assert.Len(t, FilterJobs(jobs, FilterOptions{}), 239)
	})

	t.Run("keyword matches title", func(t *testing.T) {
		got := FilterJobs(jobs, FilterOptions{Keyword: "伺服器"})
		require.Len(t, got, 68)
		assert.Equal(t, "J0106", got[0].GetJobCode())
	})

	t.Run("keyword matches requirements, including nil-safe rows", func(t *testing.T) {
		got := FilterJobs(jobs, FilterOptions{Keyword: "多益"})
		require.Len(t, got, 48)
		assert.Equal(t, "J0106", got[0].GetJobCode())
	})

	t.Run("keyword is case-insensitive", func(t *testing.T) {
		got := FilterJobs(jobs, FilterOptions{Keyword: "SOFTWARE ENGINEER"})
		assert.NotEmpty(t, got)
	})

	t.Run("location ids narrow to a single location", func(t *testing.T) {
		assert.Len(t, FilterJobs(jobs, FilterOptions{LocationIDs: []string{"3"}}), 173)
	})

	t.Run("location ids are a set (OR within the set)", func(t *testing.T) {
		single := FilterJobs(jobs, FilterOptions{LocationIDs: []string{"3"}})
		union := FilterJobs(jobs, FilterOptions{LocationIDs: []string{"3", "0"}})
		assert.Greater(t, len(union), len(single))
	})

	t.Run("category ids narrow to a single category", func(t *testing.T) {
		assert.Len(t, FilterJobs(jobs, FilterOptions{CategoryIDs: []string{"1"}}), 29)
	})

	t.Run("category ids are a set (OR within the set)", func(t *testing.T) {
		assert.Len(t, FilterJobs(jobs, FilterOptions{CategoryIDs: []string{"1", "2"}}), 62)
	})

	t.Run("fields AND together", func(t *testing.T) {
		got := FilterJobs(jobs, FilterOptions{Keyword: "伺服器", LocationIDs: []string{"3"}})
		assert.Len(t, got, 53)
	})

	t.Run("no match returns nil, not the whole dump", func(t *testing.T) {
		assert.Empty(t, FilterJobs(jobs, FilterOptions{Keyword: "this keyword does not exist anywhere"}))
	})
}

func TestFindBySerial(t *testing.T) {
	jobs := fixtureJobs(t)

	t.Run("hit", func(t *testing.T) {
		j, ok := FindBySerial(jobs, "106")
		require.True(t, ok)
		assert.Equal(t, "J0106", j.GetJobCode())
		assert.Equal(t, "雲端伺服器硬體設計", j.GetTitle())
	})

	t.Run("hit with nil requirements", func(t *testing.T) {
		j, ok := FindBySerial(jobs, "2194")
		require.True(t, ok)
		assert.True(t, j.GetRequirements().IsNull())
	})

	t.Run("miss", func(t *testing.T) {
		_, ok := FindBySerial(jobs, "no-such-serial")
		assert.False(t, ok)
	})
}
