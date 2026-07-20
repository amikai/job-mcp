package quanta

import "strings"

// FilterOptions narrows a jobs dump client-side. The QueryJob endpoint's
// query string is a no-op (see the deviation notes in openapi.yaml), so
// every consumer — CLI and MCP tools alike — filters the full dump with
// this instead, mirroring the site's own JS. Zero-valued fields don't
// filter; set fields must all match (AND semantics).
type FilterOptions struct {
	// Keyword is a case-insensitive substring matched against the
	// concatenation of a job's category name, description, title,
	// location name, requirements, category id, location id, and
	// keywords — the same fields the site's own filter concatenates.
	Keyword string
	// LocationIDs, if non-empty, restricts results to jobs whose
	// location id (GetLocationID) is one of these values.
	LocationIDs []string
	// CategoryIDs, if non-empty, restricts results to jobs whose
	// category id (GetCategoryID) is one of these values.
	CategoryIDs []string
}

// FilterJobs returns the jobs matching every set field of opts, in their
// original order. The input slice is never modified.
func FilterJobs(jobs []Job, opts FilterOptions) []Job {
	keyword := strings.ToLower(opts.Keyword)
	locationIDs := toSet(opts.LocationIDs)
	categoryIDs := toSet(opts.CategoryIDs)

	var out []Job
	for _, j := range jobs {
		if keyword != "" && !strings.Contains(searchHaystack(j), keyword) {
			continue
		}
		if len(locationIDs) > 0 && !locationIDs[j.GetLocationID()] {
			continue
		}
		if len(categoryIDs) > 0 && !categoryIDs[j.GetCategoryID()] {
			continue
		}
		out = append(out, j)
	}
	return out
}

// searchHaystack builds the lowercased, nil-safe concatenation of fields
// the site's own client-side filter matches a keyword against:
// caponm+descnm+jobdnm+locatm+requnm+capoid+locati+keywrd.
func searchHaystack(j Job) string {
	fields := []string{
		j.GetCategoryName(),
		j.GetDescription(),
		j.GetTitle(),
		j.GetLocationName(),
		j.GetRequirements().Or(""),
		j.GetCategoryID(),
		j.GetLocationID(),
		j.GetKeywords(),
	}
	return strings.ToLower(strings.Join(fields, ""))
}

// toSet builds a membership set from a slice, treating a nil or empty
// slice as "no restriction" (callers check len before consulting it).
func toSet(ids []string) map[string]bool {
	if len(ids) == 0 {
		return nil
	}
	set := make(map[string]bool, len(ids))
	for _, id := range ids {
		set[id] = true
	}
	return set
}

// FindBySerial returns the job whose serial matches, and whether one was
// found — there is no detail endpoint, so detail lookup is always a scan
// of the current dump.
func FindBySerial(jobs []Job, serial string) (Job, bool) {
	for _, j := range jobs {
		if j.GetSerial() == serial {
			return j, true
		}
	}
	return Job{}, false
}
