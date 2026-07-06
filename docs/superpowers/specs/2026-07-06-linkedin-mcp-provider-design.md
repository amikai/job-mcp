# LinkedIn MCP Provider — Design

## Context

`internal/provider/linkedin` and the standalone `cmd/linkedin` CLI already exist
(merged in [#71](https://github.com/amikai/openings-mcp/pull/71)) and scrape
LinkedIn's unauthenticated guest job-search surface. They are not yet wired
into `openings-mcp`, the MCP server that exposes the other five providers
(104, Cake.me, Google, NVIDIA, TSMC) as tools. This design wires LinkedIn in
following the same pattern.

## Architecture

No new architectural shape: this mirrors how tsmc/nvidia/etc. are wired.

- New file `internal/openingsmcp/linkedin.go` exporting
  `RegisterLinkedin(s *mcp.Server, c *linkedin.Client)`, which registers two
  tools: `linkedin_search_jobs` and `linkedin_get_job_detail` (the standard
  search-then-detail flow already documented in `serverInstructions`).
- `cmd/openings-mcp/main.go`: construct one shared `linkedin.Client` at
  server startup and pass it into `newServer`/`RegisterLinkedin`. A single
  shared client (and its cookie jar) persists for the server's lifetime —
  LinkedIn's job-detail endpoint depends on cookies warmed by a prior
  request to the same host, exactly how `cmd/linkedin`'s own CLI already
  uses the client. Update the `serverInstructions` constant to mention six
  job boards instead of five.
- `internal/openingsmcp/linkedin_test.go`: mirrors `tsmc_test.go`, backed by
  `linkedin.NewMockServer()` (already exists in the provider package).

## Client construction

```go
jar, _ := cookiejar.New(nil)
cLinkedin := linkedin.NewClient("https://www.linkedin.com", &http.Client{
    Timeout: 30 * time.Second,
    Jar:     jar,
})
```

An explicit jar is constructed at the call site (rather than relying on
`linkedin.NewClient`'s nil-fallback) so both the timeout and cookie
persistence are visible in `main.go`, matching the 30s timeout convention
used for the other providers there.

## `linkedin_search_jobs`

### Input schema

Hand-written raw JSON schema (matching the tsmc/nvidia style: human-readable
label enums instead of the site's raw form-field codes, converted via the
provider's existing `WorkplaceTypeIDs`/`JobTypeIDs` lookup tables).

| Field | Type | Required | Notes |
|---|---|---|---|
| `keyword` | string | no | free-text; maps to `Keywords` |
| `location` | string | no | free-text; LinkedIn has no structured location field |
| `workplace_type` | enum: `On-site`, `Remote`, `Hybrid` | no | maps via `linkedin.WorkplaceTypeIDs` |
| `job_type` | enum: `Full-time`, `Part-time`, `Contract`, `Temporary`, `Internship` | no | maps via `linkedin.JobTypeIDs` |
| `company_ids` | array of string | no | opaque numeric LinkedIn company IDs, passed through raw (not resolvable by a host LLM without a prior lookup — documented as such; search summaries don't carry them) |
| `posted_within` | enum: `Past day`, `Past week`, `Past month` | no | maps to 86400 / 604800 / 2592000 seconds |
| `start` | integer, default 0 | no | raw zero-based result offset (not abstracted into a page number — LinkedIn's own `start` semantics are exposed directly). The endpoint always returns exactly 10 cards regardless of `start`, so the description instructs paging in steps of 10; a fresh MCP session starts at 0, not the browser-mimicking `linkedin.DefaultStart` (25) that `cmd/linkedin` defaults to. |

Two provider parameters are deliberately not exposed:

- `distance` — its live behavior was never verified (a quick browser check
  found a negative value silently swaps in unrelated results instead of
  erroring); the field was removed from `JobsRequest` entirely, and
  openapi.yaml documents the parameter as real-but-unimplemented.
- `easy_apply` — `JobsRequest.EasyApply` stays for programmatic use, but
  neither the MCP schema nor the CLI exposes it.

No field is `required`, unlike tsmc — LinkedIn's search accepts an entirely
empty query.

Tool descriptions stay terse: they state that LinkedIn rate-limits
aggressively and to back off rather than retry on a 429, but quantitative
detail (the ~10-request session budget, the missing `Retry-After`) lives in
the client's 429/999 error strings instead — the LLM pays for that guidance
only when a block actually happens, and error-time is also when it acts on
it.

### Output

```go
type linkedinSearchOutput struct {
    Data []linkedinJobSummary `json:"data"`
}

type linkedinJobSummary struct {
    ID         string `json:"id" jsonschema:"Numeric LinkedIn job ID; pass to linkedin_get_job_detail's job_id param."`
    Title      string `json:"title"`
    Company    string `json:"company,omitempty"`
    CompanyURL string `json:"company_url,omitempty"`
    Location   string `json:"location,omitempty"`
    PostedDate string `json:"posted_date,omitempty"`
    Remote     bool   `json:"remote,omitempty"`
    URL        string `json:"url,omitempty" jsonschema:"Public job posting URL."`
}
```

`remote` keeps the provider's own `Job.Remote` field name (and matches the
`remote` field the google/104 outputs use) rather than the `looks_remote`
name an earlier draft of this design proposed. It is still a keyword
heuristic over title/location, not a field LinkedIn provides; the detail
output's `remote` carries that caveat in its jsonschema description.

No `total` field: unlike the other providers, LinkedIn's search response
never reports a result count — only ever the current page's up-to-10 cards.
`URL` is synthesized via a `linkedinJobURL(id)` helper that hardcodes
`https://www.linkedin.com/jobs/view/%s`, the same way `tsmcJobURL` hardcodes
TSMC's careers-site URL rather than threading a base URL through from the
client (whose `baseURL` field is unexported) — so the host doesn't have to
build the link itself.

## `linkedin_get_job_detail`

### Input

```go
type linkedinDetailInput struct {
    JobID string `json:"job_id" jsonschema:"Numeric LinkedIn job ID (id from search results, e.g. 4422697744)."`
}
```

### Output

```go
type linkedinDetailOutput struct {
    ID             string `json:"id"`
    URL            string `json:"url,omitempty" jsonschema:"Public job posting URL."`
    Title          string `json:"title"`
    Company        string `json:"company,omitempty"`
    Location       string `json:"location,omitempty"`
    Posted         string `json:"posted,omitempty" jsonschema:"Relative time, e.g. '1 month ago'; LinkedIn doesn't expose an exact date."`
    SeniorityLevel string `json:"seniority_level,omitempty"`
    EmploymentType string `json:"employment_type,omitempty"`
    JobFunction    string `json:"job_function,omitempty"`
    Industries     string `json:"industries,omitempty"`
    Description    string `json:"description,omitempty" jsonschema:"Full job description as plain text."`
    ApplyURL       string `json:"apply_url,omitempty" jsonschema:"External ATS apply URL."`
    Remote         bool   `json:"remote,omitempty" jsonschema:"Keyword heuristic over title/location only (not the full description), not a field LinkedIn provides. False does not mean confirmed on-site."`
}
```

`CompanyLogo` from the client's `JobDetailResponse` is dropped — it's an
image URL with no use to a text-oriented MCP host, and no other provider's
detail output carries a logo field.

## Error handling

`linkedin.Client` is hand-written (unlike nvidia's openapi-generated client),
so there's no typed error to distinguish upstream status codes. Its errors
are descriptive strings that pass straight through `errorResult(err)` — the
same treatment tsmc and google (also hand-written clients) already get.
Because the error string is the only guidance channel the host LLM has at
failure time, the 429 and 999 messages carry their own remedy: the 429 says
immediate retries keep failing (LinkedIn sends no `Retry-After`) and to back
off, and the 999 says one retry may pass now that the session carries
cookies (`warmSession` has already primed the jar by then) but to stop if it
recurs.

## Conversion functions

- `linkedinMCPToHTTPRequest(in *linkedinSearchInput) (*linkedin.JobsRequest, error)`
  — validates `workplace_type`/`job_type`/`posted_within` against their
  lookup tables (unknown label → error); `company_ids` passes through as-is
  (both sides are `[]string`).
- `linkedinHTTPToMCPResponse(resp *linkedin.JobsResponse) *linkedinSearchOutput`
  — synthesizes `URL` per job via `linkedinJobURL(id)`.
- `linkedinHTTPToMCPDetail(detail *linkedin.JobDetailResponse) *linkedinDetailOutput`

## Testing

- `TestRegisterLinkedin`: tool registration (`assertTools`), golden
  input-schema assertion — same shape as `TestRegisterTsmc`.
- `TestLinkedinSearchJobsE2E` / `TestLinkedinGetJobDetailE2E`: against
  `linkedin.NewMockServer()`, same shape as the tsmc E2E tests.
- `TestLinkedinMCPToHTTPRequest`: unit coverage for the conversion function,
  including invalid `workplace_type`/`job_type`/`posted_within` labels
  producing errors — mirrors `nvidiaMCPToHTTPRequest`'s test coverage.

## Out of scope

- No client-side self-throttling/pacing added to `linkedin.Client` itself —
  rate-limit avoidance is communicated to the host via tool descriptions and
  the 429/999 error strings, consistent with how the other providers surface
  upstream errors reactively rather than pre-emptively.
- Provider changes are limited to what the MCP wiring surfaced: dropping the
  unverified `Distance` field (and `--distance`/`--easy-apply` CLI flags)
  and making the 429/999 error strings self-explanatory for an LLM caller.
  The scraping/parsing logic itself is untouched.
