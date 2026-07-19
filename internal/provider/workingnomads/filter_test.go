package workingnomads

import "testing"

func TestFilterJobs(t *testing.T) {
	jobs := []Job{
		{
			ID: "a", Title: "Senior Go Engineer", Company: "Acme Corp",
			Category: "Development", Tags: []string{"go", "kubernetes"},
			Location: "Europe, North America", Description: "Build backend services in Go.",
		},
		{
			ID: "b", Title: "Product Designer", Company: "Widget Inc",
			Category: "Design", Tags: []string{"figma"},
			Location: "Global", Description: "Design product experiences.",
		},
		{
			ID: "c", Title: "Support Specialist", Company: "Acme Corp",
			Category: "Customer Success", Tags: nil,
			Location: "Texas, Oklahoma", Description: "Help customers with billing questions.",
		},
	}

	tests := []struct {
		name string
		opts FilterOptions
		want []string
	}{
		{"no filter", FilterOptions{}, []string{"a", "b", "c"}},
		{"keyword matches title", FilterOptions{Keyword: "go engineer"}, []string{"a"}},
		{"keyword matches description", FilterOptions{Keyword: "billing"}, []string{"c"}},
		{"keyword matches tags", FilterOptions{Keyword: "figma"}, []string{"b"}},
		{"category substring case-insensitive", FilterOptions{Category: "develop"}, []string{"a"}},
		{"category no match", FilterOptions{Category: "Marketing"}, nil},
		{"company substring", FilterOptions{Company: "acme"}, []string{"a", "c"}},
		{"location substring", FilterOptions{Location: "texas"}, []string{"c"}},
		{"combined filters AND together", FilterOptions{Company: "acme", Category: "customer"}, []string{"c"}},
		{"combined filters no match", FilterOptions{Company: "acme", Category: "design"}, nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FilterJobs(jobs, tt.opts)
			if len(got) != len(tt.want) {
				t.Fatalf("got %d jobs, want %d (%v)", len(got), len(tt.want), tt.want)
			}
			for i, j := range got {
				if j.ID != tt.want[i] {
					t.Errorf("job[%d].ID = %q, want %q", i, j.ID, tt.want[i])
				}
			}
		})
	}
}

func TestFilterJobs_doesNotMutateInput(t *testing.T) {
	jobs := []Job{{ID: "a", Title: "X"}, {ID: "b", Title: "Y"}}
	_ = FilterJobs(jobs, FilterOptions{Keyword: "x"})
	if len(jobs) != 2 || jobs[0].ID != "a" || jobs[1].ID != "b" {
		t.Fatalf("input slice was mutated: %+v", jobs)
	}
}
