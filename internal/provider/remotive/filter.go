package remotive

import "strings"

// FilterOptions narrows a jobs dump client-side. Remotive's documented
// server-side query params are no-ops (see the deviation notes in
// openapi.yaml), so every consumer — CLI and MCP tools alike — filters
// the full dump with this instead. Zero-valued fields don't filter;
// set fields must all match (AND semantics).
type FilterOptions struct {
	// Keyword is a case-insensitive substring matched against a job's
	// title, tags, and HTML description.
	Keyword string
	// Category matches the job's category display name
	// case-insensitively as a substring, with hyphens treated as spaces
	// so both the display name ("Software Development") and the slug
	// from /remote-jobs/categories ("software-development") work.
	Category string
	// Company is a case-insensitive substring of company_name —
	// the same partial-match semantics the official README documents
	// for the (dead) company_name query param.
	Company string
	// JobType is an exact job_type value, e.g. "full_time" or "contract".
	JobType string
	// Location is a case-insensitive substring of
	// candidate_required_location, e.g. "usa" or "europe".
	Location string
}

// FilterJobs returns the jobs matching every set field of opts, in their
// original order. The input slice is never modified.
func FilterJobs(jobs []Job, opts FilterOptions) []Job {
	keyword := strings.ToLower(opts.Keyword)
	category := strings.ToLower(strings.ReplaceAll(opts.Category, "-", " "))
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
		if company != "" && !strings.Contains(strings.ToLower(j.CompanyName), company) {
			continue
		}
		if opts.JobType != "" && j.JobType != opts.JobType {
			continue
		}
		if location != "" && !strings.Contains(strings.ToLower(j.CandidateRequiredLocation), location) {
			continue
		}
		out = append(out, j)
	}
	return out
}
