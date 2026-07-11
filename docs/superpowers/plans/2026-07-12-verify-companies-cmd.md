# verify-companies cmd Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** A standalone CLI `cmd/verify-companies` that verifies every curated companies.yaml entry by running a real search through the unified `internal/ats` adapters and reports each entry's total job count.

**Architecture:** One `main.go`. Adapters are constructed with the same base URLs as `cmd/openings-mcp/main.go`; rosters come from `Adapter.Roster()`, verification calls `Adapter.Search(slug, page 1)`, a bounded worker pool fans out, and results print as text or JSON with exit code 1 (any INVALID) / 2 (no INVALID but any ERROR) / 0.

**Tech Stack:** Go 1.26, `ff/v4` (repo's CLI convention), `internal/ats`, ogen runtime (`validate`) for error classification.

Spec: `docs/superpowers/specs/2026-07-12-verify-companies-cmd-design.md`

## Global Constraints

- No test file — per user decision, `cmd/verify-companies/main.go` only.
- Do NOT git commit — the user commits manually (standing preference overrides this skill's commit steps).
- Classification: Search success → OK with `TotalCount`; upstream 404 (all providers) or 422 (workday) → INVALID; anything else → ERROR.
- The cmd imports `internal/ats` but no `internal/provider/*` package. Typed status-code errors are matched via `interface{ error; GetStatusCode() int }` plus ogen's `*validate.UnexpectedStatusCodeError`; ashby/greenhouse typed 404s arrive as errors containing `"not found upstream"`.

---

### Task 1: cmd/verify-companies/main.go

**Files:**
- Create: `cmd/verify-companies/main.go`

**Interfaces:**
- Consumes: `ats.NewLeverAdapter(baseURL, *http.Client)`, `ats.NewAshbyAdapter(baseURL, *http.Client)`, `ats.NewGreenhouseAdapter(baseURL, *http.Client)` (all return `(*XxxAdapter, error)`), `ats.NewWorkdayAdapter(*http.Client)`; `ats.Adapter` (`Name() string`, `Roster() []ats.CompanyInfo`, `Search(ctx, slug string, ats.SearchParams) (*ats.SearchResult, error)`); `ats.CompanyInfo{Slug, Name string}`; `ats.SearchResult.TotalCount int`; `validate.UnexpectedStatusCodeError` from `github.com/ogen-go/ogen/validate`.
- Produces: the `verify-companies` binary; nothing consumes it.

- [ ] **Step 1: Write `cmd/verify-companies/main.go`**

```go
// Command verify-companies verifies every curated companies.yaml entry by
// running a real search through the unified internal/ats adapters — the
// same code path the MCP server serves — and reports each entry's total
// job count. See
// docs/superpowers/specs/2026-07-12-verify-companies-cmd-design.md.
package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"slices"
	"strconv"
	"strings"
	"sync"
	"text/tabwriter"
	"time"

	"github.com/ogen-go/ogen/validate"
	"github.com/peterbourgon/ff/v4"
	"github.com/peterbourgon/ff/v4/ffhelp"

	"github.com/amikai/openings-mcp/internal/ats"
)

// providerOrder fixes the --provider default and the report's grouping order.
var providerOrder = []string{"ashby", "greenhouse", "lever", "workday"}

// Result statuses. INVALID means upstream said the identifier is gone;
// ERROR means the check couldn't decide (timeout, 5xx, network or decode
// error).
const (
	statusOK      = "OK"
	statusInvalid = "INVALID"
	statusError   = "ERROR"
)

// check is one roster entry to verify against its adapter.
type check struct {
	adapter ats.Adapter
	company string
	slug    string
}

// result is one classified check: OK carries the company's total job
// count; INVALID and ERROR carry the error message in Detail.
type result struct {
	Provider string `json:"provider"`
	Company  string `json:"company"`
	Slug     string `json:"slug"`
	Status   string `json:"status"`
	Jobs     int    `json:"jobs"`
	Detail   string `json:"detail,omitempty"`
}

func main() {
	fs := ff.NewFlagSet("verify-companies")
	var (
		providers   = fs.StringLong("provider", strings.Join(providerOrder, ","), "comma-separated subset of ashby,greenhouse,lever,workday")
		timeout     = fs.DurationLong("timeout", 300*time.Second, "per-request timeout")
		concurrency = fs.IntLong("concurrency", 8, "number of concurrent checks")
		format      = fs.StringEnumLong("format", "output format", "text", "json")
	)

	var invalidCount, errorCount int
	cmd := &ff.Command{
		Name:  "verify-companies",
		Usage: "verify-companies [--provider LIST] [--timeout D] [--concurrency N] [--format text|json]",
		Flags: fs,
		Exec: func(ctx context.Context, args []string) error {
			var err error
			invalidCount, errorCount, err = run(ctx, *providers, *timeout, *concurrency, *format)
			return err
		},
	}

	if err := cmd.Parse(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, ffhelp.Command(cmd))
		if errors.Is(err, ff.ErrHelp) {
			os.Exit(0)
		}
		fmt.Fprintln(os.Stderr, "err:", err)
		os.Exit(1)
	}

	if err := cmd.Run(context.Background()); err != nil {
		fmt.Fprintln(os.Stderr, "err:", err)
		os.Exit(1)
	}
	switch {
	case invalidCount > 0:
		os.Exit(1)
	case errorCount > 0:
		os.Exit(2)
	}
}

func run(ctx context.Context, providerList string, timeout time.Duration, concurrency int, format string) (invalid, errs int, err error) {
	names, err := parseProviders(providerList)
	if err != nil {
		return 0, 0, err
	}
	if concurrency < 1 {
		return 0, 0, fmt.Errorf("--concurrency must be at least 1, got %d", concurrency)
	}

	adapters, err := buildAdapters(names)
	if err != nil {
		return 0, 0, err
	}
	results := runChecks(ctx, buildChecks(adapters), timeout, concurrency)

	if format == "json" {
		err = printJSON(results)
	} else {
		printText(results)
	}
	_, invalid, errs = tally(results)
	return invalid, errs, err
}

// parseProviders validates the --provider list and returns it in
// providerOrder so the report grouping is stable regardless of input order.
func parseProviders(list string) ([]string, error) {
	selected := map[string]bool{}
	for name := range strings.SplitSeq(list, ",") {
		name = strings.ToLower(strings.TrimSpace(name))
		if !slices.Contains(providerOrder, name) {
			return nil, fmt.Errorf("unknown provider %q (want any of %s)", name, strings.Join(providerOrder, ", "))
		}
		selected[name] = true
	}
	var names []string
	for _, name := range providerOrder {
		if selected[name] {
			names = append(names, name)
		}
	}
	return names, nil
}

// buildAdapters constructs the selected adapters with the same base URLs
// cmd/openings-mcp/main.go uses.
func buildAdapters(names []string) ([]ats.Adapter, error) {
	hc := &http.Client{}
	var adapters []ats.Adapter
	for _, name := range names {
		var (
			a   ats.Adapter
			err error
		)
		switch name {
		case "ashby":
			a, err = ats.NewAshbyAdapter("https://api.ashbyhq.com", hc)
		case "greenhouse":
			a, err = ats.NewGreenhouseAdapter("https://boards-api.greenhouse.io/v1", hc)
		case "lever":
			a, err = ats.NewLeverAdapter("https://api.lever.co", hc)
		case "workday":
			a = ats.NewWorkdayAdapter(hc)
		}
		if err != nil {
			return nil, fmt.Errorf("build %s adapter: %w", name, err)
		}
		adapters = append(adapters, a)
	}
	return adapters, nil
}

// buildChecks flattens the adapters' rosters into checks, in roster order.
func buildChecks(adapters []ats.Adapter) []check {
	var checks []check
	for _, a := range adapters {
		for _, c := range a.Roster() {
			checks = append(checks, check{adapter: a, company: c.Name, slug: c.Slug})
		}
	}
	return checks
}

// runChecks executes checks through a worker pool of size concurrency and
// returns results in check order.
func runChecks(ctx context.Context, checks []check, timeout time.Duration, concurrency int) []result {
	results := make([]result, len(checks))
	sem := make(chan struct{}, concurrency)
	var wg sync.WaitGroup
	for i, c := range checks {
		wg.Go(func() {
			sem <- struct{}{}
			defer func() { <-sem }()
			results[i] = c.do(ctx, timeout)
		})
	}
	wg.Wait()
	return results
}

// do searches page 1 for the entry and classifies the outcome.
func (c check) do(ctx context.Context, timeout time.Duration) result {
	r := result{Provider: c.adapter.Name(), Company: c.company, Slug: c.slug}

	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	res, err := c.adapter.Search(ctx, c.slug, ats.SearchParams{Page: 1})
	if err != nil {
		r.Detail = err.Error()
		if isGone(err, c.adapter.Name()) {
			r.Status = statusInvalid
		} else {
			r.Status = statusError
		}
		return r
	}
	r.Status = statusOK
	r.Jobs = res.TotalCount
	return r
}

// isGone reports whether err means the roster identifier no longer exists
// upstream: HTTP 404 for every provider, plus 422 for workday (its response
// to an unknown tenant). Lever and workday surface typed status-code errors
// (matched by GetStatusCode, with ogen's UnexpectedStatusCodeError as
// fallback); the ashby and greenhouse adapters translate their typed 404
// responses into "not found upstream" errors before any status code is
// visible.
func isGone(err error, provider string) bool {
	code := 0
	var scErr interface {
		error
		GetStatusCode() int
	}
	var unexpected *validate.UnexpectedStatusCodeError
	switch {
	case errors.As(err, &scErr):
		code = scErr.GetStatusCode()
	case errors.As(err, &unexpected):
		code = unexpected.StatusCode
	}
	if code == http.StatusNotFound {
		return true
	}
	if provider == "workday" && code == http.StatusUnprocessableEntity {
		return true
	}
	return strings.Contains(err.Error(), "not found upstream")
}

// printText writes one line per entry plus a summary. Jobs is shown only
// for OK entries; Detail only for non-OK entries, where it explains the
// classification.
func printText(results []result) {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	for _, r := range results {
		jobs, detail := "", r.Detail
		if r.Status == statusOK {
			jobs, detail = strconv.Itoa(r.Jobs), ""
		}
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\n", r.Status, r.Provider, r.Company, r.Slug, jobs, detail)
	}
	w.Flush()
	ok, invalid, errs := tally(results)
	fmt.Printf("\ntotal %d: ok %d, invalid %d, error %d\n", len(results), ok, invalid, errs)
}

func printJSON(results []result) error {
	ok, invalid, errs := tally(results)
	out := struct {
		Results []result       `json:"results"`
		Summary map[string]int `json:"summary"`
	}{
		Results: results,
		Summary: map[string]int{"ok": ok, "invalid": invalid, "error": errs},
	}
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(out)
}

func tally(results []result) (ok, invalid, errs int) {
	for _, r := range results {
		switch r.Status {
		case statusOK:
			ok++
		case statusInvalid:
			invalid++
		default:
			errs++
		}
	}
	return ok, invalid, errs
}
```

- [ ] **Step 2: Format, vet, build**

Run:
```bash
gofmt -l cmd/verify-companies && go vet ./cmd/verify-companies && go build ./cmd/verify-companies
```
Expected: no gofmt output, vet and build succeed.

- [ ] **Step 3: Smoke run against the smallest roster (lever, 20 entries)**

Run:
```bash
go run ./cmd/verify-companies --provider lever
```
Expected: 20 lines `OK  lever  <company>  <slug>  <jobs>` with non-zero job counts, summary `total 20: ok 20, invalid 0, error 0`, exit code 0 (`echo $?`).

- [ ] **Step 4: Verify JSON format and flag validation**

Run:
```bash
go run ./cmd/verify-companies --provider lever --format json | head -20
go run ./cmd/verify-companies --provider nope; echo "exit=$?"
```
Expected: JSON object with `results` (each result has a `jobs` field) and `summary`; second command prints `err: unknown provider "nope" ...` and `exit=1`.

- [ ] **Step 5: Full sweep across all four providers**

Run:
```bash
go build ./cmd/verify-companies && ./verify-companies; echo "exit=$?"
```
Expected: ~550 lines with job counts. Exit 0 if everything is live; exit 1 with INVALID lines if entries have gone stale (that output is the audit deliverable for issue #91); exit 2 if only transient ERRORs. Rerun once if ERRORs look transient.

- [ ] **Step 6: Do NOT commit**

Leave changes uncommitted; the user commits manually.

## Self-review notes

- Spec coverage: adapter-based verification, job counts, structure (ats-only imports), all four flags, text/JSON output, exit codes 0/1/2 — all in Task 1.
- No placeholders; single task because the spec is one self-contained file.
- Signatures checked against `internal/ats`: constructor arities, `Adapter` methods, `CompanyInfo{Slug, Name}`, `SearchResult.TotalCount`; `GetStatusCode()` exists on lever/workday `ErrorResponseStatusCode`.
