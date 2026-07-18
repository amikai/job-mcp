package workingnomads

import "strings"

// FilterOptions narrows a jobs dump client-side. Working Nomads has no
// server-side search — see doc.go — so this is the only way to narrow
// results. Zero-valued fields don't filter; set fields must all match
// (AND semantics).
type FilterOptions struct {
	// Keyword is a case-insensitive substring matched against a job's
	// title, tags, and HTML description.
	Keyword string
	// Category is a case-insensitive substring of the job's category
	// (e.g. "Development", "Design"); there is no fixed enum to match
	// exactly against, see doc.go.
	Category string
	// Company is a case-insensitive substring of the company name.
	Company string
	// Location is a case-insensitive substring of the free-text location
	// field, e.g. "europe" or "remote".
	Location string
}

// FilterJobs returns the jobs matching every set field of opts, in their
// original order. The input slice is never modified.
func FilterJobs(jobs []Job, opts FilterOptions) []Job {
	keyword := strings.ToLower(opts.Keyword)
	category := strings.ToLower(opts.Category)
	company := strings.ToLower(opts.Company)
	location := strings.ToLower(opts.Location)

	var out []Job
	for _, j := range jobs {
		if keyword != "" {
			haystack := strings.ToLower(j.Title + " " + strings.Join(j.Tags, " ") + " " + j.Description)
			if !strings.Contains(haystack, keyword) {
				continue
			}
		}
		if category != "" && !strings.Contains(strings.ToLower(j.Category), category) {
			continue
		}
		if company != "" && !strings.Contains(strings.ToLower(j.Company), company) {
			continue
		}
		if location != "" && !strings.Contains(strings.ToLower(j.Location), location) {
			continue
		}
		out = append(out, j)
	}
	return out
}
