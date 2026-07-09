// Package ats exposes one company-based interface over the ATS providers.
package ats

import (
	"context"
	"time"
)

// PageSize is the largest safe uniform page size; some Workday tenants cap it at 20.
const PageSize = 20

// clampPage and totalPages implement the shared 1-based pagination contract.
func clampPage(p int) int { return max(p, 1) }

func totalPages(total int) int { return (total + PageSize - 1) / PageSize }

// isoDate renders the unified PostedAt format for real upstream timestamps.
func isoDate(t time.Time) string { return t.UTC().Format("2006-01-02") }

// Adapter implements the unified search interface for one ATS. Slugs come from
// Roster and are validated by Registry before reaching an adapter.
type Adapter interface {
	// Name identifies the adapter in logs and errors.
	Name() string
	// Roster lists the curated companies served by this ATS.
	Roster() []CompanyInfo
	Search(ctx context.Context, slug string, p SearchParams) (*SearchResult, error)
	Filters(ctx context.Context, slug string) (FilterSet, error)
	Detail(ctx context.Context, slug, jobID string) (*JobDetail, error)
}

// CompanyInfo contains the name and slug the registry needs for resolution.
type CompanyInfo struct {
	Slug string // unique key; the provider roster's tenant/site/board slug
	Name string // display name; the resolver also matches on it
}

// SearchParams are the provider-independent search inputs.
type SearchParams struct {
	Query    string              // keywords: titles, skills, tech — never locations
	Location string              // fuzzy text match; full-dump adapters special-case "remote" via their remote fields, workday matches location facet labels
	Filters  map[string][]string // escape hatch; keys/values come from Filters(); OR within a key, AND across keys
	Page     int                 // 1-based; values < 1 mean page 1
}

// SearchResult is one page of unified search results.
type SearchResult struct {
	Jobs       []JobSummary
	TotalCount int
	Page       int
	TotalPages int
}

// JobSummary omits full descriptions so search responses stay small.
type JobSummary struct {
	JobID    string // provider-native id (workday externalPath, lever uuid, ashby id)
	Title    string
	Location string
	PostedAt string // ISO 8601 date where the upstream provides one; otherwise its raw text
	URL      string // human-clickable posting page
}

// FilterSet maps a filter dimension to its current display values.
type FilterSet map[string][]string

// JobDetail is a full posting with a plain-text description.
type JobDetail struct {
	JobID       string
	Title       string
	Company     string
	Location    string
	PostedAt    string
	URL         string
	Description string
}
