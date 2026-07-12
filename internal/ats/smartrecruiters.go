package ats

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"slices"
	"strconv"
	"strings"

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
	return nil, errors.New("smartrecruiters: Search not implemented yet")
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
	return nil, errors.New("smartrecruiters: Detail not implemented yet")
}
