package googlejobs

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"
)

const (
	// DefaultResultsWanted matches JobSpy's scrape_jobs default.
	DefaultResultsWanted = 15
	// MaxResultsWanted matches JobSpy's Google-specific cap.
	MaxResultsWanted = 900
	// MaxOffset bounds cursor work for one request while retaining deep paging.
	MaxOffset = 900

	defaultAsyncParam = "_basejs:/xjs/_/js/k=xjs.s.en_US.JwveA-JiKmg.2018.O/am=AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAIAAAAAAAAACAAAoICAAAAAAAKMAfAAAAIAQAAAAAAAAAAAAACCAAAEJDAAACAAAAAGABAIAAARBAAABAAAAAgAgQAABAASKAfv8JAAABAAAAAAwAQAQACQAAAAAAcAEAQABoCAAAABAAAIABAACAAAAEAAAAFAAAAAAAAAAAAAAAAAAAAAAAAACAQADoBwAAAAAAAAAAAAAQBAAAAATQAAoACOAHAAAAAAAAAQAAAIIAAAA_ZAACAAAAAAAAcB8APB4wHFJ4AAAAAAAAAAAAAAAACECCYA5If0EACAAAAAAAAAAAAAAAAAAAUgRNXG4AMAE/dg=0/br=1/rs=ACT90oGxMeaFMCopIHq5tuQM-6_3M_VMjQ,_basecss:/xjs/_/ss/k=xjs.s.IwsGu62EDtU.L.B1.O/am=QOoQIAQAAAQAREADEBAAAAAAAAAAAAAAAAAAAAAgAQAAIAAAgAQAAAIAIAIAoEwCAADIC8AfsgEAawwAPkAAjgoAGAAAAAAAAEADAAAAAAIgAECHAAAAAAAAAAABAQAggAARQAAAQCEAAAAAIAAAABgAAAAAIAQIACCAAfB-AAFIQABoCEA_CgEAAIABAACEgHAEwwAEFQAM4CgAAAAAAAAAAAAACABCAAAAQEAAABAgAMCPAAA4AoE2BAEAggSAAIoAQAAAAAgAAAAACCAQAAAxEwA_ZAACAAAAAAAAAAkAAAAAAAAgAAAAAAAAAAAAAAAAAAAAAAAAQAEAAAAAAAAAAAAAAAAAAAAAQA/br=1/rs=ACT90oGZc36t3uUQkj0srnIvvbHjO2hgyg,_basecomb:/xjs/_/js/k=xjs.s.en_US.JwveA-JiKmg.2018.O/ck=xjs.s.IwsGu62EDtU.L.B1.O/am=QOoQIAQAAAQAREADEBAAAAAAAAAAAAAAAAAAAAAgAQAAIAAAgAQAAAKAIAoIqEwCAADIK8AfsgEAawwAPkAAjgoAGAAACCAAAEJDAAACAAIgAGCHAIAAARBAAABBAQAggAgRQABAQSOAfv8JIAABABgAAAwAYAQICSCAAfB-cAFIQABoCEA_ChEAAIABAACEgHAEwwAEFQAM4CgAAAAAAAAAAAAACABCAACAQEDoBxAgAMCPAAA4AoE2BAEAggTQAIoASOAHAAgAAAAACSAQAIIxEwA_ZAACAAAAAAAAcB8APB4wHFJ4AAAAAAAAAAAAAAAACECCYA5If0EACAAAAAAAAAAAAAAAAAAAUgRNXG4AMAE/d=1/ed=1/dg=0/br=1/ujg=1/rs=ACT90oFNLTjPzD_OAqhhtXwe2pg1T3WpBg,_fmt:prog,_id:fc_5FwaZ86OKsfdwN4P4La3yA4_2"
)

// JobType is a JobSpy-compatible query phrase selector.
type JobType string

const (
	JobTypeFullTime   JobType = "fulltime"
	JobTypePartTime   JobType = "parttime"
	JobTypeInternship JobType = "internship"
	JobTypeContract   JobType = "contract"
)

// SearchRequest describes one JobSpy-compatible Google Jobs scrape.
type SearchRequest struct {
	SearchTerm       string  `json:"search_term,omitempty"`
	GoogleSearchTerm string  `json:"google_search_term,omitempty"`
	Location         string  `json:"location,omitempty"`
	JobType          JobType `json:"job_type,omitempty"`
	Remote           bool    `json:"remote,omitempty"`
	ResultsWanted    int     `json:"results_wanted,omitempty"`
	Offset           int     `json:"offset,omitempty"`
	HoursOld         int     `json:"hours_old,omitempty"`
}

// Job is the normalized subset extracted by JobSpy's Google scraper.
type Job struct {
	ID          string   `json:"id"`
	Title       string   `json:"title"`
	Company     string   `json:"company,omitempty"`
	Location    string   `json:"location,omitempty"`
	City        string   `json:"city,omitempty"`
	State       string   `json:"state,omitempty"`
	Country     string   `json:"country,omitempty"`
	URL         string   `json:"url,omitempty"`
	DatePosted  string   `json:"date_posted,omitempty"`
	Remote      bool     `json:"remote,omitempty"`
	Description string   `json:"description,omitempty"`
	Emails      []string `json:"emails,omitempty"`
	JobTypes    []string `json:"job_types,omitempty"`
}

// SearchResponse contains unique jobs and a non-fatal pagination warning.
type SearchResponse struct {
	Jobs    []Job  `json:"jobs"`
	Warning string `json:"warning,omitempty"`
}

// ScraperOption customizes the JobSpy-derived scraper.
type ScraperOption func(*Scraper) error

// WithAsyncParam replaces the volatile Google frontend bootstrap descriptor.
// Use this when the value copied from the JobSpy upstream stops working.
func WithAsyncParam(value string) ScraperOption {
	return func(s *Scraper) error {
		if value == "" {
			return errors.New("googlejobs: async parameter is empty")
		}
		s.asyncParam = value
		return nil
	}
}

// Scraper adds JobSpy's query construction, cursor loop, and parser around
// the generated transport client.
type Scraper struct {
	client     Invoker
	asyncParam string
	now        func() time.Time
}

// NewScraper builds a JobSpy-compatible Google Jobs scraper.
func NewScraper(client Invoker, opts ...ScraperOption) (*Scraper, error) {
	if client == nil {
		return nil, errors.New("googlejobs: nil generated client")
	}
	s := &Scraper{
		client:     client,
		asyncParam: defaultAsyncParam,
		now:        time.Now,
	}
	for _, opt := range opts {
		if opt == nil {
			continue
		}
		if err := opt(s); err != nil {
			return nil, fmt.Errorf("googlejobs: apply scraper option: %w", err)
		}
	}
	return s, nil
}

// Search executes the initial HTML search and follows async cursors until the
// requested unique-URL window is available or Google stops yielding results.
func (s *Scraper) Search(ctx context.Context, req SearchRequest) (*SearchResponse, error) {
	if s == nil || s.client == nil {
		return nil, errors.New("googlejobs: uninitialized scraper")
	}
	query, err := buildQuery(req)
	if err != nil {
		return nil, err
	}
	resultsWanted, err := normalizeResultWindow(req.ResultsWanted, req.Offset)
	if err != nil {
		return nil, err
	}

	initial, err := s.client.SearchJobsInitial(ctx, SearchJobsInitialParams{
		Q:              query,
		Udm:            SearchJobsInitialUdm8,
		AcceptLanguage: NewOptString("en-US,en;q=0.9"),
		UserAgent:      NewOptString(defaultUserAgent),
	})
	if err != nil {
		return nil, fmt.Errorf("googlejobs: initial search: %w", err)
	}
	body, err := initialResponseBody(initial)
	if err != nil {
		return nil, err
	}
	initialJobs, cursor, err := parseInitialPage(body, s.now())
	if err != nil {
		return nil, err
	}

	jobs := make([]Job, 0, min(resultsWanted+req.Offset, MaxResultsWanted+MaxOffset))
	seenURLs := make(map[string]struct{}, cap(jobs))
	appendUniqueJobs(&jobs, seenURLs, initialJobs)

	response := &SearchResponse{Jobs: []Job{}}
	target := resultsWanted + req.Offset
	if cursor == "" {
		response.Warning = "initial cursor not found; the query may have at most 10 results or Google returned an unexpected page"
	}

	for len(seenURLs) < target && cursor != "" {
		if err := ctx.Err(); err != nil {
			return nil, fmt.Errorf("googlejobs: paginate: %w", err)
		}
		next, err := s.client.SearchJobsNextPage(ctx, SearchJobsNextPageParams{
			Fc:             cursor,
			Fcv:            SearchJobsNextPageFcv3,
			Async:          s.asyncParam,
			AcceptLanguage: NewOptString("en-US,en;q=0.9"),
			UserAgent:      NewOptString(defaultUserAgent),
		})
		if err != nil {
			response.Warning = fmt.Sprintf("pagination stopped after %d unique jobs: %v", len(seenURLs), err)
			break
		}
		body, err := nextResponseBody(next)
		if err != nil {
			response.Warning = fmt.Sprintf("pagination stopped after %d unique jobs: %v", len(seenURLs), err)
			break
		}
		nextJobs, nextCursor, err := parseNextPage(body, s.now())
		if err != nil {
			response.Warning = fmt.Sprintf("pagination stopped after %d unique jobs: %v", len(seenURLs), err)
			break
		}
		if len(nextJobs) == 0 {
			break
		}
		before := len(seenURLs)
		appendUniqueJobs(&jobs, seenURLs, nextJobs)
		if len(seenURLs) == before {
			response.Warning = fmt.Sprintf("pagination stopped after %d unique jobs: next page contained only duplicate URLs", len(seenURLs))
			break
		}
		if nextCursor == cursor {
			response.Warning = fmt.Sprintf("pagination stopped after %d unique jobs: Google repeated the same cursor", len(seenURLs))
			break
		}
		cursor = nextCursor
	}

	start := min(req.Offset, len(jobs))
	end := min(start+resultsWanted, len(jobs))
	response.Jobs = append(response.Jobs, jobs[start:end]...)
	return response, nil
}

func buildQuery(req SearchRequest) (string, error) {
	if req.Offset < 0 || req.Offset > MaxOffset {
		return "", fmt.Errorf("googlejobs: offset must be between 0 and %d", MaxOffset)
	}
	if req.GoogleSearchTerm != "" {
		return req.GoogleSearchTerm, nil
	}
	if req.HoursOld < 0 {
		return "", errors.New("googlejobs: hours_old must not be negative")
	}
	if strings.TrimSpace(req.SearchTerm) == "" {
		return "", errors.New("googlejobs: search_term or google_search_term is required")
	}

	var query strings.Builder
	query.WriteString(req.SearchTerm)
	query.WriteString(" jobs")

	jobTypePhrase, err := jobTypeQueryPhrase(req.JobType)
	if err != nil {
		return "", err
	}
	if jobTypePhrase != "" {
		query.WriteByte(' ')
		query.WriteString(jobTypePhrase)
	}
	if req.Location != "" {
		query.WriteString(" near ")
		query.WriteString(req.Location)
	}
	if req.HoursOld > 0 {
		query.WriteByte(' ')
		query.WriteString(timeRangePhrase(req.HoursOld))
	}
	if req.Remote {
		query.WriteString(" remote")
	}
	return query.String(), nil
}

func normalizeResultWindow(resultsWanted, offset int) (int, error) {
	if resultsWanted < 0 {
		return 0, errors.New("googlejobs: results_wanted must not be negative")
	}
	if offset < 0 || offset > MaxOffset {
		return 0, fmt.Errorf("googlejobs: offset must be between 0 and %d", MaxOffset)
	}
	if resultsWanted == 0 {
		resultsWanted = DefaultResultsWanted
	}
	return min(resultsWanted, MaxResultsWanted), nil
}

func jobTypeQueryPhrase(jobType JobType) (string, error) {
	switch jobType {
	case "":
		return "", nil
	case JobTypeFullTime:
		return "Full time", nil
	case JobTypePartTime:
		return "Part time", nil
	case JobTypeInternship:
		return "Internship", nil
	case JobTypeContract:
		return "Contract", nil
	default:
		return "", fmt.Errorf("googlejobs: unsupported job_type %q", jobType)
	}
}

func timeRangePhrase(hoursOld int) string {
	switch {
	case hoursOld <= 24:
		return "since yesterday"
	case hoursOld <= 72:
		return "in the last 3 days"
	case hoursOld <= 168:
		return "in the last week"
	default:
		return "in the last month"
	}
}

func appendUniqueJobs(dst *[]Job, seenURLs map[string]struct{}, jobs []Job) {
	for _, job := range jobs {
		if _, ok := seenURLs[job.URL]; ok {
			continue
		}
		seenURLs[job.URL] = struct{}{}
		*dst = append(*dst, job)
	}
}

func initialResponseBody(response SearchJobsInitialRes) ([]byte, error) {
	switch response := response.(type) {
	case *SearchJobsInitialOK:
		return io.ReadAll(response)
	case *UnexpectedResponseTextHTMLStatusCode:
		return nil, unexpectedStatusError("initial search", response.StatusCode, response.Response.Data)
	case *UnexpectedResponseTextPlainStatusCode:
		return nil, unexpectedStatusError("initial search", response.StatusCode, response.Response.Data)
	default:
		return nil, fmt.Errorf("googlejobs: initial search returned unexpected response type %T", response)
	}
}

func nextResponseBody(response SearchJobsNextPageRes) ([]byte, error) {
	switch response := response.(type) {
	case *SearchJobsNextPageOKTextHTML:
		return io.ReadAll(response)
	case *SearchJobsNextPageOKTextPlain:
		return io.ReadAll(response)
	case *UnexpectedResponseTextHTMLStatusCode:
		return nil, unexpectedStatusError("next page", response.StatusCode, response.Response.Data)
	case *UnexpectedResponseTextPlainStatusCode:
		return nil, unexpectedStatusError("next page", response.StatusCode, response.Response.Data)
	default:
		return nil, fmt.Errorf("googlejobs: next page returned unexpected response type %T", response)
	}
}

func unexpectedStatusError(operation string, status int, body io.Reader) error {
	sample, err := io.ReadAll(io.LimitReader(body, 512))
	if err != nil {
		return fmt.Errorf("googlejobs: %s returned HTTP %d and response sample could not be read: %w", operation, status, err)
	}
	return fmt.Errorf("googlejobs: %s returned HTTP %d: %q", operation, status, strings.TrimSpace(string(sample)))
}
