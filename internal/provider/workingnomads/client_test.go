package workingnomads

import (
	"context"
	"testing"
)

func testClient(t *testing.T) *Client {
	t.Helper()
	srv := NewMockServer()
	t.Cleanup(srv.Close)
	return NewClient(srv.URL, srv.Client())
}

func TestClient_Jobs(t *testing.T) {
	c := testClient(t)
	jobs, err := c.Jobs(context.Background())
	if err != nil {
		t.Fatalf("Jobs: %v", err)
	}
	if len(jobs) != 34 {
		t.Fatalf("got %d jobs, want 34 (the captured fixture's item count)", len(jobs))
	}

	j := jobs[0]
	if j.ID != "1734670" {
		t.Errorf("ID = %q, want %q (parsed from the /job/go/<id>/ URL)", j.ID, "1734670")
	}
	if j.Title != "Senior AI Engineer" {
		t.Errorf("Title = %q, want %q", j.Title, "Senior AI Engineer")
	}
	if j.Company != "Lemon.io" {
		t.Errorf("Company = %q, want %q", j.Company, "Lemon.io")
	}
	if j.Category != "Development" {
		t.Errorf("Category = %q, want %q", j.Category, "Development")
	}
	wantTags := []string{"python", "machine learning", "architecture", "software engineering", "startup"}
	if len(j.Tags) != len(wantTags) {
		t.Fatalf("Tags = %v, want %v", j.Tags, wantTags)
	}
	for i, tag := range wantTags {
		if j.Tags[i] != tag {
			t.Errorf("Tags[%d] = %q, want %q", i, j.Tags[i], tag)
		}
	}
	if j.Location == "" {
		t.Error("Location is empty")
	}
	if j.Description == "" {
		t.Error("Description is empty")
	}
	if j.PostedAt.IsZero() {
		t.Error("PostedAt did not parse")
	}
	if j.URL != "https://www.workingnomads.com/job/go/1734670/" {
		t.Errorf("URL = %q, want the raw entry url", j.URL)
	}
}

func TestClient_Jobs_idsAreUnique(t *testing.T) {
	c := testClient(t)
	jobs, err := c.Jobs(context.Background())
	if err != nil {
		t.Fatalf("Jobs: %v", err)
	}
	seen := make(map[string]bool)
	for _, j := range jobs {
		if seen[j.ID] {
			t.Fatalf("duplicate job ID %q", j.ID)
		}
		seen[j.ID] = true
	}
}

func TestClient_Search(t *testing.T) {
	c := testClient(t)
	jobs, err := c.Search(context.Background(), FilterOptions{Category: "Development"})
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if len(jobs) == 0 {
		t.Fatal("expected at least one Development job")
	}
	for _, j := range jobs {
		if j.Category != "Development" {
			t.Errorf("job %q has Category %q, want Development", j.ID, j.Category)
		}
	}
}

func TestClient_Detail(t *testing.T) {
	c := testClient(t)
	jobs, err := c.Jobs(context.Background())
	if err != nil {
		t.Fatalf("Jobs: %v", err)
	}
	want := jobs[0]

	got, err := c.Detail(context.Background(), want.ID)
	if err != nil {
		t.Fatalf("Detail: %v", err)
	}
	if got.ID != want.ID || got.Title != want.Title || got.Description != want.Description {
		t.Fatalf("Detail returned a different job than Jobs: got %+v, want %+v", got.ID, want.ID)
	}
}

func TestClient_Detail_notFound(t *testing.T) {
	c := testClient(t)
	_, err := c.Detail(context.Background(), "no-such-job-id")
	if err == nil {
		t.Fatal("expected an error for an unknown job ID")
	}
}
