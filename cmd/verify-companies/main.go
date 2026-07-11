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
