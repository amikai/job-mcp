# Greenhouse CLI Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add `cmd/greenhouse`, a standalone CLI (`companies` / `search` / `get`) over the existing Greenhouse provider, plus the `companies.go` roster loader it needs.

**Architecture:** Mirrors `cmd/ashby`'s ff/v4 command-tree skeleton, but with Greenhouse's request strategy: `search` calls the lightweight `listJobs` (no `content=true`) and filters client-side; `get` calls the native single-job endpoint `getJob` with `pay_transparency=true`. Roster lives in the provider package as embedded YAML, exported as plain vars.

**Tech Stack:** Go, `peterbourgon/ff/v4` (flags), `goccy/go-yaml` (roster), `jaytaylor/html2text` + stdlib `html` (JD rendering), `stretchr/testify` (tests). All already in go.mod — no new dependencies.

**Spec:** `docs/superpowers/specs/2026-07-06-greenhouse-cli-design.md`

## Global Constraints

- No new go.mod dependencies.
- Export package vars directly (`Companies`, `CompaniesByBoard`) — no wrapper getters (user preference).
- API base URL: `https://boards-api.greenhouse.io/v1` (the single server in the provider's openapi.yaml).
- `search` must NOT pass `content=true`; `get` must pass `pay_transparency=true` and must NOT pass `questions`.
- Filters are case-insensitive substring matches; `--keyword` on title, `--location` on location name; both given → AND.
- Text output for `search` entries: numbered title, then Location / Posted / URL / ID lines. Text output for `get`: title, Company / Location / Posted / URL lines, pay ranges, description. JSON for search: `{"total": <pre-filter count>, "jobs": [...]}`; JSON for get: the full generated `JobDetail`.
- Pay range line: `title: min – max CURRENCY` (cents→units, currency from `currency_type` verbatim, no hard-coded `$`).
- Description: `html.UnescapeString` first (content is entity-encoded HTML), then `html2text.FromString`; on conversion error print the decoded string, don't swallow.
- Commit messages follow repo convention: `feat(greenhouse): ...` / `feat(greenhouse-cli): ...`, and end with the `Co-Authored-By: Claude Fable 5 <noreply@anthropic.com>` trailer.
- Run `gofmt` and `go vet` before each commit.

---

### Task 1: Roster loader `companies.go`

**Files:**
- Create: `internal/provider/greenhouse/companies.go`
- (Already exists, do not touch: `internal/provider/greenhouse/companies.yaml` — 62 verified entries, keys `company` + `board`)

**Interfaces:**
- Consumes: `internal/provider/greenhouse/companies.yaml` (embedded).
- Produces: `greenhouse.Company{Name string; Board string}`, `greenhouse.Companies []Company` (sorted by Name), `greenhouse.CompaniesByBoard map[string]Company` (keys lowercased). Later tasks import these for `--board` validation and the `companies` subcommand.

Note: the repo deliberately dropped roster unit tests for lever and ashby (commits 0d916e6, 8eefed6) — do NOT add a `companies_test.go`. The verification cycle here is compile + existing provider tests.

- [ ] **Step 1: Write `companies.go`**

Mirror `internal/provider/ashby/companies.go` (drop its `BoardURL` helper — YAGNI; Greenhouse responses carry `absolute_url`):

```go
package greenhouse

import (
	_ "embed"
	"fmt"
	"sort"
	"strings"

	"github.com/goccy/go-yaml"
)

//go:embed companies.yaml
var companiesYAML []byte

// Company is a confirmed organization hosting a public Greenhouse job
// board, drawn from a curated list
// (internal/provider/greenhouse/companies.yaml). Every entry was verified
// against the live Job Board API — HTTP 200 with a non-empty jobs array.
// It's keyed by board token (e.g. "stripe"), the same identifier the API
// takes as its board_token path parameter.
type Company struct {
	Name  string `yaml:"company" json:"company"`
	Board string `yaml:"board" json:"board"`
}

// Companies holds every confirmed Greenhouse board, sorted by company name.
var Companies = mustLoadCompanies()

// CompaniesByBoard looks up a confirmed company by board token. Keys are
// lowercased, so callers must lowercase their input before indexing.
var CompaniesByBoard = buildBoardIndex(Companies)

// mustLoadCompanies parses the embedded companies.yaml. A parse failure is
// a build-time bug in a file this package owns, not a runtime condition to
// recover from.
func mustLoadCompanies() []Company {
	var cs []Company
	if err := yaml.Unmarshal(companiesYAML, &cs); err != nil {
		panic(fmt.Sprintf("greenhouse: parse companies.yaml: %v", err))
	}
	sort.Slice(cs, func(i, j int) bool { return cs[i].Name < cs[j].Name })
	return cs
}

func buildBoardIndex(cs []Company) map[string]Company {
	m := make(map[string]Company, len(cs))
	for _, c := range cs {
		m[strings.ToLower(c.Board)] = c
	}
	return m
}
```

- [ ] **Step 2: Verify it compiles and existing provider tests still pass**

Run: `go build ./... && go test ./internal/provider/greenhouse/`
Expected: build OK, existing client tests PASS.

- [ ] **Step 3: Format, vet, commit**

```bash
gofmt -l internal/provider/greenhouse/companies.go   # expect no output
go vet ./internal/provider/greenhouse/
git add internal/provider/greenhouse/companies.go internal/provider/greenhouse/companies.yaml
git commit -m "feat(greenhouse): add curated company board roster

Co-Authored-By: Claude Fable 5 <noreply@anthropic.com>"
```

(companies.yaml is currently untracked — this commit picks it up too.)

---

### Task 2: CLI skeleton + pure helpers (TDD)

**Files:**
- Create: `cmd/greenhouse/main.go` (stub `main`, projection + rendering helpers)
- Create: `cmd/greenhouse/main_test.go`

**Interfaces:**
- Consumes: generated types `greenhouse.JobSummary`, `greenhouse.PayInputRange` (fields are ogen Opt wrappers: `ID OptInt`, `Title OptString`, `Location OptLocation{Name OptString}`, `AbsoluteURL OptURI`, `FirstPublished/UpdatedAt OptDateTime`; `PayInputRange{MinCents, MaxCents OptInt; CurrencyType, Title, Blurb OptString}`).
- Produces (used by Task 3):
  - `type jobSummaryJSON struct` with fields `ID int`, `Title, Location, PostedAt, UpdatedAt, URL string`
  - `func summarize(j greenhouse.JobSummary) jobSummaryJSON`
  - `func matches(s jobSummaryJSON, keyword, location string) bool`
  - `func formatCents(cents int) string`
  - `func payRangeLine(r greenhouse.PayInputRange) string`
  - `func renderDescription(content string) string`
  - `func printSummary(s jobSummaryJSON)` (text lines below the title)

- [ ] **Step 1: Write the failing tests**

`cmd/greenhouse/main_test.go`:

```go
package main

import (
	"net/url"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	greenhouse "github.com/amikai/openings-mcp/internal/provider/greenhouse"
)

func mustURL(t *testing.T, s string) url.URL {
	t.Helper()
	u, err := url.Parse(s)
	assert.NoError(t, err)
	return *u
}

func TestSummarize(t *testing.T) {
	j := greenhouse.JobSummary{
		ID:             greenhouse.NewOptInt(4425455),
		Title:          greenhouse.NewOptString("Staff Engineer"),
		FirstPublished: greenhouse.NewOptDateTime(time.Date(2026, 5, 1, 9, 0, 0, 0, time.UTC)),
		UpdatedAt:      greenhouse.NewOptDateTime(time.Date(2026, 6, 20, 9, 0, 0, 0, time.UTC)),
		Location:       greenhouse.NewOptLocation(greenhouse.Location{Name: greenhouse.NewOptString("Taipei, Taiwan")}),
		AbsoluteURL:    greenhouse.NewOptURI(mustURL(t, "https://boards.greenhouse.io/acme/jobs/4425455")),
	}
	assert.Equal(t, jobSummaryJSON{
		ID:        4425455,
		Title:     "Staff Engineer",
		Location:  "Taipei, Taiwan",
		PostedAt:  "2026-05-01",
		UpdatedAt: "2026-06-20",
		URL:       "https://boards.greenhouse.io/acme/jobs/4425455",
	}, summarize(j))
}

func TestSummarizeEmptyOptionals(t *testing.T) {
	s := summarize(greenhouse.JobSummary{ID: greenhouse.NewOptInt(1), Title: greenhouse.NewOptString("X")})
	assert.Equal(t, jobSummaryJSON{ID: 1, Title: "X"}, s)
}

func TestMatches(t *testing.T) {
	s := jobSummaryJSON{Title: "Senior Software Engineer", Location: "Taipei, Taiwan"}
	assert.True(t, matches(s, "", ""), "empty filters match everything")
	assert.True(t, matches(s, "software", ""), "keyword is case-insensitive substring on title")
	assert.True(t, matches(s, "", "taipei"), "location is case-insensitive substring")
	assert.True(t, matches(s, "engineer", "taiwan"), "both filters AND together")
	assert.False(t, matches(s, "manager", ""))
	assert.False(t, matches(s, "engineer", "london"), "one failing filter fails the AND")
}

func TestFormatCents(t *testing.T) {
	assert.Equal(t, "136000", formatCents(13600000), "whole units drop the decimals")
	assert.Equal(t, "1359.99", formatCents(135999), "fractional cents keep two decimals")
}

func TestPayRangeLine(t *testing.T) {
	r := greenhouse.PayInputRange{
		MinCents:     greenhouse.NewOptInt(13600000),
		MaxCents:     greenhouse.NewOptInt(20000000),
		CurrencyType: greenhouse.NewOptString("USD"),
		Title:        greenhouse.NewOptString("Base Salary"),
	}
	assert.Equal(t, "Base Salary: 136000 – 200000 USD", payRangeLine(r))

	untitled := greenhouse.PayInputRange{
		MinCents:     greenhouse.NewOptInt(5000000),
		MaxCents:     greenhouse.NewOptInt(7000000),
		CurrencyType: greenhouse.NewOptString("EUR"),
	}
	assert.Equal(t, "50000 – 70000 EUR", payRangeLine(untitled))
}

func TestRenderDescription(t *testing.T) {
	// Greenhouse sends entity-encoded HTML: decode first, then strip tags.
	got := renderDescription("&lt;p&gt;Build &amp;amp; ship things.&lt;/p&gt;")
	assert.Contains(t, got, "Build & ship things.")
	assert.NotContains(t, got, "<p>")
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./cmd/greenhouse/`
Expected: FAIL to build — `undefined: summarize`, `undefined: jobSummaryJSON`, etc.

- [ ] **Step 3: Write the skeleton + helpers**

`cmd/greenhouse/main.go`:

```go
package main

import (
	"fmt"
	"html"
	"strings"

	"github.com/jaytaylor/html2text"

	greenhouse "github.com/amikai/openings-mcp/internal/provider/greenhouse"
)

// apiBaseURL is Greenhouse's public Job Board API origin — the single
// production server in the provider's openapi.yaml.
const apiBaseURL = "https://boards-api.greenhouse.io/v1"

func main() {
	// Command tree wired in a later task.
}

// jobSummaryJSON is the --format json shape for one search result: the
// compact fields a listing needs, no description. It's a flat, stable
// projection of the generated greenhouse.JobSummary so the CLI's output
// doesn't change shape when the spec's generated types do.
type jobSummaryJSON struct {
	ID        int    `json:"id"`
	Title     string `json:"title"`
	Location  string `json:"location,omitempty"`
	PostedAt  string `json:"postedAt,omitempty"`
	UpdatedAt string `json:"updatedAt,omitempty"`
	URL       string `json:"url,omitempty"`
}

type searchResultJSON struct {
	Total int              `json:"total"`
	Jobs  []jobSummaryJSON `json:"jobs"`
}

func summarize(j greenhouse.JobSummary) jobSummaryJSON {
	s := jobSummaryJSON{
		ID:       j.ID.Value,
		Title:    j.Title.Value,
		Location: j.Location.Value.Name.Value,
	}
	if j.AbsoluteURL.Set {
		s.URL = j.AbsoluteURL.Value.String()
	}
	if j.FirstPublished.Set {
		s.PostedAt = j.FirstPublished.Value.Format("2006-01-02")
	}
	if j.UpdatedAt.Set {
		s.UpdatedAt = j.UpdatedAt.Value.Format("2006-01-02")
	}
	return s
}

// matches applies the client-side search filters: case-insensitive
// substring on title (keyword) and location name (location), ANDed. The
// Job Board API has no server-side filtering, so this is the whole search.
func matches(s jobSummaryJSON, keyword, location string) bool {
	return containsFold(s.Title, keyword) && containsFold(s.Location, location)
}

func containsFold(s, sub string) bool {
	if sub == "" {
		return true
	}
	return strings.Contains(strings.ToLower(s), strings.ToLower(sub))
}

// formatCents renders a pay_input_ranges amount: whole currency units when
// the cents divide evenly (the common case), two decimals otherwise.
func formatCents(cents int) string {
	if cents%100 == 0 {
		return fmt.Sprintf("%d", cents/100)
	}
	return fmt.Sprintf("%.2f", float64(cents)/100)
}

// payRangeLine renders one pay range as "title: min – max CURRENCY". The
// currency comes from currency_type verbatim — no hard-coded "$", the
// roster has EUR boards.
func payRangeLine(r greenhouse.PayInputRange) string {
	span := fmt.Sprintf("%s – %s %s",
		formatCents(r.MinCents.Value), formatCents(r.MaxCents.Value), r.CurrencyType.Value)
	if t := r.Title.Value; t != "" {
		return t + ": " + span
	}
	return span
}

// renderDescription converts a job's content field to plain text. Greenhouse
// sends it HTML entity-encoded, so decode first, then strip tags; on a
// conversion failure fall back to the decoded HTML rather than dropping it.
func renderDescription(content string) string {
	decoded := html.UnescapeString(content)
	if text, err := html2text.FromString(decoded, html2text.Options{}); err == nil {
		return text
	}
	return decoded
}

// printSummary prints one job's compact text block (everything below the
// title line).
func printSummary(s jobSummaryJSON) {
	if s.Location != "" {
		fmt.Printf("Location: %s\n", s.Location)
	}
	if s.PostedAt != "" {
		fmt.Printf("Posted: %s\n", s.PostedAt)
	}
	if s.URL != "" {
		fmt.Printf("URL: %s\n", s.URL)
	}
	fmt.Printf("ID: %d\n", s.ID)
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./cmd/greenhouse/`
Expected: PASS (6 tests). (The `NewOpt*` constructors used in the tests are all verified to exist in `oas_schemas_gen.go`, including `NewOptURI(v url.URL)` and `NewOptLocation(v Location)`.)

- [ ] **Step 5: Format, vet, commit**

```bash
gofmt -l cmd/greenhouse   # expect no output
go vet ./cmd/greenhouse/
git add cmd/greenhouse/
git commit -m "feat(greenhouse-cli): add summary projection and rendering helpers

Co-Authored-By: Claude Fable 5 <noreply@anthropic.com>"
```

---

### Task 3: run functions (companies / search / get)

**Files:**
- Modify: `cmd/greenhouse/main.go` (append run functions)
- Modify: `cmd/greenhouse/main_test.go` (append error-path tests)

**Interfaces:**
- Consumes (from Task 1): `greenhouse.Companies`, `greenhouse.CompaniesByBoard`. (From Task 2): `summarize`, `matches`, `payRangeLine`, `renderDescription`, `printSummary`, `searchResultJSON`, `apiBaseURL`. (Generated client): `greenhouse.NewClient(serverURL string, opts ...ClientOption)`, `client.ListJobs(ctx, greenhouse.ListJobsParams{BoardToken})` → `ListJobsRes` (`*JobListResponse` | `*ListJobsNotFound`), `client.GetJob(ctx, greenhouse.GetJobParams{BoardToken, JobID, PayTransparency})` → `GetJobRes` (`*JobDetail` | `*GetJobNotFound`).
- Produces (used by Task 4's command tree):
  - `func runCompanies(format string) error`
  - `func runSearch(ctx context.Context, board string, timeout time.Duration, keyword, location, format string) error`
  - `func runGet(ctx context.Context, board string, timeout time.Duration, jobID int, format string) error`

- [ ] **Step 1: Write the failing error-path tests**

Append to `cmd/greenhouse/main_test.go` (same style as `cmd/ashby/main_test.go`; none of these touch the network — the guard fires before any client call):

```go
func TestRunSearchMissingBoard(t *testing.T) {
	err := runSearch(context.Background(), "", time.Second, "", "", "text")
	assert.ErrorContains(t, err, "--board is required")
}

func TestRunSearchUnknownBoard(t *testing.T) {
	err := runSearch(context.Background(), "doesnotexist-board-xyz", time.Second, "", "", "text")
	assert.ErrorContains(t, err, `board "doesnotexist-board-xyz" not found`)
	assert.ErrorContains(t, err, "greenhouse companies")
}

func TestRunGetMissingID(t *testing.T) {
	err := runGet(context.Background(), "anthropic", time.Second, 0, "text")
	assert.ErrorContains(t, err, "--id is required")
}

func TestRunGetMissingBoard(t *testing.T) {
	err := runGet(context.Background(), "", time.Second, 123, "text")
	assert.ErrorContains(t, err, "--board is required")
}

func TestRunGetUnknownBoard(t *testing.T) {
	err := runGet(context.Background(), "doesnotexist-board-xyz", time.Second, 123, "text")
	assert.ErrorContains(t, err, `board "doesnotexist-board-xyz" not found`)
	assert.ErrorContains(t, err, "greenhouse companies")
}
```

Add `"context"` and `"time"` to the test file's imports.

- [ ] **Step 2: Run tests to verify the new ones fail**

Run: `go test ./cmd/greenhouse/`
Expected: FAIL to build — `undefined: runSearch`, `undefined: runGet`.

- [ ] **Step 3: Implement the run functions**

Append to `cmd/greenhouse/main.go` (add `"context"`, `"encoding/json"`, `"os"`, `"time"` to imports):

```go
// runCompanies lists every confirmed Greenhouse board embedded in the CLI
// (internal/provider/greenhouse/companies.yaml), sorted by company name. It
// makes no network call.
func runCompanies(format string) error {
	cs := greenhouse.Companies

	if format == "json" {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(cs)
	}

	for _, c := range cs {
		fmt.Printf("%s (%s)\n", c.Name, c.Board)
	}
	return nil
}

// normalizeBoard lowercases the --board value and requires it to be a
// curated board — same policy as cmd/ashby's fetchBoard front half.
func normalizeBoard(board string) (string, error) {
	if board == "" {
		return "", fmt.Errorf("--board is required")
	}
	slug := strings.ToLower(board)
	if _, ok := greenhouse.CompaniesByBoard[slug]; !ok {
		return "", fmt.Errorf("board %q not found; run 'greenhouse companies' to see supported boards", board)
	}
	return slug, nil
}

// runSearch fetches the board's whole job list (the API has no pagination
// and no server-side filters) WITHOUT content=true — summaries stay small —
// then filters client-side and prints summaries.
func runSearch(ctx context.Context, board string, timeout time.Duration, keyword, location, format string) error {
	slug, err := normalizeBoard(board)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	client, err := greenhouse.NewClient(apiBaseURL)
	if err != nil {
		return err
	}

	res, err := client.ListJobs(ctx, greenhouse.ListJobsParams{BoardToken: slug})
	if err != nil {
		return err
	}
	var resp *greenhouse.JobListResponse
	switch r := res.(type) {
	case *greenhouse.JobListResponse:
		resp = r
	case *greenhouse.ListJobsNotFound:
		// Theoretically unreachable for roster boards, but reported
		// rather than swallowed.
		return fmt.Errorf("board %q not found upstream", board)
	default:
		return fmt.Errorf("unexpected response type %T", res)
	}

	matched := make([]jobSummaryJSON, 0, len(resp.Jobs))
	for _, j := range resp.Jobs {
		s := summarize(j)
		if matches(s, keyword, location) {
			matched = append(matched, s)
		}
	}

	if format == "json" {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(searchResultJSON{Total: len(resp.Jobs), Jobs: matched})
	}

	fmt.Printf("Greenhouse Jobs Report (board: %s)\n", slug)
	fmt.Printf("Found %d jobs; showing %d\n\n", len(resp.Jobs), len(matched))
	for i, s := range matched {
		fmt.Printf("%d. %s\n", i+1, s.Title)
		printSummary(s)
		fmt.Println()
	}
	return nil
}

// runGet fetches one job in full via Greenhouse's single-job endpoint —
// unlike Ashby there's no need to re-fetch the whole board — with
// pay_transparency=true so pay_input_ranges come back.
func runGet(ctx context.Context, board string, timeout time.Duration, jobID int, format string) error {
	if jobID == 0 {
		return fmt.Errorf("--id is required (take it from a search result's ID)")
	}
	slug, err := normalizeBoard(board)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	client, err := greenhouse.NewClient(apiBaseURL)
	if err != nil {
		return err
	}

	res, err := client.GetJob(ctx, greenhouse.GetJobParams{
		BoardToken:      slug,
		JobID:           jobID,
		PayTransparency: greenhouse.NewOptBool(true),
	})
	if err != nil {
		return err
	}
	switch r := res.(type) {
	case *greenhouse.JobDetail:
		return printDetail(r, format)
	case *greenhouse.GetJobNotFound:
		return fmt.Errorf("job %d not found on board %q", jobID, board)
	default:
		return fmt.Errorf("unexpected response type %T", res)
	}
}

// printDetail renders one full job. JSON mode encodes the generated
// JobDetail as-is — detail is for seeing the whole record.
func printDetail(d *greenhouse.JobDetail, format string) error {
	if format == "json" {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(d)
	}

	fmt.Println(d.Title.Value)
	if d.CompanyName.Value != "" {
		fmt.Printf("Company: %s\n", d.CompanyName.Value)
	}
	if name := d.Location.Value.Name.Value; name != "" {
		fmt.Printf("Location: %s\n", name)
	}
	if d.FirstPublished.Set {
		fmt.Printf("Posted: %s\n", d.FirstPublished.Value.Format("2006-01-02"))
	}
	if d.AbsoluteURL.Set {
		fmt.Printf("URL: %s\n", d.AbsoluteURL.Value.String())
	}
	if len(d.PayInputRanges) > 0 {
		fmt.Println("Pay ranges:")
		for _, r := range d.PayInputRanges {
			fmt.Printf("  %s\n", payRangeLine(r))
			if b := r.Blurb.Value; b != "" {
				fmt.Printf("    %s\n", b)
			}
		}
	}
	if d.Content.Value != "" {
		fmt.Printf("\nDescription:\n%s\n", renderDescription(d.Content.Value))
	}
	return nil
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./cmd/greenhouse/`
Expected: PASS (11 tests).

- [ ] **Step 5: Format, vet, commit**

```bash
gofmt -l cmd/greenhouse   # expect no output
go vet ./cmd/greenhouse/
git add cmd/greenhouse/
git commit -m "feat(greenhouse-cli): add companies, search, and get run functions

Co-Authored-By: Claude Fable 5 <noreply@anthropic.com>"
```

---

### Task 4: ff command tree + end-to-end smoke test

**Files:**
- Modify: `cmd/greenhouse/main.go` (replace the stub `main`)

**Interfaces:**
- Consumes: `runCompanies`, `runSearch`, `runGet` from Task 3.
- Produces: the finished `greenhouse` binary. Nothing downstream.

- [ ] **Step 1: Replace the stub `main` with the command tree**

Same skeleton as `cmd/ashby/main.go` (add `"errors"`, `"github.com/peterbourgon/ff/v4"`, `"github.com/peterbourgon/ff/v4/ffhelp"` to imports):

```go
func main() {
	rootFlags := ff.NewFlagSet("greenhouse")
	var (
		board   = rootFlags.StringLong("board", "", "confirmed Greenhouse board token, e.g. stripe (see 'greenhouse companies' for the full list)")
		timeout = rootFlags.DurationLong("timeout", 60*time.Second, "request timeout")
		format  = rootFlags.StringEnumLong("format", "output format", "text", "json")
	)
	rootCmd := &ff.Command{
		Name:  "greenhouse",
		Usage: "greenhouse --board BOARD [FLAGS] <companies|search|get> [FLAGS]",
		Flags: rootFlags,
	}

	companiesFlags := ff.NewFlagSet("companies").SetParent(rootFlags)
	companiesCmd := &ff.Command{
		Name:      "companies",
		Usage:     "greenhouse companies [--format text|json]",
		ShortHelp: "list confirmed Greenhouse boards (company name and board token)",
		Flags:     companiesFlags,
		Exec: func(ctx context.Context, args []string) error {
			return runCompanies(*format)
		},
	}
	rootCmd.Subcommands = append(rootCmd.Subcommands, companiesCmd)

	searchFlags := ff.NewFlagSet("search").SetParent(rootFlags)
	var (
		keyword  = searchFlags.StringLong("keyword", "", "case-insensitive substring filter on job titles (empty lists every job)")
		location = searchFlags.StringLong("location", "", "case-insensitive substring filter on location names")
	)
	searchCmd := &ff.Command{
		Name:      "search",
		Usage:     "greenhouse --board BOARD search [--keyword TEXT] [--location TEXT] [--format text|json]",
		ShortHelp: "list a board's jobs as summaries (client-side filters)",
		Flags:     searchFlags,
		Exec: func(ctx context.Context, args []string) error {
			return runSearch(ctx, *board, *timeout, *keyword, *location, *format)
		},
	}
	rootCmd.Subcommands = append(rootCmd.Subcommands, searchCmd)

	getFlags := ff.NewFlagSet("get").SetParent(rootFlags)
	jobID := getFlags.IntLong("id", 0, "job posting id from search results")
	getCmd := &ff.Command{
		Name:      "get",
		Usage:     "greenhouse --board BOARD get --id JOB-ID [--format text|json]",
		ShortHelp: "print one job in full (description and pay ranges)",
		Flags:     getFlags,
		Exec: func(ctx context.Context, args []string) error {
			return runGet(ctx, *board, *timeout, *jobID, *format)
		},
	}
	rootCmd.Subcommands = append(rootCmd.Subcommands, getCmd)

	if err := rootCmd.Parse(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, ffhelp.Command(rootCmd.GetSelected()))
		if errors.Is(err, ff.ErrHelp) {
			os.Exit(0)
		}
		fmt.Fprintln(os.Stderr, "err:", err)
		os.Exit(1)
	}

	if rootCmd.GetSelected() == rootCmd {
		fmt.Fprintln(os.Stderr, ffhelp.Command(rootCmd))
		fmt.Fprintln(os.Stderr, "err: a subcommand (companies, search, or get) is required")
		os.Exit(1)
	}

	if err := rootCmd.Run(context.Background()); err != nil {
		fmt.Fprintln(os.Stderr, "err:", err)
		os.Exit(1)
	}
}
```

- [ ] **Step 2: Build and run unit tests**

Run: `go build ./... && go test ./cmd/greenhouse/`
Expected: build OK, 11 tests PASS.

- [ ] **Step 3: Manual smoke test against the live API**

Use a small board (Speechmatics, 12 jobs) to keep output readable:

```bash
go run ./cmd/greenhouse companies | head -5
# Expected: 5 lines like "Airbnb (airbnb)", sorted by name

go run ./cmd/greenhouse --board speechmatics search
# Expected: "Found N jobs; showing N" and numbered summaries with Location/Posted/URL/ID

go run ./cmd/greenhouse --board speechmatics search --keyword engineer --location united
# Expected: only titles containing "engineer" AND locations containing "united"

go run ./cmd/greenhouse --board speechmatics search --format json | head -20
# Expected: {"total": N, "jobs": [...]}

# Take an ID from the search output above, then:
go run ./cmd/greenhouse --board speechmatics get --id <ID>
# Expected: title, Company/Location/Posted/URL, Description as plain text
#           (no HTML tags, no &lt; entities); "Pay ranges:" only if the
#           posting publishes them (try an Anthropic ID if not)

go run ./cmd/greenhouse --board nosuchboard search
# Expected: err: board "nosuchboard" not found; run 'greenhouse companies' ...
```

- [ ] **Step 4: Format, vet, commit**

```bash
gofmt -l cmd/greenhouse   # expect no output
go vet ./cmd/greenhouse/
git add cmd/greenhouse/main.go
git commit -m "feat(greenhouse): add standalone CLI (companies, search, get)

Co-Authored-By: Claude Fable 5 <noreply@anthropic.com>"
```
