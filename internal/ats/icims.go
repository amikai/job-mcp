package ats

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/jaytaylor/html2text"

	"github.com/amikai/openings-mcp/internal/provider/icims"
)

var _ Adapter = (*ICIMSAdapter)(nil)

// ICIMSAdapter serves public iCIMS career portals. Search and detail are
// server-rendered HTML (see internal/provider/icims/openapi.yaml). Roster
// slugs are lowercase portal hostnames (e.g. careers-peraton.icims.com).
// ParseCareersURL also accepts any *.icims.com host so uncurated tenants
// work when passed as a careers URL.
type ICIMSAdapter struct {
	hc      *http.Client
	baseURL func(host string) string
}

func NewICIMSAdapter(hc *http.Client) *ICIMSAdapter {
	return &ICIMSAdapter{
		hc:      hc,
		baseURL: func(host string) string { return "https://" + host },
	}
}

func (a *ICIMSAdapter) Name() string { return "icims" }

func (a *ICIMSAdapter) Roster() []CompanyInfo {
	infos := make([]CompanyInfo, 0, len(icims.Companies))
	for _, c := range icims.Companies {
		infos = append(infos, CompanyInfo{Slug: c.Host, Name: c.Name})
	}
	return infos
}

// ParseCareersURL recognizes any *.icims.com host. Curated roster names are
// preferred via the registry's name index; uncurated hosts still resolve so
// callers can pass a careers URL directly.
func (a *ICIMSAdapter) ParseCareersURL(u *url.URL) (string, bool) {
	host := strings.ToLower(u.Hostname())
	if !strings.HasSuffix(host, ".icims.com") || host == "icims.com" {
		return "", false
	}
	// login / CDN hosts are not career portals.
	if strings.HasPrefix(host, "login") || strings.HasPrefix(host, "cdn") ||
		strings.HasPrefix(host, "api.") || host == "www.icims.com" {
		return "", false
	}
	return host, true
}

func (a *ICIMSAdapter) Search(ctx context.Context, slug string, p SearchParams) (*SearchResult, error) {
	host, companyName, err := a.resolveHost(slug)
	if err != nil {
		return nil, err
	}

	page := clampPage(p.Page)
	start := (page - 1) * pageSize

	client := icims.NewClient(a.baseURL(host), a.hc)
	baseReq := icims.SearchRequest{
		Keyword:  p.Query,
		Location: strings.TrimSpace(p.Location),
	}

	// Upstream page size is tenant-configured (observed 20 and 50). Guess
	// pr using the common size of 20, then correct after we see PageSize.
	guessPr := 0
	if start > 0 {
		guessPr = start / 20
	}
	res, err := client.Search(ctx, &icims.SearchRequest{
		Keyword:  baseReq.Keyword,
		Location: baseReq.Location,
		Page:     guessPr,
	})
	if err != nil {
		return nil, fmt.Errorf("icims: search %q: %w", host, err)
	}

	upSize := res.PageSize
	totalPagesUp := res.TotalPages
	if upSize == 0 {
		total := 0
		if totalPagesUp <= 1 {
			total = 0
		}
		return &SearchResult{Jobs: nil, TotalCount: total, Page: page, TotalPages: totalPages(total)}, nil
	}

	correctPr := start / upSize
	if correctPr != guessPr {
		res, err = client.Search(ctx, &icims.SearchRequest{
			Keyword:  baseReq.Keyword,
			Location: baseReq.Location,
			Page:     correctPr,
		})
		if err != nil {
			return nil, fmt.Errorf("icims: search %q page %d: %w", host, correctPr, err)
		}
		if res.PageSize > 0 {
			upSize = res.PageSize
		}
		totalPagesUp = res.TotalPages
	}

	offsetInPage := start % upSize
	var selected []icims.Job
	if offsetInPage < len(res.Jobs) {
		selected = append(selected, res.Jobs[offsetInPage:]...)
	}

	// Stitch one more upstream page when the slice starts mid-page and
	// needs more rows to fill the unified pageSize.
	if len(selected) < pageSize && correctPr+1 < totalPagesUp {
		more, err := client.Search(ctx, &icims.SearchRequest{
			Keyword:  baseReq.Keyword,
			Location: baseReq.Location,
			Page:     correctPr + 1,
		})
		if err != nil {
			return nil, fmt.Errorf("icims: search %q page %d: %w", host, correctPr+1, err)
		}
		selected = append(selected, more.Jobs...)
	}

	if len(selected) > pageSize {
		selected = selected[:pageSize]
	}

	total := totalPagesUp * upSize
	if totalPagesUp <= 1 {
		// Single upstream page: exact count is the card list length.
		// When we sliced from pr=0, that is len(res.Jobs); when start
		// pointed past the end, total is still the full board size.
		if correctPr == 0 {
			total = len(res.Jobs)
		} else {
			total = 0
		}
	}

	return &SearchResult{
		Jobs:       icimsJobSummaries(selected, host, companyName),
		TotalCount: total,
		Page:       page,
		TotalPages: totalPages(total),
	}, nil
}

func (a *ICIMSAdapter) Filters(ctx context.Context, slug string) (FilterSet, error) {
	if _, _, err := a.resolveHost(slug); err != nil {
		return nil, err
	}
	// iCIMS portal filters (category, position type, …) are HTML form
	// selects without a stable cross-tenant facet JSON API. Structured
	// Filters are not exposed; callers use keyword + location.
	return FilterSet{}, nil
}

func (a *ICIMSAdapter) Detail(ctx context.Context, slug, jobID string) (*JobDetail, error) {
	host, companyName, err := a.resolveHost(slug)
	if err != nil {
		return nil, err
	}
	client := icims.NewClient(a.baseURL(host), a.hc)
	d, err := client.JobDetail(ctx, jobID)
	if errors.Is(err, icims.ErrJobNotFound) {
		return nil, fmt.Errorf(
			"icims: job %q not found for company %q; pass a job_id exactly as returned by the job search",
			jobID, slug,
		)
	}
	if err != nil {
		return nil, fmt.Errorf("icims: fetch job %q for %q: %w", jobID, slug, err)
	}

	desc := d.DescriptionHTML
	if desc != "" {
		if text, err := html2text.FromString(desc, html2text.Options{}); err == nil {
			desc = text
		}
	}

	company := companyName
	if company == "" {
		company = d.Employer
	}

	return &JobDetail{
		JobID:       jobID,
		Title:       d.Title,
		Company:     company,
		Location:    d.Location,
		PostedAt:    icimsPostedAt(d.PostedAtRaw),
		URL:         icims.JobURL(host, jobID),
		Description: desc,
	}, nil
}

// resolveHost returns the portal host and a display name (empty when the
// host is not on the curated roster).
func (a *ICIMSAdapter) resolveHost(slug string) (host, name string, err error) {
	key := strings.ToLower(strings.TrimSpace(slug))
	if c, ok := icims.CompaniesByHost[key]; ok {
		return c.Host, c.Name, nil
	}
	// Careers-URL path: accept any *.icims.com host even if uncurated.
	if strings.HasSuffix(key, ".icims.com") && key != "icims.com" {
		return key, "", nil
	}
	return "", "", fmt.Errorf("icims: unknown company %q; pass a roster career-portal host or a *.icims.com careers URL", slug)
}

func icimsJobSummaries(jobs []icims.Job, host, companyName string) []JobSummary {
	_ = companyName
	out := make([]JobSummary, 0, len(jobs))
	for _, j := range jobs {
		out = append(out, JobSummary{
			JobID:    j.ID,
			Title:    j.Title,
			Location: j.Location,
			URL:      icims.JobURL(host, j.ID),
		})
	}
	return out
}

func icimsPostedAt(raw string) string {
	if raw == "" {
		return ""
	}
	// JSON-LD datePosted is typically ISO-8601 with Z.
	for _, layout := range []string{
		time.RFC3339Nano,
		time.RFC3339,
		"2006-01-02T15:04:05.000Z",
		"2006-01-02",
	} {
		if t, err := time.Parse(layout, raw); err == nil {
			return isoDate(t)
		}
	}
	return raw
}
