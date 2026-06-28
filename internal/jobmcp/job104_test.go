package jobmcp

import (
	"net/http"
	"testing"

	job104 "github.com/amikai/job-mcp/internal/provider/job104"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func TestRegisterJob104(t *testing.T) {
	server := mcp.NewServer(&mcp.Implementation{Name: "test", Version: "v0"}, nil)

	RegisterJob104(server, job104.NewClient(job104.Config{HTTPClient: http.DefaultClient}))

	assertTools(t, server, "104_search_jobs", "104_get_job_detail")
}

func TestJob104ToRequest(t *testing.T) {
	in := job104SearchInput{
		Keyword: "golang",
		Area:    "taipei",
		JobType: "part",
		Sort:    "newest",
		Remote:  "full",
		Page:    2,
	}
	got, err := job104ToRequest(in)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Keyword != "golang" {
		t.Errorf("Keyword = %q, want golang", got.Keyword)
	}
	if got.Area != job104.AreaTaipei {
		t.Errorf("Area = %q, want %q", got.Area, job104.AreaTaipei)
	}
	if got.RO == nil || *got.RO != 1 {
		t.Errorf("RO = %v, want 1", got.RO)
	}
	if got.Order == nil || *got.Order != 15 {
		t.Errorf("Order = %v, want 15", got.Order)
	}
	if got.RemoteWork == nil || *got.RemoteWork != 2 {
		t.Errorf("RemoteWork = %v, want 2", got.RemoteWork)
	}
	if got.Page == nil || *got.Page != 2 {
		t.Errorf("Page = %v, want 2", got.Page)
	}
}

func TestJob104ToRequestInvalidArea(t *testing.T) {
	_, err := job104ToRequest(job104SearchInput{Keyword: "x", Area: "atlantis"})
	if err == nil {
		t.Fatal("expected error for invalid area, got nil")
	}
}
