package ats

import (
	"cmp"
	"context"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/jaytaylor/html2text"

	"github.com/amikai/openings-mcp/internal/provider/bamboohr"
)

var _ Adapter = (*BambooHRAdapter)(nil)

// bambooHRCareersHostRE matches BambooHR hosted careers-site hosts and
// captures the tenant slug (subdomain). Reserved product hosts are rejected
// after the match.
//
// Examples (hostname):
//   - concept2.bamboohr.com
//   - acme.bamboohr.com
var bambooHRCareersHostRE = regexp.MustCompile(
	`(?i)^(?P<slug>[^.]+)\.bamboohr\.com$`,
)

// BambooHRAdapter serves BambooHR hosted careers sites. The public
// /careers/list endpoint returns the complete board in one response but
// only as sparse summaries - descriptions, compensation, and posting dates
// live on the per-job /careers/{id}/detail endpoint - so Search and
// Filters run searchDump over the summary dump (query matching covers
// titles and departments only, and results carry no posted date), while
// Detail hits the per-job endpoint directly.
type BambooHRAdapter struct {
	hc      *http.Client
	baseURL func(slug string) string
}

// NewBambooHRAdapter derives a redirect-blocking copy of hc: BambooHR
// 302-redirects unknown tenants to its marketing site, and following that
// redirect would turn a diagnosable "no such tenant" into an HTML decode
// error.
func NewBambooHRAdapter(hc *http.Client) *BambooHRAdapter {
	c := *hc
	c.CheckRedirect = func(*http.Request, []*http.Request) error {
		return http.ErrUseLastResponse
	}
	return &BambooHRAdapter{
		hc: &c,
		baseURL: func(slug string) string {
			return "https://" + slug + ".bamboohr.com"
		},
	}
}

func (a *BambooHRAdapter) Name() string { return "bamboohr" }

func (a *BambooHRAdapter) Roster() []CompanyInfo {
	infos := make([]CompanyInfo, 0, len(bamboohr.Companies))
	for _, c := range bamboohr.Companies {
		infos = append(infos, CompanyInfo{Slug: c.Slug, Name: c.Name})
	}
	return infos
}

var bambooHRReservedHosts = map[string]bool{
	"api":           true,
	"app":           true,
	"careers":       true,
	"developers":    true,
	"documentation": true,
	"help":          true,
	"marketplace":   true,
	"partners":      true,
	"status":        true,
	"support":       true,
	"www":           true,
}

// ParseCareersURL recognizes BambooHR subdomain careers pages.
func (a *BambooHRAdapter) ParseCareersURL(u *url.URL) (string, bool) {
	m := bambooHRCareersHostRE.FindStringSubmatch(strings.ToLower(u.Hostname()))
	if m == nil {
		return "", false
	}
	slug := namedGroup(bambooHRCareersHostRE, m, "slug")
	if slug == "" || bambooHRReservedHosts[slug] {
		return "", false
	}
	return slug, true
}

func (a *BambooHRAdapter) Search(
	ctx context.Context,
	slug string,
	p SearchParams,
) (*SearchResult, error) {
	jobs, err := a.dump(ctx, slug)
	if err != nil {
		return nil, err
	}
	return searchDump(jobs, p)
}

func (a *BambooHRAdapter) Filters(ctx context.Context, slug string) (FilterSet, error) {
	jobs, err := a.dump(ctx, slug)
	if err != nil {
		return nil, err
	}
	return distinctFilters(jobs), nil
}

func (a *BambooHRAdapter) Detail(
	ctx context.Context,
	slug string,
	jobID string,
) (*JobDetail, error) {
	slug = strings.ToLower(slug)
	client, err := a.client(slug)
	if err != nil {
		return nil, err
	}
	res, err := client.GetJobDetail(ctx, bamboohr.GetJobDetailParams{ID: jobID})
	if err != nil {
		return nil, fmt.Errorf("bamboohr: fetch job %q for %q: %w", jobID, slug, err)
	}

	switch r := res.(type) {
	case *bamboohr.DetailResponse:
		jo := r.Result.JobOpening
		return &JobDetail{
			JobID:    jobID,
			Title:    jo.JobOpeningName,
			Company:  cmp.Or(bamboohr.CompaniesBySlug[slug].Name, slug),
			Location: bambooHRJoin(jo.Location.City.Or(""), jo.Location.State.Or(""), jo.Location.AddressCountry.Or("")),
			PostedAt: jo.DatePosted.Or(""),
			URL:      cmp.Or(jo.JobOpeningShareUrl, bambooHRPostingURL(slug, jobID)),
			Description: bambooHRDescription(jo.Description),
		}, nil
	case *bamboohr.NotFoundError:
		return nil, fmt.Errorf(
			"bamboohr: job %q not found for company %q; pass a job_id exactly as returned by the job search",
			jobID,
			slug,
		)
	case *bamboohr.GetJobDetailFound:
		return nil, fmt.Errorf("bamboohr: careers-site subdomain %q not found upstream", slug)
	default:
		return nil, fmt.Errorf("bamboohr: unexpected response type %T", res)
	}
}

func (a *BambooHRAdapter) client(slug string) (*bamboohr.Client, error) {
	client, err := bamboohr.NewClient(a.baseURL(slug), bamboohr.WithClient(a.hc))
	if err != nil {
		return nil, fmt.Errorf("bamboohr: create client for %q: %w", slug, err)
	}
	return client, nil
}

func (a *BambooHRAdapter) dump(ctx context.Context, slug string) ([]dumpJob, error) {
	slug = strings.ToLower(slug)
	client, err := a.client(slug)
	if err != nil {
		return nil, err
	}
	res, err := client.ListJobs(ctx)
	if err != nil {
		return nil, fmt.Errorf("bamboohr: fetch board for %q: %w", slug, err)
	}

	var rows []bamboohr.ListJob
	switch r := res.(type) {
	case *bamboohr.ListResponse:
		rows = r.Result
	case *bamboohr.ListJobsFound:
		return nil, fmt.Errorf("bamboohr: careers-site subdomain %q not found upstream", slug)
	default:
		return nil, fmt.Errorf("bamboohr: unexpected response type %T", res)
	}

	jobs := make([]dumpJob, 0, len(rows))
	for _, row := range rows {
		fields := map[string][]string{}
		if v := row.DepartmentLabel.Or(""); v != "" {
			fields["department"] = []string{v}
		}
		if row.EmploymentStatusLabel != "" {
			fields["employmentType"] = []string{row.EmploymentStatusLabel}
		}
		workMode := bamboohr.WorkModeLabel(row.LocationType.Or(""))
		if workMode != "" {
			fields["workplaceType"] = []string{workMode}
		}

		jobs = append(jobs, dumpJob{
			summary: JobSummary{
				JobID:    row.ID,
				Title:    row.JobOpeningName,
				Location: bambooHRListLocation(&row),
				// The list feed carries no posting date; it lives only on
				// the detail endpoint.
				PostedAt: "",
				URL:      bambooHRPostingURL(slug, row.ID),
			},
			sortKey: time.Time{}, // no posting date in the dump; ordering falls to rank, then id
			orgUnit: row.DepartmentLabel.Or(""),
			// The dump carries no description, so tier-3 query matching has
			// nothing to search: queries cover titles and departments only.
			description: "",
			locations:   bambooHRSearchLocations(&row, workMode),
			fields:      fields,
			isRemote:    row.LocationType.Or("") == "1",
		})
	}
	return jobs, nil
}

// bambooHRPostingURL builds the human-clickable posting page, the same URL
// the detail endpoint reports as jobOpeningShareUrl.
func bambooHRPostingURL(slug, id string) string {
	return fmt.Sprintf("https://%s.bamboohr.com/careers/%s", slug, id)
}

// bambooHRListLocation renders a list row's display location, preferring
// the structured `location` and falling back to `atsLocation` (which alone
// carries the country) when the former is all-null.
func bambooHRListLocation(row *bamboohr.ListJob) string {
	if s := bambooHRJoin(row.Location.City.Or(""), row.Location.State.Or("")); s != "" {
		return s
	}
	return bambooHRJoin(row.AtsLocation.City.Or(""), row.AtsLocation.State.Or(""), row.AtsLocation.Country.Or(""))
}

// bambooHRSearchLocations joins every location string a row carries for
// fuzzy matching, including the work-mode label so "remote"/"hybrid"
// queries hit rows whose only locality signal is locationType.
func bambooHRSearchLocations(row *bamboohr.ListJob, workMode string) string {
	parts := []string{
		row.Location.City.Or(""),
		row.Location.State.Or(""),
		row.AtsLocation.City.Or(""),
		row.AtsLocation.State.Or(""),
		row.AtsLocation.Province.Or(""),
		row.AtsLocation.Country.Or(""),
		workMode,
	}
	kept := make([]string, 0, len(parts))
	for _, p := range parts {
		if p != "" {
			kept = append(kept, p)
		}
	}
	return strings.Join(kept, "; ")
}

func bambooHRJoin(parts ...string) string {
	kept := make([]string, 0, len(parts))
	for _, p := range parts {
		if p != "" {
			kept = append(kept, p)
		}
	}
	return strings.Join(kept, ", ")
}

func bambooHRDescription(content string) string {
	if content == "" {
		return ""
	}
	text, err := html2text.FromString(content, html2text.Options{})
	if err != nil {
		return content
	}
	return text
}
