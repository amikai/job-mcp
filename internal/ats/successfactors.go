package ats

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"maps"
	"net/http"
	"net/url"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/jaytaylor/html2text"

	"github.com/amikai/openings-mcp/internal/provider/successfactors"
)

var _ Adapter = (*SuccessFactorsAdapter)(nil)

// successFactorsDateLayout matches java.util.Date#toString, the format
// SuccessFactors emits for itemprop="datePosted" (e.g.
// "Mon Jul 13 00:00:00 UTC 2026").
const successFactorsDateLayout = "Mon Jan 2 15:04:05 MST 2006"

// successFactorsUpstreamPageSize is the fixed row count the /search/ table
// always returns per request, independent of startRow (see openapi.yaml).
const successFactorsUpstreamPageSize = 25

// maxFilterCombinations caps the cartesian product of OR'd filter values
// across dimensions (see searchWithFanout) so a broad selection fails
// loudly, asking the caller to narrow it, instead of firing an unbounded
// number of upstream requests.
const maxFilterCombinations = 12

// maxPagesPerCombination caps how many upstream pages searchWithFanout
// fetches for any single filter combination (25 * 10 = 250 jobs) before
// erroring rather than silently returning a partial merge.
const maxPagesPerCombination = 10

// SuccessFactorsAdapter serves SAP SuccessFactors Career Site Builder
// tenants. Search and detail are server-rendered HTML pages (see
// internal/provider/successfactors/openapi.yaml); slugs are lowercase
// career-site hostnames rather than a shared-host tenant token, since every
// tenant has its own custom domain.
type SuccessFactorsAdapter struct {
	hc      *http.Client
	baseURL func(host string) string
}

func NewSuccessFactorsAdapter(hc *http.Client) *SuccessFactorsAdapter {
	return &SuccessFactorsAdapter{
		hc:      hc,
		baseURL: func(host string) string { return "https://" + host },
	}
}

func (a *SuccessFactorsAdapter) Name() string { return "successfactors" }

func (a *SuccessFactorsAdapter) Roster() []CompanyInfo {
	infos := make([]CompanyInfo, 0, len(successfactors.Companies))
	for _, c := range successfactors.Companies {
		infos = append(infos, CompanyInfo{Slug: c.Host, Name: c.Name})
	}
	return infos
}

// ParseCareersURL only recognizes curated hosts: unlike Teamtailor's shared
// *.teamtailor.com suffix, every SuccessFactors CSB tenant uses its own
// custom domain with no common pattern to match uncurated tenants against.
func (a *SuccessFactorsAdapter) ParseCareersURL(u *url.URL) (string, bool) {
	host := strings.ToLower(u.Hostname())
	if _, ok := successfactors.CompaniesByHost[host]; !ok {
		return "", false
	}
	return host, true
}

func (a *SuccessFactorsAdapter) Search(ctx context.Context, slug string, p SearchParams) (*SearchResult, error) {
	c, err := a.resolveSlug(slug)
	if err != nil {
		return nil, err
	}

	// filterValues holds every resolved value per dimension (preserving
	// FilterSet's OR semantics within a key); combos expands that into the
	// single-value-per-dimension requests the upstream single-select
	// dropdowns can each express.
	filterValues, err := a.resolveFilters(ctx, c, p.Filters)
	if err != nil {
		return nil, err
	}
	combos := filterCombinations(filterValues)

	client := successfactors.NewClient(a.baseURL(c.Host), a.hc)
	base := successfactors.SearchRequest{Query: p.Query, LocationSearch: strings.TrimSpace(p.Location)}
	page := clampPage(p.Page)

	if len(combos) <= 1 {
		// Fast path: no OR fan-out needed (zero or one value per
		// dimension), so the unified page maps directly onto one upstream
		// request, same as before dynamic filters existed.
		req := base
		req.Filters = combos[0]
		req.StartRow = (page - 1) * pageSize
		res, err := client.Search(ctx, &req)
		if err != nil {
			return nil, fmt.Errorf("successfactors: search %q: %w", c.Host, err)
		}
		// The upstream table always returns (up to) 25 rows regardless of
		// startRow (see openapi.yaml); trim to the unified pageSize.
		jobs := res.Jobs
		if len(jobs) > pageSize {
			jobs = jobs[:pageSize]
		}
		return &SearchResult{
			Jobs:       successFactorsJobSummaries(jobs, c.Host),
			TotalCount: res.TotalCount,
			Page:       page,
			TotalPages: totalPages(res.TotalCount),
		}, nil
	}

	merged, err := searchWithFanout(ctx, client, base, combos)
	if err != nil {
		return nil, fmt.Errorf("successfactors: search %q: %w", c.Host, err)
	}
	total := len(merged)
	pageIndex := page - 1
	start := total
	if pageIndex <= total/pageSize {
		start = pageIndex * pageSize
	}
	end := start + min(pageSize, total-start)
	return &SearchResult{
		Jobs:       successFactorsJobSummaries(merged[start:end], c.Host),
		TotalCount: total,
		Page:       page,
		TotalPages: totalPages(total),
	}, nil
}

// filterCombinations expands OR'd filter values into every AND'd
// single-value combination the upstream's single-select dropdowns can
// express in one request each: for dimensions d1={a,b} and d2={x}, that's
// {d1:a,d2:x} and {d1:b,d2:x} — the union of those two requests' results is
// exactly "(d1=a OR d1=b) AND d2=x". Always returns at least one (possibly
// nil-map) combination, so callers don't special-case the no-filter case.
func filterCombinations(filterValues map[string][]string) []map[string]string {
	if len(filterValues) == 0 {
		return []map[string]string{nil}
	}
	dims := slices.Sorted(maps.Keys(filterValues))

	combos := []map[string]string{{}}
	for _, dim := range dims {
		next := make([]map[string]string, 0, len(combos)*len(filterValues[dim]))
		for _, combo := range combos {
			for _, v := range filterValues[dim] {
				c := make(map[string]string, len(combo)+1)
				maps.Copy(c, combo)
				c[dim] = v
				next = append(next, c)
			}
		}
		combos = next
	}
	return combos
}

// searchWithFanout runs one upstream request per filter combination,
// paginating each one fully, then unions and dedupes the results by job
// ID before the caller applies its own page — the same fetch-all-then-
// paginate-locally shape searchDump uses for full-dump providers, just
// assembled from several already-server-filtered upstream calls instead of
// one full board dump. Bounded by maxFilterCombinations and
// maxPagesPerCombination so a broad OR selection fails loudly instead of
// silently returning a partial merge.
func searchWithFanout(
	ctx context.Context,
	client *successfactors.Client,
	base successfactors.SearchRequest,
	combos []map[string]string,
) ([]successfactors.Job, error) {
	if len(combos) > maxFilterCombinations {
		return nil, fmt.Errorf(
			"filter selection expands to %d combinations (max %d); narrow the OR'd values",
			len(combos), maxFilterCombinations,
		)
	}

	seen := make(map[string]successfactors.Job)
	for _, combo := range combos {
		for page := range maxPagesPerCombination {
			req := base
			req.Filters = combo
			req.StartRow = page * successFactorsUpstreamPageSize
			res, err := client.Search(ctx, &req)
			if err != nil {
				return nil, err
			}
			for _, j := range res.Jobs {
				seen[j.ID] = j
			}
			if len(res.Jobs) == 0 || (page+1)*successFactorsUpstreamPageSize >= res.TotalCount {
				break
			}
			if page == maxPagesPerCombination-1 {
				return nil, fmt.Errorf(
					"filter combination %v has more than %d results; narrow the search",
					combo, maxPagesPerCombination*successFactorsUpstreamPageSize,
				)
			}
		}
	}

	merged := make([]successfactors.Job, 0, len(seen))
	for _, j := range seen {
		merged = append(merged, j)
	}
	slices.SortFunc(merged, func(a, b successfactors.Job) int {
		ai, aErr := strconv.ParseInt(a.ID, 10, 64)
		bi, bErr := strconv.ParseInt(b.ID, 10, 64)
		if aErr == nil && bErr == nil {
			return cmp.Compare(bi, ai) // descending = newest first, matching this platform's ID scheme
		}
		return strings.Compare(b.ID, a.ID)
	})
	return merged, nil
}

func (a *SuccessFactorsAdapter) Filters(ctx context.Context, slug string) (FilterSet, error) {
	c, err := a.resolveSlug(slug)
	if err != nil {
		return nil, err
	}
	client := successfactors.NewClient(a.baseURL(c.Host), a.hc)
	res, err := client.FacetValues(ctx, &successfactors.SearchRequest{})
	if err != nil {
		return nil, fmt.Errorf("successfactors: facets %q: %w", c.Host, err)
	}

	seen := make(map[string]map[string]struct{}, len(res.Facets))
	for dimension, options := range res.Facets {
		if len(options) == 0 {
			continue
		}
		labels := make(map[string]struct{}, len(options))
		for _, o := range options {
			labels[displayLabel(o)] = struct{}{}
		}
		seen[dimension] = labels
	}
	return toFilterSet(seen), nil
}

func (a *SuccessFactorsAdapter) Detail(ctx context.Context, slug, jobID string) (*JobDetail, error) {
	c, err := a.resolveSlug(slug)
	if err != nil {
		return nil, err
	}
	client := successfactors.NewClient(a.baseURL(c.Host), a.hc)
	d, err := client.JobDetail(ctx, jobID)
	if errors.Is(err, successfactors.ErrJobNotFound) {
		return nil, fmt.Errorf(
			"successfactors: job %q not found for company %q; pass a job_id exactly as returned by the job search",
			jobID, slug,
		)
	}
	if err != nil {
		return nil, fmt.Errorf("successfactors: fetch job %q for %q: %w", jobID, slug, err)
	}

	desc := d.DescriptionHTML
	if desc != "" {
		if text, err := html2text.FromString(desc, html2text.Options{}); err == nil {
			desc = text
		}
	}

	return &JobDetail{
		JobID:       jobID,
		Title:       d.Title,
		Company:     c.Name,
		Location:    d.Location,
		PostedAt:    successFactorsPostedAt(d.PostedAtRaw),
		URL:         successFactorsJobURL(c.Host, jobID),
		Description: desc,
	}, nil
}

func (a *SuccessFactorsAdapter) resolveSlug(slug string) (successfactors.Company, error) {
	c, ok := successfactors.CompaniesByHost[strings.ToLower(slug)]
	if !ok {
		return successfactors.Company{}, fmt.Errorf("successfactors: unknown company %q; pass a roster career-site host", slug)
	}
	return c, nil
}

// resolveFilters turns unified filter labels (as reported by Filters())
// into the upstream's raw facet values, preserving OR semantics (every
// resolved value per key, not just the first), probing one unfiltered
// facetValues call to learn the tenant's current options — mirrors
// Workday's probeFacets and Eightfold's resolveFilters. The probe is
// deliberately unscoped by query/location; narrower facets aren't needed
// just to resolve labels. Search expands the result via filterCombinations
// into the single-value-per-dimension requests the upstream's single-select
// dropdowns can each express.
func (a *SuccessFactorsAdapter) resolveFilters(ctx context.Context, c successfactors.Company, filters FilterSet) (map[string][]string, error) {
	if len(filters) == 0 {
		return nil, nil
	}

	client := successfactors.NewClient(a.baseURL(c.Host), a.hc)
	probe, err := client.FacetValues(ctx, &successfactors.SearchRequest{})
	if err != nil {
		return nil, fmt.Errorf("successfactors: facets %q: %w", c.Host, err)
	}

	valid := make(map[string]bool, len(probe.Facets))
	for dimension, options := range probe.Facets {
		if len(options) > 0 {
			valid[dimension] = true
		}
	}

	resolved := make(map[string][]string, len(filters))
	for key, values := range filters {
		options, ok := probe.Facets[key]
		if !ok || len(options) == 0 {
			return nil, errUnknownFilterKey(key, valid)
		}
		if len(values) == 0 {
			continue
		}
		resolvedValues := make([]string, 0, len(values))
		for _, label := range values {
			v, ok := resolveSuccessFactorsFacetValue(options, label)
			if !ok {
				labels := make([]string, len(options))
				for i, o := range options {
					labels[i] = displayLabel(o)
				}
				return nil, fmt.Errorf("filter value %q not found for %q; available: %s", label, key, strings.Join(labels, ", "))
			}
			resolvedValues = append(resolvedValues, v)
		}
		resolved[key] = resolvedValues
	}
	return resolved, nil
}

func resolveSuccessFactorsFacetValue(options []successfactors.FacetOption, label string) (string, bool) {
	for _, o := range options {
		if strings.EqualFold(displayLabel(o), label) {
			return o.Name, true
		}
	}
	return "", false
}

func displayLabel(o successfactors.FacetOption) string {
	if o.Translated != "" {
		return o.Translated
	}
	return o.Name
}

func successFactorsJobURL(host, id string) string {
	return fmt.Sprintf("https://%s/job/%s/%s/", host, id, id)
}

func successFactorsJobSummaries(jobs []successfactors.Job, host string) []JobSummary {
	out := make([]JobSummary, 0, len(jobs))
	for _, j := range jobs {
		out = append(out, JobSummary{
			JobID:    j.ID,
			Title:    j.Title,
			Location: j.Location,
			URL:      successFactorsJobURL(host, j.ID),
		})
	}
	return out
}

// successFactorsPostedAt renders the unified PostedAt format when the
// upstream's Java Date#toString value parses cleanly; otherwise it passes
// the raw text through (some tenants omit datePosted entirely, in which
// case raw is already "").
func successFactorsPostedAt(raw string) string {
	if raw == "" {
		return ""
	}
	t, err := time.Parse(successFactorsDateLayout, raw)
	if err != nil {
		return raw
	}
	return isoDate(t)
}
