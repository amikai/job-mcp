// Package icims is a client for public iCIMS career-portal HTML pages,
// documented in openapi.yaml.
package icims

import (
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

const (
	searchPath = "/jobs/search"
	userAgent  = "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/124.0.0.0 Safari/537.36"
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
	// Page is the zero-based upstream pr index.
	Page int
}

// SearchResponse holds one upstream page of cards plus pagination metadata.
type SearchResponse struct {
	Jobs []Job
	// TotalPages comes from the "Page X of Y" label (at least 1).
	TotalPages int
	// PageSize is len(Jobs) on this response; used by the adapter to map
	// unified page offsets onto pr indices. Empty result pages leave this 0.
	PageSize int
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
func (c *Client) Search(ctx context.Context, req *SearchRequest) (*SearchResponse, error) {
	if req == nil {
		req = &SearchRequest{}
	}
	u, err := url.Parse(c.baseURL + searchPath)
	if err != nil {
		return nil, fmt.Errorf("parse base url %q: %w", c.baseURL, err)
	}
	q := u.Query()
	q.Set("ss", "1")
	q.Set("in_iframe", "1")
	if req.Page > 0 {
		q.Set("pr", strconv.Itoa(req.Page))
	} else {
		q.Set("pr", "0")
	}
	if kw := strings.TrimSpace(req.Keyword); kw != "" {
		q.Set("searchKeyword", kw)
	}
	if loc := strings.TrimSpace(req.Location); loc != "" {
		q.Set("searchLocation", loc)
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

	jobs, totalPages, pageSize, err := parseSearchHTML(doc)
	if err != nil {
		return nil, fmt.Errorf("search jobs: %w", err)
	}
	return &SearchResponse{Jobs: jobs, TotalPages: totalPages, PageSize: pageSize}, nil
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
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(string(body)))
	if err != nil {
		return nil, resp.StatusCode, fmt.Errorf("parse html: %w", err)
	}
	return doc, resp.StatusCode, nil
}
