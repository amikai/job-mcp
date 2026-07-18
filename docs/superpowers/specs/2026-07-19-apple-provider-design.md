# Apple Jobs Provider — Design

## Context and surface

`jobs.apple.com` is Apple’s single public careers site, not a multi-tenant ATS.
It therefore gets dedicated `apple_search_jobs` and
`apple_get_job_detail` MCP tools rather than an `internal/ats.Adapter`.

No official Apple Jobs API or OpenAPI document is published. The public React
application uses a JSON API under `/api/v1`; the minimal contract below was
derived from the site bundle and verified against live responses on 2026-07-19.

## Live API contract

Search is a three-part session flow:

1. `GET /api/v1/CSRFToken` creates a `jssid` session cookie and returns the
   token in the `x-apple-csrf-token` response header.
2. `POST /api/v1/search` sends that token in the matching request header while
   retaining the cookie. The JSON body carries `query`, `filters`, one-based
   `page`, `locale`, `sort`, and the date formats expected by the web app.
3. The response contains `res.searchResults` (20 records per page) and
   `res.totalRecords`.

Detail is simpler: `GET /api/v1/jobDetails/{jobId}?locale=en-us` works without
a search session. It accepts the stable numeric `positionId` returned by
search, even when the public posting URL includes a location suffix. An
unknown position returns HTTP 404 with Apple’s generic service-error JSON.

The search API treats a query as relevance ranking when `sort` is
`relevance`; an empty sort can make the query ineffective. Country filters
are arrays of Apple IDs such as `postLocation-TWN`, not the richer location
objects embedded in the SSR page. These quirks are documented in
`openapi.yaml` and enforced by the provider wrapper.

## Generated client and provider wrapper

The minimal OpenAPI 3.1 specification covers only session initialization,
search, and detail. ogen output lives in `internal/provider/apple/api` so the
public `apple.Client` can compose the generated client with protocol behavior
that OpenAPI alone cannot express:

- clone the supplied `http.Client` and attach a private cookie jar;
- serialize the CSRF-token and search requests with a mutex, preventing
  concurrent searches from crossing session/token pairs;
- supply the fixed `en-us` locale and date formats;
- translate generated response unions into ordinary Go results and useful
  not-found errors.

Search accepts a keyword, ISO 3166-1 alpha-3 country code, sort order, and
one-based page. The client validates the country as exactly three ASCII
letters, uppercases it, and sends `postLocation-<CODE>`. This gives MCP and CLI
callers a stable human-facing parameter without shipping a large, fast-aging
location roster. Search summaries expose the numeric position ID, title,
team, locations, posting date, weekly hours, home-office flag, summary, and a
public Apple Jobs URL. Detail exposes the full summary, responsibilities,
minimum and preferred qualifications, teams, locations, employment type,
posting date, and public URL.

## Debug CLI

`cmd/apple` uses ff/v4 and follows the current subcommand convention:

- `search --keyword TEXT --country ISO3 [--sort relevance|newest] [--page N]`
- `detail --job-id ID`

Both commands support `--format text|json`, reject stray positional
arguments, validate required values before the network call, and apply a
caller-configurable timeout.

## Fixtures and testing

Captured fixtures are real API responses:

- a scoped Taiwan keyword search;
- a second-page/newest search to verify pagination and sort mapping;
- one full detail record;
- the live not-found error response.

Each search hurl file performs the CSRF request, captures the response header,
and reuses its cookie state for the POST. `NewMockServer` reproduces the token,
cookie, search, detail, and 404 behavior. Provider tests exercise request
mapping and generated decoding; CLI tests cover validation and conversions;
MCP tests run both tools through an in-memory MCP transport.

## Server and docs

`cmd/openings-mcp` constructs the Apple client over the shared timeout-enabled
HTTP transport, registers both tools, and lists Apple Careers in its routing
instructions and tool-list contract test. The README dedicated-site list is
updated. A final live stdio smoke test searches Apple Taiwan jobs and fetches
one returned position through the real MCP server.
