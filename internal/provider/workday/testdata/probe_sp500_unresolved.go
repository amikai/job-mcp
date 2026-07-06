//go:build ignore

// probe_sp500_unresolved concurrently probes the Workday CXS API to find
// tenants for S&P 500 companies whose status is "not_found" (the
// "unresolved" bucket) in ../sp500_tenants.json.
//
// Workday tenants need three unknowns to build a working CXS URL — tenant
// slug, instance pod (wd1, wd5, ...), and site name — and site names are
// largely company-specific (e.g. eBay's is "TCGPlayer_External_Career"), so
// they can't be brute-forced reliably. Instead this exploits how the CXS
// API answers a wrong site name on a real tenant: POSTing to
// /wday/cxs/{tenant}/{anything}/jobs returns HTTP 404 with errorCode "S21"
// ("not found: Job_Posting_Site_ID=...") if the tenant+instance exist, or
// HTTP 422 if they don't. That confirms "this company uses Workday"
// independent of guessing the site correctly.
//
// Stage 1 (tenant probe): try generated tenant-slug candidates against
// every known instance pod with a nonsense site name; a 404/S21 response
// confirms the (tenant, instance) pair.
// Stage 2 (site probe): for each confirmed tenant, try a list of common
// site names (mined from the frequency of sites already in companies.yaml)
// plus a few tenant-derived guesses; a real HTTP 200 with a jobs array
// confirms the full (tenant, instance, site) triple.
//
// Only fully-confirmed triples (stage 2 success) get appended to
// companies.yaml — a tenant confirmed in stage 1 without a working site
// guess is reported but left out, since this package's invariant is that
// every entry's BaseURL() actually works.
//
// Run with: go run testdata/probe_sp500_unresolved.go [-fix] [-c N]
package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"
)

const userAgent = "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/124.0.0.0 Safari/537.36"

var instances = []string{
	"wd1", "wd2", "wd3", "wd4", "wd5",
	"wd12", "wd101", "wd102", "wd103", "wd108",
	"wd501", "wd502", "wd503", "wd505",
}

var genericSites = []string{
	"External", "Careers", "careers", "jobs", "Search", "search",
	"External_Careers", "External_Career_Site", "EXT", "SearchJobs", "Domestic",
	"ExternalCareers", "ExternalCareerSite", "ExternalSite", "Global",
	"External_Career", "Corporate", "Careers_External", "GLOBAL", "Jobs",
}

var suffixWords = map[string]bool{
	"corporation": true, "corp": true, "inc": true, "co": true, "company": true,
	"plc": true, "ltd": true, "enterprise": true, "enterprises": true,
	"technologies": true, "technology": true, "group": true, "holdings": true,
	"industries": true, "systems": true, "international": true, "the": true,
}

var parenRe = regexp.MustCompile(`\([^)]*\)`)
var nonAlnumRe = regexp.MustCompile(`[^A-Za-z0-9]+`)

func tenantCandidates(name string) []string {
	seen := map[string]bool{}
	var out []string
	add := func(s string) {
		s = strings.ToLower(nonAlnumRe.ReplaceAllString(s, ""))
		if s != "" && !seen[s] {
			seen[s] = true
			out = append(out, s)
		}
	}

	noParen := strings.TrimSpace(parenRe.ReplaceAllString(name, ""))
	add(name)
	add(noParen)
	add(stripSuffixWords(noParen))
	return out
}

func stripSuffixWords(s string) string {
	fields := strings.Fields(s)
	for len(fields) > 1 {
		last := strings.ToLower(strings.Trim(fields[len(fields)-1], ".,&"))
		if !suffixWords[last] {
			break
		}
		fields = fields[:len(fields)-1]
	}
	return strings.Join(fields, " ")
}

func siteCandidates(tenant string) []string {
	title := strings.ToUpper(tenant[:1]) + tenant[1:]
	templated := []string{
		title + "Careers",
		tenant + "_Careers",
		tenant + "careers",
		title + "_External_Career_Site",
		title + "ExternalCareerSite",
	}
	return append(append([]string{}, genericSites...), templated...)
}

type sp500Company struct {
	Company string `json:"company"`
	Status  string `json:"status"`
}

type sp500File struct {
	Companies []sp500Company `json:"companies"`
}

func loadUnresolved(path string) ([]string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var f sp500File
	if err := json.Unmarshal(data, &f); err != nil {
		return nil, err
	}
	var names []string
	for _, c := range f.Companies {
		if c.Status == "not_found" {
			names = append(names, c.Company)
		}
	}
	return names, nil
}

type ycompany struct {
	Name     string
	Tenant   string
	Instance string
	Site     string
}

var nameLineRe = regexp.MustCompile(`^- company: "([^"]+)"$`)
var tenantLineRe = regexp.MustCompile(`^\s*tenant: "([^"]+)"$`)
var instanceLineRe = regexp.MustCompile(`^\s*instance: "([^"]+)"$`)
var siteLineRe = regexp.MustCompile(`^\s*site: "([^"]+)"$`)

func loadCompaniesYAML(path string) ([]ycompany, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var cs []ycompany
	var cur ycompany
	for _, line := range strings.Split(string(data), "\n") {
		if m := nameLineRe.FindStringSubmatch(line); m != nil {
			if cur.Name != "" {
				cs = append(cs, cur)
			}
			cur = ycompany{Name: m[1]}
			continue
		}
		if m := tenantLineRe.FindStringSubmatch(line); m != nil {
			cur.Tenant = m[1]
			continue
		}
		if m := instanceLineRe.FindStringSubmatch(line); m != nil {
			cur.Instance = m[1]
			continue
		}
		if m := siteLineRe.FindStringSubmatch(line); m != nil {
			cur.Site = m[1]
			continue
		}
	}
	if cur.Name != "" {
		cs = append(cs, cur)
	}
	return cs, nil
}

func writeCompaniesYAML(path string, cs []ycompany) error {
	sort.Slice(cs, func(i, j int) bool { return cs[i].Name < cs[j].Name })
	var out []byte
	for _, c := range cs {
		out = append(out, fmt.Sprintf("- company: %q\n  tenant: %q\n  instance: %q\n  site: %q\n",
			c.Name, c.Tenant, c.Instance, c.Site)...)
	}
	return os.WriteFile(path, out, 0o644)
}

var httpClient = &http.Client{Timeout: 20 * time.Second}

// postCXS posts a minimal jobs search to a tenant/instance/site combo and
// returns the HTTP status and parsed errorCode (empty if the body wasn't an
// error envelope or wasn't valid JSON).
func postCXS(tenant, instance, site string) (status int, errorCode string, jobCount int, err error) {
	url := fmt.Sprintf("https://%s.%s.myworkdayjobs.com/wday/cxs/%s/%s/jobs", tenant, instance, tenant, site)
	body := []byte(`{"appliedFacets":{},"limit":1,"offset":0,"searchText":""}`)

	var lastErr error
	for attempt := 0; attempt < 2; attempt++ {
		req, rerr := http.NewRequest(http.MethodPost, url, bytes.NewReader(body))
		if rerr != nil {
			return 0, "", 0, rerr
		}
		req.Header.Set("User-Agent", userAgent)
		req.Header.Set("Accept", "application/json")
		req.Header.Set("Content-Type", "application/json")

		resp, rerr := httpClient.Do(req)
		if rerr != nil {
			lastErr = rerr
			continue
		}
		respBody, rerr := io.ReadAll(resp.Body)
		resp.Body.Close()
		if rerr != nil {
			lastErr = rerr
			continue
		}

		var envelope struct {
			ErrorCode string `json:"errorCode"`
			Jobs      []json.RawMessage `json:"jobPostings"`
		}
		_ = json.Unmarshal(respBody, &envelope)
		return resp.StatusCode, envelope.ErrorCode, len(envelope.Jobs), nil
	}
	return 0, "", 0, lastErr
}

type tenantHit struct {
	company  string
	tenant   string
	instance string
}

func main() {
	fix := flag.Bool("fix", false, "append fully-confirmed companies to companies.yaml")
	concurrency := flag.Int("c", 20, "max concurrent requests")
	flag.Parse()

	names, err := loadUnresolved("sp500_tenants.json")
	if err != nil {
		fmt.Fprintln(os.Stderr, "load sp500_tenants.json:", err)
		os.Exit(1)
	}
	fmt.Printf("probing %d unresolved companies\n\n", len(names))

	// Stage 1: find (tenant, instance) pairs that exist on Workday.
	type probeTask struct {
		company, tenant, instance string
	}
	var tasks []probeTask
	for _, name := range names {
		for _, tenant := range tenantCandidates(name) {
			for _, instance := range instances {
				tasks = append(tasks, probeTask{name, tenant, instance})
			}
		}
	}

	hits := make(chan tenantHit, len(tasks))
	taskCh := make(chan probeTask)
	var wg sync.WaitGroup
	for i := 0; i < *concurrency; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for t := range taskCh {
				status, code, _, err := postCXS(t.tenant, t.instance, "zzz-probe-nonexistent-site")
				if err != nil {
					continue
				}
				if status == http.StatusNotFound && code == "S21" {
					hits <- tenantHit{t.company, t.tenant, t.instance}
				}
			}
		}()
	}
	go func() {
		for _, t := range tasks {
			taskCh <- t
		}
		close(taskCh)
	}()
	wg.Wait()
	close(hits)

	confirmedTenant := map[string]tenantHit{} // company -> first confirmed hit
	claimants := map[string][]string{}        // "tenant/instance" -> companies that matched it
	for h := range hits {
		if _, ok := confirmedTenant[h.company]; !ok {
			confirmedTenant[h.company] = h
		}
		key := h.tenant + "/" + h.instance
		claimants[key] = append(claimants[key], h.company)
	}

	// A tenant slug shared by multiple different companies is a collision —
	// almost certainly an overly generic candidate matching an unrelated
	// real tenant, not evidence any of them use Workday. Drop it from
	// consideration entirely rather than guess which company (if any) it
	// really belongs to.
	ambiguous := map[string]bool{}
	for _, companies := range claimants {
		if len(companies) > 1 {
			for _, c := range companies {
				ambiguous[c] = true
				delete(confirmedTenant, c)
			}
		}
	}

	// Stage 2: for confirmed tenants, find a working site name.
	type siteResult struct {
		company, tenant, instance, site string
		jobs                            int
	}
	siteResults := make(chan siteResult, 4096)
	type siteTask struct {
		company, tenant, instance, site string
	}
	var siteTasks []siteTask
	for company, h := range confirmedTenant {
		for _, site := range siteCandidates(h.tenant) {
			siteTasks = append(siteTasks, siteTask{company, h.tenant, h.instance, site})
		}
	}

	var siteWG sync.WaitGroup
	siteTaskCh := make(chan siteTask)
	for i := 0; i < *concurrency; i++ {
		siteWG.Add(1)
		go func() {
			defer siteWG.Done()
			for t := range siteTaskCh {
				status, _, jobs, err := postCXS(t.tenant, t.instance, t.site)
				if err != nil || status != http.StatusOK {
					continue
				}
				siteResults <- siteResult{t.company, t.tenant, t.instance, t.site, jobs}
			}
		}()
	}
	go func() {
		for _, t := range siteTasks {
			siteTaskCh <- t
		}
		close(siteTaskCh)
	}()
	siteWG.Wait()
	close(siteResults)

	confirmedFull := map[string]siteResult{}
	for r := range siteResults {
		if _, ok := confirmedFull[r.company]; !ok {
			confirmedFull[r.company] = r
		}
	}

	sort.Strings(names)
	var fullCount, tenantOnlyCount, ambiguousCount, noneCount int
	for _, name := range names {
		if r, ok := confirmedFull[name]; ok {
			fmt.Printf("FULL    %-40s tenant=%-20s instance=%-8s site=%-24s %d jobs\n",
				name, r.tenant, r.instance, r.site, r.jobs)
			fullCount++
		} else if h, ok := confirmedTenant[name]; ok {
			fmt.Printf("TENANT  %-40s tenant=%-20s instance=%-8s (site unknown, needs manual lookup)\n",
				name, h.tenant, h.instance)
			tenantOnlyCount++
		} else if ambiguous[name] {
			fmt.Printf("AMBIG   %-40s (candidate tenant matched by another company too, skipped)\n", name)
			ambiguousCount++
		} else {
			fmt.Printf("NONE    %-40s\n", name)
			noneCount++
		}
	}
	fmt.Printf("\n%d fully confirmed, %d tenant-only (site unknown), %d ambiguous (skipped), %d not found (of %d)\n",
		fullCount, tenantOnlyCount, ambiguousCount, noneCount, len(names))

	if *fix && fullCount > 0 {
		existing, err := loadCompaniesYAML("companies.yaml")
		if err != nil {
			fmt.Fprintln(os.Stderr, "load companies.yaml:", err)
			os.Exit(1)
		}
		existingTenants := map[string]bool{}
		for _, c := range existing {
			existingTenants[strings.ToLower(c.Tenant)] = true
		}
		added := 0
		for _, name := range names {
			r, ok := confirmedFull[name]
			if !ok || existingTenants[strings.ToLower(r.tenant)] {
				continue
			}
			existing = append(existing, ycompany{Name: name, Tenant: r.tenant, Instance: r.instance, Site: r.site})
			added++
		}
		if err := writeCompaniesYAML("companies.yaml", existing); err != nil {
			fmt.Fprintln(os.Stderr, "write companies.yaml:", err)
			os.Exit(1)
		}
		fmt.Printf("added %d companies, wrote %d entries to companies.yaml\n", added, len(existing))
	}
}
