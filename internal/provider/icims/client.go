// Package icims is a client for public iCIMS career-portal HTML pages,
// documented in openapi.yaml.
package icims

import (
	"bytes"
	"cmp"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/PuerkitoBio/goquery"
)

// ErrJobNotFound indicates the requested job ID returned HTTP 410 or a
// body without JobPosting JSON-LD (expired IDs fall back to the listing).
var ErrJobNotFound = errors.New("icims: job not found")

// ErrCompanyNotFound indicates the career-portal host does not exist
// (HTTP 404 on /jobs/search).
var ErrCompanyNotFound = errors.New("icims: company not found")

// ErrLocationTooBroad indicates free text matched more location options than
// can be searched safely in one request.
var ErrLocationTooBroad = errors.New("icims: location is too broad")

// ErrLocationRequestLimit indicates a multi-location search would exceed the
// bounded number of upstream requests.
var ErrLocationRequestLimit = errors.New("icims: location search request limit exceeded")

const (
	searchPath          = "/jobs/search"
	userAgent           = "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/124.0.0.0 Safari/537.36"
	maxLocationMatches  = 10
	maxUpstreamRequests = 50
)

// Client talks to one iCIMS career-portal origin.
type Client struct {
	httpClient *http.Client
	baseURL    string
}

// NewClient builds a client for baseURL (e.g. "https://careers-peraton.icims.com").
func NewClient(baseURL string, httpClient *http.Client) *Client {
	return &Client{
		httpClient: cmp.Or(httpClient, http.DefaultClient),
		baseURL:    strings.TrimRight(baseURL, "/"),
	}
}

// SearchRequest mirrors the /jobs/search query parameters.
type SearchRequest struct {
	Keyword  string
	Location string
	// Categories and PositionTypes are encoded option values from the
	// portal's searchCategory / searchPositionType selects, sent as repeated
	// query parameters (the server ORs values within one field). The server
	// silently ignores unknown values — returning unfiltered results — so
	// callers must resolve labels against SearchResponse options first.
	Categories    []string
	PositionTypes []string
	// Page is the zero-based upstream pr index.
	Page int
}

// SearchResponse holds one upstream page of cards plus pagination metadata.
type SearchResponse struct {
	Jobs []Job
	// TotalPages comes from the "Page X of Y" label (at least 1).
	TotalPages int
	// PageSize is len(Jobs) on this response. On a full upstream page this
	// equals the tenant's configured page size; on a partial last page it is
	// smaller. Callers that translate unified offsets must discover the
	// stable size from a known full page (typically pr=0 when TotalPages > 1),
	// not from an arbitrary response. Empty result pages leave this 0.
	PageSize int
	// Locations, Categories, and PositionTypes are the portal's search-form
	// <option> entries, for the selects the tenant renders.
	Locations     []SelectOption
	Categories    []SelectOption
	PositionTypes []SelectOption
}

// Job is one listing card.
type Job struct {
	ID       string
	Slug     string
	Title    string
	Location string
}

// JobDetailResponse is the JSON-LD-backed detail payload.
type JobDetailResponse struct {
	ID              string
	Title           string
	Location        string
	Employer        string
	PostedAtRaw     string
	EmploymentType  string
	Category        string
	DescriptionHTML string
	URL             string
}

// Search fetches one upstream results page.
//
// Location must be either an encoded portal option value (e.g.
// "12781-12827-Austin") or free text matched against option labels/values
// (e.g. "Austin"). Free text is resolved via a probe request before the
// real search. A single option match is sent as searchLocation; several
// option matches are searched separately and merged, up to a strict match
// limit. Unknown or excessively broad locations never fall back to an
// unfiltered result set.
func (c *Client) Search(ctx context.Context, req *SearchRequest) (*SearchResponse, error) {
	if req == nil {
		req = &SearchRequest{}
	}
	r := *req // shallow copy so resolution never mutates the caller's request
	r.Keyword = strings.TrimSpace(r.Keyword)
	r.Location = strings.TrimSpace(r.Location)
	r.Page = max(r.Page, 0)

	if r.Location != "" && !LooksLikeLocationValue(r.Location) {
		probeReq := r
		probeReq.Location = ""
		probeReq.Page = 0
		probe, err := c.doSearch(ctx, &probeReq)
		if err != nil {
			return nil, err
		}
		matches := normalizeLocationValues(MatchLocationOptions(probe.Locations, r.Location))
		if err := validateLocationCount(matches); err != nil {
			return nil, err
		}
		switch len(matches) {
		case 0:
			probe.Jobs = nil
			probe.TotalPages = 1
			probe.PageSize = 0
			return probe, nil
		case 1:
			r.Location = matches[0]
		default:
			return c.searchMerged(ctx, &r, matches, probe)
		}
	}

	return c.doSearch(ctx, &r)
}

// SearchAllForLocations fetches every page for a bounded set of encoded
// location values and returns a stable de-duplicated union. It uses the
// portal's own encoded filters as the source of truth because job-card text
// may omit country/state tokens present in the option labels. base's
// keyword and category / position-type values apply to every request; its
// Location and Page are ignored.
func (c *Client) SearchAllForLocations(ctx context.Context, base *SearchRequest, locations []string) ([]Job, []SelectOption, error) {
	if base == nil {
		base = &SearchRequest{}
	}
	locations = normalizeLocationValues(locations)
	if err := validateLocationCount(locations); err != nil {
		return nil, nil, err
	}

	var (
		out      []Job
		opts     []SelectOption
		seen     = make(map[string]struct{})
		requests int
	)
	for i, loc := range locations {
		for page := 0; ; page++ {
			if requests >= maxUpstreamRequests {
				return nil, nil, locationRequestLimitError(requests)
			}
			r := *base
			r.Location = loc
			r.Page = page
			res, err := c.doSearch(ctx, &r)
			if err != nil {
				return nil, nil, err
			}
			requests++
			if i == 0 && page == 0 {
				opts = res.Locations
			}
			for _, j := range res.Jobs {
				if _, dup := seen[j.ID]; dup {
					continue
				}
				seen[j.ID] = struct{}{}
				out = append(out, j)
			}
			if res.PageSize == 0 || page+1 >= res.TotalPages {
				break
			}

			// Stop as soon as the advertised remaining pages plus one probe
			// for each untouched location cannot fit in the request budget.
			budget := maxUpstreamRequests - requests
			untouchedLocations := len(locations) - (i + 1)
			remainingCurrent := res.TotalPages - (page + 1)
			if untouchedLocations > budget || remainingCurrent > budget-untouchedLocations {
				return nil, nil, locationRequestLimitError(requests)
			}
		}
	}
	return out, opts, nil
}

// searchMerged unions one upstream page across a bounded set of locations.
// probe supplies the select options for the merged response.
func (c *Client) searchMerged(ctx context.Context, base *SearchRequest, locations []string, probe *SearchResponse) (*SearchResponse, error) {
	locations = normalizeLocationValues(locations)
	if err := validateLocationCount(locations); err != nil {
		return nil, err
	}

	var (
		jobs     []Job
		seen     = make(map[string]struct{})
		maxPages int
	)
	for _, loc := range locations {
		r := *base
		r.Location = loc
		res, err := c.doSearch(ctx, &r)
		if err != nil {
			return nil, err
		}
		if res.TotalPages > maxPages {
			maxPages = res.TotalPages
		}
		for _, j := range res.Jobs {
			if _, dup := seen[j.ID]; dup {
				continue
			}
			seen[j.ID] = struct{}{}
			jobs = append(jobs, j)
		}
	}
	return &SearchResponse{
		Jobs:          jobs,
		TotalPages:    max(maxPages, 1),
		PageSize:      len(jobs),
		Locations:     probe.Locations,
		Categories:    probe.Categories,
		PositionTypes: probe.PositionTypes,
	}, nil
}

func normalizeLocationValues(locations []string) []string {
	out := make([]string, 0, len(locations))
	seen := make(map[string]struct{}, len(locations))
	for _, loc := range locations {
		loc = strings.TrimSpace(loc)
		if loc == "" {
			continue
		}
		if _, duplicate := seen[loc]; duplicate {
			continue
		}
		seen[loc] = struct{}{}
		out = append(out, loc)
	}
	return out
}

func validateLocationCount(locations []string) error {
	if len(locations) <= maxLocationMatches {
		return nil
	}
	return fmt.Errorf("%w: matched %d options (maximum %d); provide a more specific city or state", ErrLocationTooBroad, len(locations), maxLocationMatches)
}

func locationRequestLimitError(requests int) error {
	return fmt.Errorf("%w after %d requests (maximum %d); provide a more specific location or keyword", ErrLocationRequestLimit, requests, maxUpstreamRequests)
}

// doSearch issues one upstream request. req's location and category /
// position-type values must already be encoded option values — free text is
// resolved in Search.
func (c *Client) doSearch(ctx context.Context, req *SearchRequest) (*SearchResponse, error) {
	u, err := url.Parse(c.baseURL + searchPath)
	if err != nil {
		return nil, fmt.Errorf("parse base url %q: %w", c.baseURL, err)
	}
	q := u.Query()
	q.Set("ss", "1")
	q.Set("in_iframe", "1")
	q.Set("pr", strconv.Itoa(max(req.Page, 0)))
	if req.Keyword != "" {
		q.Set("searchKeyword", req.Keyword)
	}
	if req.Location != "" {
		q.Set("searchLocation", req.Location)
	}
	for _, v := range req.Categories {
		q.Add("searchCategory", v)
	}
	for _, v := range req.PositionTypes {
		q.Add("searchPositionType", v)
	}
	u.RawQuery = q.Encode()

	doc, status, err := c.getHTML(ctx, u.String())
	if err != nil {
		return nil, fmt.Errorf("search jobs: %w", err)
	}
	if status == http.StatusNotFound {
		return nil, fmt.Errorf("search jobs: %w", ErrCompanyNotFound)
	}
	if status != http.StatusOK {
		return nil, fmt.Errorf("search jobs: HTTP %d", status)
	}

	res, err := parseSearchHTML(doc)
	if err != nil {
		return nil, fmt.Errorf("search jobs: %w", err)
	}
	return res, nil
}

// JobDetail fetches one posting. The path slug is cosmetic (see openapi.yaml);
// this always uses "job" as the slug segment.
func (c *Client) JobDetail(ctx context.Context, id string) (*JobDetailResponse, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return nil, fmt.Errorf("icims: empty job id")
	}
	if _, err := strconv.Atoi(id); err != nil {
		return nil, fmt.Errorf("icims: job id %q must be numeric", id)
	}

	u, err := url.Parse(c.baseURL)
	if err != nil {
		return nil, fmt.Errorf("parse base url %q: %w", c.baseURL, err)
	}
	u = u.JoinPath("jobs", id, "job", "job")
	q := u.Query()
	q.Set("in_iframe", "1")
	u.RawQuery = q.Encode()

	doc, status, err := c.getHTML(ctx, u.String())
	if err != nil {
		return nil, fmt.Errorf("job detail %q: %w", id, err)
	}
	if status == http.StatusGone || status == http.StatusNotFound {
		return nil, fmt.Errorf("job detail %q: %w", id, ErrJobNotFound)
	}
	if status != http.StatusOK {
		return nil, fmt.Errorf("job detail %q: HTTP %d", id, status)
	}

	detail, ok := parseJobDetailHTML(doc, id)
	if !ok {
		if isSearchLikeDetailBody(doc) {
			return nil, fmt.Errorf("job detail %q: %w", id, ErrJobNotFound)
		}
		return nil, fmt.Errorf("job detail %q: unrecognized detail page", id)
	}
	return detail, nil
}

// JobURL builds the public (non-iframe) posting URL for id.
func JobURL(host, id string) string {
	return fmt.Sprintf("https://%s/jobs/%s/job/job", host, id)
}

func (c *Client) getHTML(ctx context.Context, rawURL string) (*goquery.Document, int, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, 0, err
	}
	req.Header.Set("User-Agent", userAgent)
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")
	req.Header.Set("Accept-Language", "en-US,en;q=0.9")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close()

	// Read body for both success and the statuses we parse (404/410).
	body, err := io.ReadAll(io.LimitReader(resp.Body, 8<<20))
	if err != nil {
		return nil, resp.StatusCode, fmt.Errorf("read body: %w", err)
	}
	doc, err := goquery.NewDocumentFromReader(bytes.NewReader(body))
	if err != nil {
		return nil, resp.StatusCode, fmt.Errorf("parse html: %w", err)
	}
	return doc, resp.StatusCode, nil
}
