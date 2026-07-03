package tsmc

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestJobs(t *testing.T) {
	srv := NewMockServer()
	defer srv.Close()
	c := &Client{httpClient: srv.Client(), baseURL: srv.URL}

	got, err := c.Jobs(t.Context(), &JobsRequest{
		Keyword:         "engineer",
		Locations:       []string{LocTaiwan},
		Categories:      []string{CatRD},
		JobTypes:        []string{JobTypeEngineer},
		EmploymentTypes: []string{EmployRegular},
	})
	require.NoError(t, err)

	want := &JobsResponse{Total: 22, Jobs: wantJobs}
	assert.Equal(t, want, got)
}

func TestJobDetail(t *testing.T) {
	srv := NewMockServer()
	defer srv.Close()
	c := &Client{httpClient: srv.Client(), baseURL: srv.URL}

	got, err := c.JobDetail(t.Context(), "21826")
	require.NoError(t, err)

	assert.Equal(t, wantDetail, got)
}
