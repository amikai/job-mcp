package quanta

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

	// The dump is the whole board: no pagination, no server-side
	// filtering. See the full-dump note in openapi.yaml.
	jobs := res.GetJobResult()
	require.Len(t, jobs, 239)

	first := jobs[0]
	assert.Equal(t, "J0106", first.GetJobCode())
	assert.Equal(t, "106", first.GetSerial())
	assert.Equal(t, "雲端伺服器硬體設計", first.GetTitle())
	assert.Equal(t, "硬體研發", first.GetCategoryName())
	assert.Equal(t, "林口研發中心", first.GetLocationName())
	req, ok := first.GetRequirements().Get()
	require.True(t, ok)
	assert.Contains(t, req, "大學以上電子/電機相關科系畢")

	// requnm is null on exactly the two disability-quota rows at capture
	// time; confirm both decode without error and report the null.
	for _, serial := range []string{"2194", "2290"} {
		j, ok := FindBySerial(jobs, serial)
		require.True(t, ok, "serial %s should be present in the fixture", serial)
		assert.True(t, j.GetRequirements().IsNull())
	}
}
