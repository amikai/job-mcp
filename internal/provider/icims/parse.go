package icims

import (
	"encoding/json"
	"errors"
	"regexp"
	"strconv"
	"strings"

	"github.com/PuerkitoBio/goquery"
)

// pageOfPattern matches "Page 1 of 31" inside paging controls.
var pageOfPattern = regexp.MustCompile(`(?i)Page\s+(\d+)\s+of\s+(\d+)`)

// jobHrefPattern extracts id and slug from /jobs/{id}/{slug}/job links.
var jobHrefPattern = regexp.MustCompile(`(?i)/jobs/(\d+)/([^/?#]+)/job`)

// parseSearchHTML extracts job cards and pagination metadata.
//
// A page with the search form (or job table chrome) but zero cards is a
// genuine empty result set. A page that looks nothing like the portal is an
// error so bot-challenge / login walls surface clearly.
func parseSearchHTML(doc *goquery.Document) ([]Job, int, int, error) {
	var jobs []Job
	seen := make(map[string]struct{})
	doc.Find("li.iCIMS_JobCardItem").Each(func(_ int, card *goquery.Selection) {
		job, ok := parseJobCard(card)
		if !ok {
			return
		}
		if _, dup := seen[job.ID]; dup {
			return
		}
		seen[job.ID] = struct{}{}
		jobs = append(jobs, job)
	})

	totalPages := parseTotalPages(doc)
	if len(jobs) == 0 && totalPages == 0 && !looksLikeSearchPage(doc) {
		return nil, 0, 0, errors.New("unrecognized search page: no job cards, no pagination, and no search form")
	}
	if totalPages == 0 {
		// Single-page boards often omit the "Page X of Y" label.
		totalPages = 1
	}
	return jobs, totalPages, len(jobs), nil
}

func parseJobCard(card *goquery.Selection) (Job, bool) {
	link := card.Find("a.iCIMS_Anchor").FilterFunction(func(_ int, s *goquery.Selection) bool {
		href, _ := s.Attr("href")
		return jobHrefPattern.MatchString(href)
	}).First()
	if link.Length() == 0 {
		// Fallback: any anchor into /jobs/{id}/.../job
		card.Find("a[href]").EachWithBreak(func(_ int, s *goquery.Selection) bool {
			href, _ := s.Attr("href")
			if jobHrefPattern.MatchString(href) {
				link = s
				return false
			}
			return true
		})
	}
	if link.Length() == 0 {
		return Job{}, false
	}

	href, _ := link.Attr("href")
	id, slug := jobIDAndSlugFromHref(href)
	if id == "" {
		return Job{}, false
	}

	title := strings.TrimSpace(link.Find("h3").First().Text())
	if title == "" {
		title = strings.TrimSpace(link.Text())
	}
	if title == "" {
		return Job{}, false
	}

	location := extractCardLocation(card)
	return Job{
		ID:       id,
		Slug:     slug,
		Title:    title,
		Location: location,
	}, true
}

func extractCardLocation(card *goquery.Selection) string {
	// Prefer the sr-only "Location" / "Job Locations" label's following span.
	var loc string
	card.Find("span.sr-only.field-label").EachWithBreak(func(_ int, s *goquery.Selection) bool {
		label := strings.ToLower(strings.TrimSpace(s.Text()))
		if !strings.Contains(label, "location") {
			return true
		}
		// Next sibling span holds the value.
		next := s.NextFiltered("span")
		if next.Length() == 0 {
			next = s.Parent().Find("span").Not(".sr-only").First()
		}
		loc = strings.TrimSpace(next.Text())
		return loc == ""
	})
	return loc
}

func jobIDAndSlugFromHref(href string) (id, slug string) {
	m := jobHrefPattern.FindStringSubmatch(href)
	if m == nil {
		return "", ""
	}
	return m[1], m[2]
}

func parseTotalPages(doc *goquery.Document) int {
	// Prefer visible paging batch text.
	text := strings.TrimSpace(doc.Find(".iCIMS_PagingBatch").First().Text())
	if text == "" {
		text = strings.TrimSpace(doc.Find(".iCIMS_Paging").Text())
	}
	if m := pageOfPattern.FindStringSubmatch(text); m != nil {
		n, err := strconv.Atoi(m[2])
		if err == nil && n > 0 {
			return n
		}
	}
	// Fall back to scanning full page text once.
	if m := pageOfPattern.FindStringSubmatch(doc.Text()); m != nil {
		n, err := strconv.Atoi(m[2])
		if err == nil && n > 0 {
			return n
		}
	}
	return 0
}

func looksLikeSearchPage(doc *goquery.Document) bool {
	if doc.Find("#searchForm, form[name='searchForm'], input[name='searchKeyword']").Length() > 0 {
		return true
	}
	return doc.Find(".iCIMS_JobsTable, .iCIMS_Paging").Length() > 0
}

// parseJobDetailHTML reads the schema.org JobPosting JSON-LD block.
// Returns ok=false when no JobPosting is present (expired IDs, listing
// fallback, or unrecognized template).
func parseJobDetailHTML(doc *goquery.Document, id string) (*JobDetailResponse, bool) {
	posting := findJobPosting(doc)
	if posting == nil {
		return nil, false
	}

	title := strings.TrimSpace(asString(posting["title"]))
	if title == "" {
		return nil, false
	}

	detail := &JobDetailResponse{
		ID:              id,
		Title:           title,
		DescriptionHTML: strings.TrimSpace(asString(posting["description"])),
		PostedAtRaw:     strings.TrimSpace(asString(posting["datePosted"])),
		EmploymentType:  strings.TrimSpace(asString(posting["employmentType"])),
		Employer:        hiringOrgName(posting["hiringOrganization"]),
		Location:        locationFromJSONLD(posting["jobLocation"]),
		URL:             strings.TrimSpace(asString(posting["url"])),
		Category:        strings.TrimSpace(asString(posting["occupationalCategory"])),
	}
	return detail, true
}

func findJobPosting(doc *goquery.Document) map[string]any {
	var found map[string]any
	doc.Find(`script[type="application/ld+json"]`).EachWithBreak(func(_ int, s *goquery.Selection) bool {
		raw := strings.TrimSpace(s.Text())
		if raw == "" {
			return true
		}
		var data any
		if err := json.Unmarshal([]byte(raw), &data); err != nil {
			return true
		}
		for _, candidate := range iterLDDicts(data) {
			if typeIs(candidate["@type"], "JobPosting") {
				found = candidate
				return false
			}
		}
		return true
	})
	return found
}

func iterLDDicts(node any) []map[string]any {
	switch v := node.(type) {
	case map[string]any:
		out := []map[string]any{v}
		if graph, ok := v["@graph"].([]any); ok {
			for _, g := range graph {
				if m, ok := g.(map[string]any); ok {
					out = append(out, m)
				}
			}
		}
		return out
	case []any:
		var out []map[string]any
		for _, item := range v {
			out = append(out, iterLDDicts(item)...)
		}
		return out
	default:
		return nil
	}
}

func typeIs(v any, want string) bool {
	switch t := v.(type) {
	case string:
		return strings.EqualFold(t, want)
	case []any:
		for _, item := range t {
			if s, ok := item.(string); ok && strings.EqualFold(s, want) {
				return true
			}
		}
	}
	return false
}

func asString(v any) string {
	s, _ := v.(string)
	return s
}

func hiringOrgName(v any) string {
	switch org := v.(type) {
	case string:
		return strings.TrimSpace(org)
	case map[string]any:
		return strings.TrimSpace(asString(org["name"]))
	default:
		return ""
	}
}

func locationFromJSONLD(v any) string {
	candidates, ok := v.([]any)
	if !ok {
		if m, isMap := v.(map[string]any); isMap {
			candidates = []any{m}
		} else {
			return ""
		}
	}
	for _, c := range candidates {
		m, ok := c.(map[string]any)
		if !ok {
			continue
		}
		addr, _ := m["address"].(map[string]any)
		if addr == nil {
			continue
		}
		parts := make([]string, 0, 3)
		for _, key := range []string{"addressLocality", "addressRegion", "addressCountry"} {
			if s := strings.TrimSpace(asString(addr[key])); s != "" && !strings.EqualFold(s, "UNAVAILABLE") {
				parts = append(parts, s)
			}
		}
		if len(parts) > 0 {
			return strings.Join(parts, ", ")
		}
	}
	return ""
}

// isSearchLikeDetailBody reports whether a detail response body is actually
// a listing page (the 410 / expired-job fallback).
func isSearchLikeDetailBody(doc *goquery.Document) bool {
	return doc.Find("li.iCIMS_JobCardItem").Length() > 0 || looksLikeSearchPage(doc)
}
