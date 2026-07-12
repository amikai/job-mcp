package ats

import (
	"cmp"
	"context"
	"fmt"
	"math"
	"net/http"
	"net/url"
	"slices"
	"strconv"
	"strings"

	"github.com/jaytaylor/html2text"

	"github.com/amikai/openings-mcp/internal/provider/smartrecruiters"
)

var _ Adapter = (*SmartRecruitersAdapter)(nil)

// SmartRecruitersAdapter serves SmartRecruiters-hosted companies via the
// public Posting API. Search runs server-side: the unified Location folds
// into the q param (which full-text matches titles and location text), and
// department filter labels resolve to ids via one departments call when
// set — the stateless price, like Workday's facet probe.
type SmartRecruitersAdapter struct {
	client *smartrecruiters.Client
}

func NewSmartRecruitersAdapter(baseURL string, hc *http.Client) (*SmartRecruitersAdapter, error) {
	c, err := smartrecruiters.NewClient(baseURL, smartrecruiters.WithClient(hc))
	if err != nil {
		return nil, err
	}
	return &SmartRecruitersAdapter{client: c}, nil
}

func (a *SmartRecruitersAdapter) Name() string { return "smartrecruiters" }

func (a *SmartRecruitersAdapter) Roster() []CompanyInfo {
	infos := make([]CompanyInfo, 0, len(smartrecruiters.Companies))
	for _, c := range smartrecruiters.Companies {
		infos = append(infos, CompanyInfo{Slug: strings.ToLower(c.CompanyIdentifier), Name: c.Name})
	}
	return infos
}

// ParseCareersURL recognizes jobs.smartrecruiters.com career-site URLs; the
// first path segment is the companyIdentifier, which alone addresses a
// company (the API accepts it case-insensitively), so non-roster companies
// need no special slug form. An unknown identifier cannot be validated —
// the list endpoint answers HTTP 200 with zero results — so a typo'd URL
// degrades to an empty search, mirroring the raw API.
func (a *SmartRecruitersAdapter) ParseCareersURL(u *url.URL) (string, bool) {
	if strings.ToLower(u.Hostname()) != "jobs.smartrecruiters.com" {
		return "", false
	}
	id := firstPathSegment(u)
	if id == "" {
		return "", false
	}
	return strings.ToLower(id), true
}

// resolveSmartRecruitersCompany maps a slug to the roster's
// canonically-cased identifier (used in derived public URLs) and display
// name. Non-roster slugs from ParseCareersURL pass through as both.
func resolveSmartRecruitersCompany(slug string) (identifier, name string) {
	if c, ok := smartrecruiters.CompaniesByIdentifier[slug]; ok {
		return c.CompanyIdentifier, c.Name
	}
	return slug, slug
}

func (a *SmartRecruitersAdapter) Search(ctx context.Context, slug string, p SearchParams) (*SearchResult, error) {
	page := clampPage(p.Page)
	pageIndex := page - 1
	if pageIndex > math.MaxInt/PageSize {
		return nil, fmt.Errorf("smartrecruiters: page %d is too large; retry with a smaller page", page)
	}
	params := smartrecruiters.ListPostingsParams{
		CompanyIdentifier: slug,
		Limit:             smartrecruiters.NewOptInt(PageSize),
		Offset:            smartrecruiters.NewOptInt(pageIndex * PageSize),
	}
	// q full-text matches titles and location text upstream, so the
	// unified Location folds into it rather than guessing among the
	// exact-match country/region/city params.
	if q := strings.TrimSpace(strings.TrimSpace(p.Query) + " " + strings.TrimSpace(p.Location)); q != "" {
		params.Q = smartrecruiters.NewOptString(q)
	}
	if err := a.applyFilters(ctx, slug, p.Filters, &params); err != nil {
		return nil, err
	}
	rsp, err := a.client.ListPostings(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("smartrecruiters: search %q: %w", slug, err)
	}
	identifier, _ := resolveSmartRecruitersCompany(slug)
	jobs := make([]JobSummary, 0, len(rsp.Content))
	for _, it := range rsp.Content {
		id := it.ID.Value
		if id == "" {
			// A posting with no id can't be detailed; skip rather than
			// hand out an unusable job_id.
			continue
		}
		jobs = append(jobs, JobSummary{
			JobID:    id,
			Title:    it.Name.Value,
			Location: it.Location.Value.FullLocation.Value,
			PostedAt: smartRecruitersPostedAt(it.ReleasedDate),
			URL:      smartRecruitersPostingURL(identifier, id),
		})
	}
	return &SearchResult{
		Jobs:       jobs,
		TotalCount: rsp.TotalFound,
		Page:       page,
		TotalPages: totalPages(rsp.TotalFound),
	}, nil
}

// smartRecruitersLocationTypes maps the location_type filter's display
// values to the API's locationType enum.
var smartRecruitersLocationTypes = map[string]smartrecruiters.ListPostingsLocationTypeItem{
	"remote": smartrecruiters.ListPostingsLocationTypeItemREMOTE,
	"hybrid": smartrecruiters.ListPostingsLocationTypeItemHYBRID,
	"onsite": smartrecruiters.ListPostingsLocationTypeItemONSITE,
}

// applyFilters maps unified filters onto the list endpoint's query params,
// failing with teaching errors that name the valid alternatives.
func (a *SmartRecruitersAdapter) applyFilters(ctx context.Context, slug string, filters FilterSet, params *smartrecruiters.ListPostingsParams) error {
	for key, values := range filters {
		switch key {
		case "department":
			ids, err := a.resolveDepartments(ctx, slug, values)
			if err != nil {
				return err
			}
			// Comma-joined ids OR together (verified live against Equinox:
			// 129 + 23 postings filter to 152).
			params.Department = smartrecruiters.NewOptString(strings.Join(ids, ","))
		case "location_type":
			for _, v := range values {
				lt, ok := smartRecruitersLocationTypes[strings.ToLower(strings.TrimSpace(v))]
				if !ok {
					return fmt.Errorf("filter value %q not found for %q; available: Hybrid, Onsite, Remote", v, key)
				}
				params.LocationType = append(params.LocationType, lt)
			}
		default:
			return errUnknownFilterKey(key, map[string]bool{"department": true, "location_type": true})
		}
	}
	return nil
}

// resolveDepartments maps department display labels to ids via one
// departments call, matching labels case-insensitively.
func (a *SmartRecruitersAdapter) resolveDepartments(ctx context.Context, slug string, values []string) ([]string, error) {
	deps, err := a.departments(ctx, slug)
	if err != nil {
		return nil, err
	}
	byLabel := make(map[string]string, len(deps))
	labels := make([]string, 0, len(deps))
	for _, d := range deps {
		lower := strings.ToLower(d.label)
		if _, ok := byLabel[lower]; !ok {
			byLabel[lower] = d.id
			labels = append(labels, d.label)
		}
	}
	ids := make([]string, 0, len(values))
	for _, v := range values {
		id, ok := byLabel[strings.ToLower(strings.TrimSpace(v))]
		if !ok {
			slices.Sort(labels)
			const maxListed = 20
			listed := labels
			suffix := ""
			if len(listed) > maxListed {
				listed = listed[:maxListed]
				suffix = ", …"
			}
			return nil, fmt.Errorf("filter value %q not found for %q; available: %s%s", v, "department", strings.Join(listed, ", "), suffix)
		}
		ids = append(ids, id)
	}
	return ids, nil
}

// smartRecruitersPostedAt guards a present-but-missing releasedDate:
// OptDateTime's zero Value would otherwise format as a fake date.
func smartRecruitersPostedAt(t smartrecruiters.OptDateTime) string {
	v, ok := t.Get()
	if !ok {
		return ""
	}
	return isoDate(v)
}

// smartRecruitersPostingURL derives the public posting page. List items
// carry no postingUrl; slug-less URLs (no title suffix) resolve fine on
// jobs.smartrecruiters.com.
func smartRecruitersPostingURL(identifier, id string) string {
	return "https://jobs.smartrecruiters.com/" + url.PathEscape(identifier) + "/" + url.PathEscape(id)
}

// smartRecruitersDepartment is one non-archived, labeled department: the
// id the API's department query param takes and the display label
// Filters() reports.
type smartRecruitersDepartment struct {
	id    string
	label string
}

// departments fetches the company's departments, dropping archived and
// unlabeled entries. DepartmentId is a string-or-int sum (the API returns
// both); ids normalize to their decimal string form either way.
func (a *SmartRecruitersAdapter) departments(ctx context.Context, slug string) ([]smartRecruitersDepartment, error) {
	rsp, err := a.client.ListDepartments(ctx, smartrecruiters.ListDepartmentsParams{CompanyIdentifier: slug})
	if err != nil {
		return nil, fmt.Errorf("smartrecruiters: list departments for %q: %w", slug, err)
	}
	deps := make([]smartRecruitersDepartment, 0, len(rsp.Content))
	for _, d := range rsp.Content {
		if d.Archived.Or(false) || d.Label.Value == "" {
			continue
		}
		id, ok := smartRecruitersDepartmentID(d.ID)
		if !ok {
			continue
		}
		deps = append(deps, smartRecruitersDepartment{id: id, label: d.Label.Value})
	}
	return deps, nil
}

func smartRecruitersDepartmentID(opt smartrecruiters.OptDepartmentId) (string, bool) {
	v, ok := opt.Get()
	if !ok {
		return "", false
	}
	if s, ok := v.GetString(); ok {
		return s, s != ""
	}
	if n, ok := v.GetInt(); ok {
		return strconv.Itoa(n), true
	}
	return "", false
}

func (a *SmartRecruitersAdapter) Filters(ctx context.Context, slug string) (FilterSet, error) {
	deps, err := a.departments(ctx, slug)
	if err != nil {
		return nil, err
	}
	// location_type is a static API enum, not tenant data.
	fs := FilterSet{"location_type": []string{"Hybrid", "Onsite", "Remote"}}
	seen := make(map[string]bool, len(deps))
	labels := make([]string, 0, len(deps))
	for _, d := range deps {
		if seen[d.label] {
			continue
		}
		seen[d.label] = true
		labels = append(labels, d.label)
	}
	if len(labels) > 0 {
		slices.Sort(labels)
		fs["department"] = labels
	}
	return fs, nil
}

func (a *SmartRecruitersAdapter) Detail(ctx context.Context, slug, jobID string) (*JobDetail, error) {
	res, err := a.client.GetPosting(ctx, smartrecruiters.GetPostingParams{
		CompanyIdentifier: slug,
		PostingId:         jobID,
	})
	if err != nil {
		return nil, fmt.Errorf("smartrecruiters: fetch job %q for %q: %w", jobID, slug, err)
	}
	d, ok := res.(*smartrecruiters.Posting)
	if !ok {
		// The only other GetPostingRes variant is the 404
		// PostingErrorResponse, for an unknown company or posting id.
		return nil, fmt.Errorf("smartrecruiters: job %q not found for company %q; pass a job_id exactly as returned by the job search", jobID, slug)
	}
	_, name := resolveSmartRecruitersCompany(slug)
	return &JobDetail{
		JobID:       cmp.Or(d.ID.Value, jobID),
		Title:       d.Name.Value,
		Company:     cmp.Or(d.Company.Value.Name.Value, name),
		Location:    d.Location.Value.FullLocation.Value,
		PostedAt:    smartRecruitersPostedAt(d.ReleasedDate),
		URL:         d.PostingUrl.Value,
		Description: smartRecruitersDescription(d.JobAd),
	}, nil
}

// smartRecruitersDescription joins the jobAd's non-empty HTML sections as
// titled plain-text blocks, in the API's canonical section order.
func smartRecruitersDescription(jobAd smartrecruiters.OptJobAdSections) string {
	sections, ok := jobAd.Value.Sections.Get()
	if !ok {
		return ""
	}
	ordered := []struct {
		fallbackTitle string
		sec           smartrecruiters.OptJobAdSection
	}{
		{"Company Description", sections.CompanyDescription},
		{"Job Description", sections.JobDescription},
		{"Qualifications", sections.Qualifications},
		{"Additional Information", sections.AdditionalInformation},
	}
	var parts []string
	for _, s := range ordered {
		sec, ok := s.sec.Get()
		if !ok || sec.Text.Value == "" {
			continue
		}
		text, err := html2text.FromString(sec.Text.Value, html2text.Options{})
		if err != nil {
			// Keep the section as raw HTML rather than dropping it
			// (mirrors cmd/smartrecruiters's printSection).
			text = sec.Text.Value
		}
		parts = append(parts, cmp.Or(sec.Title.Value, s.fallbackTitle)+":\n"+text)
	}
	return strings.Join(parts, "\n\n")
}
