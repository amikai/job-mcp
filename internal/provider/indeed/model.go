package indeed

// Job type filter keys (the "attributes" composite filter's keyword keys),
// ported from python-jobspy's job_type_key_mapping.
const (
	JobTypeFullTime   = "CF3CP"
	JobTypePartTime   = "75GKK"
	JobTypeContract   = "NJXCK"
	JobTypeInternship = "VDTG7"
)

// remoteAttributeKey is the composite filter's keyword key for "Remote"
// (jobspy's is_remote branch), applied alongside JobType* when Remote is set.
const remoteAttributeKey = "DSQF7"

// JobTypeIDs maps a human label to its JobsRequest.JobType value.
var JobTypeIDs = map[string]string{
	"Full-time":  JobTypeFullTime,
	"Part-time":  JobTypePartTime,
	"Contract":   JobTypeContract,
	"Internship": JobTypeInternship,
}

// JobsRequest is a search query against Indeed's jobSearch GraphQL field.
type JobsRequest struct {
	Keywords string
	// Location is free-text, e.g. "Taipei". Must correspond to Country —
	// see openapi.yaml's Key Behaviors on what a mismatch does.
	Location string
	// RadiusMiles defaults to 25 (python-jobspy's default distance) when 0.
	RadiusMiles int
	// Country selects the indeed-co catalogue and the job_url domain via
	// CountryByName; empty defaults to DefaultCountryName.
	Country string
	// Cursor pages through results: pass the previous JobsResponse's
	// NextCursor. Empty starts from the first page.
	Cursor string
	// Limit caps results per call, max 100 (the reference implementation's
	// jobs_per_page); defaults to 25 when 0.
	Limit int
	// HoursOld, JobType/Remote, and EasyApply are mutually exclusive filters
	// in the reference query shape (see openapi.yaml); when more than one is
	// set, HoursOld wins, then EasyApply, then JobType/Remote — the same
	// precedence as python-jobspy's _build_filters.
	HoursOld  int
	JobType   string // one of JobType* above
	Remote    bool
	EasyApply bool
}

// Compensation mirrors Indeed's compensation.{baseSalary,estimated} shape;
// nil when the posting doesn't disclose one (the common case).
type Compensation struct {
	MinAmount int
	MaxAmount int
	Currency  string
	// Interval is the raw unitOfWork value (YEAR, MONTH, WEEK, DAY, HOUR),
	// uppercased; left as Indeed sends it rather than remapped, since
	// callers needing python-jobspy's YEARLY/MONTHLY/... labels can map it
	// themselves.
	Interval string
}

// Job is a jobSearch result: a lean summary, no full description.
type Job struct {
	Key        string // Indeed's opaque job key; pass to Client.JobDetail.
	Title      string
	Company    string
	CompanyURL string
	Location   string
	// JobURL is the Indeed-hosted posting page, built from the search
	// request's Country domain.
	JobURL string
	// PostedDate is YYYY-MM-DD, derived from datePublished (epoch millis).
	PostedDate string
	// JobTypes are Indeed's own attribute labels (e.g. "Full-time",
	// "Permanent"), passed through as-is rather than filtered to a fixed
	// enum.
	JobTypes     []string
	Compensation *Compensation
}

// JobsResponse is one page of jobSearch results.
type JobsResponse struct {
	Jobs []Job
	// NextCursor feeds JobsRequest.Cursor for the next page; empty means no
	// more results.
	NextCursor string
}

// JobDetail is a jobData result: the full field set, including description.
type JobDetail struct {
	Key          string
	Title        string
	Company      string
	CompanyURL   string
	Location     string
	JobURL       string
	PostedDate   string
	Description  string // HTML, as Indeed sends it.
	JobTypes     []string
	Compensation *Compensation

	CompanyWebsite     string
	CompanyIndustry    string
	CompanyEmployees   string
	CompanyRevenue     string
	CompanyDescription string
	CompanyLogo        string

	// ApplyURL is recruit.viewJobUrl: the external ATS URL a poster
	// configured for direct apply, if any. Empty for Indeed-native apply
	// flows.
	ApplyURL string
}
