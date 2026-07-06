package main

import (
	"fmt"
	"time"

	lever "github.com/amikai/openings-mcp/internal/provider/lever"
)

// leverAPIBaseURL is the global-instance base URL. Every curated site in
// companies.yaml lives on the global instance, so the CLI never needs the
// EU server (https://api.eu.lever.co).
const leverAPIBaseURL = "https://api.lever.co"

func main() {}

// postingJSON is the --format json shape for one posting, and the input
// to text rendering: a flat, stable projection of the generated
// lever.Posting so the CLI's output doesn't change shape when the spec's
// generated types do.
type postingJSON struct {
	ID          string   `json:"id"`
	Title       string   `json:"title"`
	URL         string   `json:"url,omitempty"`
	CreatedAt   string   `json:"createdAt,omitempty"` // 2006-01-02 (UTC)
	Location    string   `json:"location,omitempty"`
	Locations   []string `json:"locations,omitempty"`
	Team        string   `json:"team,omitempty"`
	Commitment  string   `json:"commitment,omitempty"`
	Description string   `json:"description,omitempty"`
}

func toPostingJSON(p lever.Posting) postingJSON {
	cats := p.Categories.Value
	r := postingJSON{
		ID:          p.ID,
		Title:       p.Text,
		URL:         p.HostedUrl.Value,
		Team:        cats.Team.Value,
		Commitment:  cats.Commitment.Value,
		Description: p.DescriptionPlain.Value,
	}
	if p.CreatedAt.Set {
		r.CreatedAt = time.UnixMilli(p.CreatedAt.Value).UTC().Format("2006-01-02")
	}
	setLocations(&r, postingLocations(p)...)
	return r
}

// postingLocations prefers the full allLocations list; the primary
// location is its first entry when present, so the fallback only matters
// for postings that carry a single location field.
func postingLocations(p lever.Posting) []string {
	cats := p.Categories.Value
	if len(cats.AllLocations) > 0 {
		return cats.AllLocations
	}
	if cats.Location.Set {
		return []string{cats.Location.Value}
	}
	return nil
}

// setLocations fills both the singular Location (first entry, for quick
// access) and the full Locations array (only when there's more than one,
// to avoid a redundant one-element array alongside the singular field) —
// mirrors cmd/workday's setLocations.
func setLocations(r *postingJSON, locations ...string) {
	if len(locations) == 0 {
		return
	}
	r.Location = locations[0]
	if len(locations) > 1 {
		r.Locations = locations
	}
}

// printPosting renders one posting as text. index > 0 numbers the entry
// (search results); index 0 prints it unnumbered (get).
func printPosting(index int, p postingJSON) {
	if index > 0 {
		fmt.Printf("%d. %s\n", index, p.Title)
	} else {
		fmt.Println(p.Title)
	}
	if p.CreatedAt != "" {
		fmt.Printf("Created: %s\n", p.CreatedAt)
	}
	if p.URL != "" {
		fmt.Printf("URL: %s\n", p.URL)
	}
	if len(p.Locations) > 0 {
		fmt.Println("Locations:")
		for _, l := range p.Locations {
			fmt.Printf("  - %s\n", l)
		}
	} else if p.Location != "" {
		fmt.Printf("Location: %s\n", p.Location)
	}
	if p.Team != "" {
		fmt.Printf("Team: %s\n", p.Team)
	}
	if p.Commitment != "" {
		fmt.Printf("Commitment: %s\n", p.Commitment)
	}
	if p.Description != "" {
		fmt.Printf("Description:\n%s\n", p.Description)
	}
}
