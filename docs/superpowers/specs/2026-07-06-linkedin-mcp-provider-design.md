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
| `distance` | integer | no | miles radius around `location` |
| `workplace_type` | enum: `On-site`, `Remote`, `Hybrid` | no | maps via `linkedin.WorkplaceTypeIDs` |
| `job_type` | enum: `Full-time`, `Part-time`, `Contract`, `Temporary`, `Internship` | no | maps via `linkedin.JobTypeIDs` |
| `easy_apply` | boolean | no | maps to `f_AL` |
| `company_ids` | string | no | comma-separated opaque numeric LinkedIn company IDs, passed through raw (not resolvable by a host LLM without a prior lookup — documented as such) |
| `posted_within` | enum: `Past day`, `Past week`, `Past month` | no | maps to 86400 / 604800 / 2592000 seconds |
| `start` | integer, default 0 | no | raw zero-based result offset (not abstracted into a page number — LinkedIn's own `start` semantics are exposed directly). Description explains: the endpoint always returns exactly 10 cards regardless of `start`; a caller paging through results must increment by exactly 10 each call to avoid the gaps a real browser's 25-per-step scroll traffic produces; a fresh MCP session should start at 0, not the browser-mimicking `linkedin.DefaultStart` (25) that `cmd/linkedin` defaults to. |

No field is `required`, unlike tsmc — LinkedIn's search accepts an entirely
empty query.

Neither tool description hedges on rate limiting: both `linkedin_search_jobs`
and `linkedin_get_job_detail` descriptions state plainly that a single
session gets rate-limited around the 10th consecutive request, the 429
response carries no `Retry-After` hint, and the host should page
conservatively and back off rather than retry immediately on a 429.

### Output

```go
type linkedinSearchOutput struct {
    Data []linkedinJobSummary `json:"data"`
}

type linkedinJobSummary struct {
    ID          string `json:"id"`
    Title       string `json:"title"`
    Company     string `json:"company,omitempty"`
    CompanyURL  string `json:"company_url,omitempty"`
    Location    string `json:"location,omitempty"`
    PostedDate  string `json:"posted_date,omitempty"`
    LooksRemote bool   `json:"looks_remote,omitempty" jsonschema:"Keyword heuristic (title/location substring match for 'remote'/'work from home'/'wfh'), not a field LinkedIn provides. False does not mean confirmed on-site."`
    URL         string `json:"url,omitempty" jsonschema:"Public job posting URL; pass the id portion to linkedin_get_job_detail."`
}
```

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
    Title          string `json:"title"`
    Company        string `json:"company,omitempty"`
    Location       string `json:"location,omitempty"`
    Posted         string `json:"posted,omitempty"`
    SeniorityLevel string `json:"seniority_level,omitempty"`
    EmploymentType string `json:"employment_type,omitempty"`
    JobFunction    string `json:"job_function,omitempty"`
    Industries     string `json:"industries,omitempty"`
    Description    string `json:"description,omitempty" jsonschema:"Full job description as plain text."`
    ApplyURL       string `json:"apply_url,omitempty" jsonschema:"External ATS apply URL; absent for LinkedIn Easy Apply postings."`
    LooksRemote    bool   `json:"looks_remote,omitempty" jsonschema:"Keyword heuristic over title/location only (not the full description), not a field LinkedIn provides. False does not mean confirmed on-site."`
}
```

`CompanyLogo` from the client's `JobDetailResponse` is dropped — it's an
image URL with no use to a text-oriented MCP host, and no other provider's
detail output carries a logo field.

## Error handling

`linkedin.Client` is hand-written (unlike nvidia's openapi-generated client),
so there's no typed error to distinguish upstream status codes. Its errors
are already descriptive strings (`"HTTP 999: bot-suspected, LinkedIn
redirected to its authwall..."`, `"HTTP 429"`, `"redirected to ...: no usable
session"`) and pass straight through `errorResult(err)` — the same treatment
tsmc and google (also hand-written clients) already get.

## Conversion functions

- `linkedinMCPToHTTPRequest(in *linkedinSearchInput) (*linkedin.JobsRequest, error)`
  — validates `workplace_type`/`job_type`/`posted_within` against their
  lookup tables (unknown label → error), splits `company_ids` on commas.
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
  rate-limit avoidance is communicated to the host via tool description text
  only, consistent with how the other providers surface upstream errors
  reactively rather than pre-emptively.
- No changes to `internal/provider/linkedin` or `cmd/linkedin` — this design
  only adds the MCP adapter layer on top of the existing, already-tested
  provider package.
