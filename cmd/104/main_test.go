package main

import (
	"bytes"
	"strings"
	"testing"

	"github.com/amikai/job-mcp/internal/provider/job104"
	"github.com/stretchr/testify/assert"
)

func TestJobCodeFromURL(t *testing.T) {
	got := jobCodeFromURL("https://www.104.com.tw/job/abc123?jobsource=foo")
	if got != "abc123" {
		t.Fatalf("jobCodeFromURL() = %q", got)
	}
}

func TestBuildJobsRequestUnfilteredByDefault(t *testing.T) {
	got := buildJobsRequest("Golang", "", "", "", "", "", "", 0)
	want := &job104.JobsRequest{Keyword: "Golang"}
	assert.Equal(t, want, got)
}

func TestBuildJobsRequestResolvesLabels(t *testing.T) {
	got := buildJobsRequest("Golang", "Taipei", "Full-time", "Newest", "edu-code", "Partial", "s9-code", 2)

	ro := job104.ROFullTime
	order := job104.OrderNewest
	remoteWork := job104.RemoteWorkPartial
	page := 2
	want := &job104.JobsRequest{
		Keyword:    "Golang",
		Area:       job104.AreaTaipei,
		RO:         &ro,
		Order:      &order,
		Page:       &page,
		Edu:        "edu-code",
		RemoteWork: &remoteWork,
		S9:         "s9-code",
	}
	assert.Equal(t, want, got)
}

func TestBuildJobsRequestPageZeroLeavesPageNil(t *testing.T) {
	got := buildJobsRequest("", "", "", "", "", "", "", 0)
	assert.Nil(t, got.Page)
}

func TestWriteDetail(t *testing.T) {
	d := detail("Go Engineer", "Build Go services")
	d.Data.JobDetail.Salary = "60k-80k"
	d.Data.Condition.WorkExp = "3 years"
	d.Data.Condition.Edu = "Bachelor"

	var buf bytes.Buffer
	writeDetail(&buf, d)
	got := buf.String()

	for _, want := range []string{
		"Salary: 60k-80k",
		"Experience: 3 years | Education: Bachelor",
		"Build Go services",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("writeDetail missing %q:\n%s", want, got)
		}
	}
}

func detail(title, description string) *job104.JobDetailResponse {
	d := &job104.JobDetailResponse{}
	d.Data.Header.JobName = title
	d.Data.JobDetail.JobDescription = description
	return d
}
