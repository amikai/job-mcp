# Remotive Provider Implementation Plan

**Goal:** Add `internal/provider/remotive` (OpenAPI spec, ogen client, fixture-replaying tests) and `cmd/remotive` (debug CLI) for Remotive's public remote-jobs API. The MCP surface is deliberately deferred to a later session (user request); this plan ends at the debug CLI + live smoke check.

**Surface decision (for the later wiring session):** Remotive is a single job board, not a multi-company ATS → dedicated `remotive_search_jobs` / `remotive_get_job_detail` tools in `internal/openingsmcp/remotive.go`, following jobindex/mynavi rather than an `ats.Adapter`.

## API shape (captured 2026-07-19)

Official docs: https://github.com/remotive-com/remote-jobs-api

- `GET https://remotive.com/api/remote-jobs` — full dump of the free tier's job list (42 jobs at capture time; `total-job-count` equals `job-count`, jobs delayed 24h upstream).
- `GET https://remotive.com/api/remote-jobs/categories` — category list; quirk: reuses the jobs envelope (`jobs` key holds categories, `job-count` the category count).

**Load-bearing quirk:** the documented query params (`category`, `company_name`, `search`, `limit`) are all no-ops live. Responses come from a Cloudflare cache whose key ignores the query string (`cf-cache-status: HIT`, multi-hour `age`, byte-identical bodies for `?limit=5`, `?search=devops&limit=3`, and unknown-company/bad-category probes). Consequence: **dump-style provider — all searching/limiting happens client-side in our code**; the generated client intentionally models no query params.

Other observations grounding the spec:

- Every job carries all fields except `company_logo_url` (present on 21/42, duplicate of `company_logo`); `salary` is always present but sometimes `""`.
- `publication_date` has no timezone (`2026-07-16T13:28:02`) — modeled as plain string, not `format: date-time` (ogen's RFC3339 decoder would reject it).
- Envelope keys `0-legal-notice` and `00-warning` start with digits; ogen must be checked to generate valid identifiers for them.
- `job_type` observed values: `full_time`, `contract`, `part_time`, `freelance`; docs also list `internship`. Modeled as open string.
- Rate limit: >2 req/min is blocked; upstream asks for ≤4 fetches/day. Tests replay fixtures; only `make hurl-test` and the CLI touch the live API.

## Tasks

- [x] Capture fixtures with 35s spacing → `internal/provider/remotive/testdata/` (`jobs`, `jobs_filtered` for the no-op proof, `categories` + hurl files)
- [x] `openapi.yaml` (two operations, quirks documented inline) + `gen.go` + `go generate` + add to `OPENAPI_SPECS` + `make validate-openapi`
- [x] `mocksrv.go` + `client_test.go` (decode fixtures via generated client) + `filter.go`/`filter_test.go` (client-side FilterJobs shared with the future MCP surface) + `doc.go`
- [x] `cmd/remotive` — `search` (client-side keyword/category/company/job-type/location/limit filtering over the dump), `detail --id` (resolve from dump), `categories`; live smoke check
- [ ] Hand off MCP wiring (`internal/openingsmcp/remotive.go` registering `remotive_search_jobs`/`remotive_get_job_detail` on top of `FilterJobs`, wire in `newServer`, README provider list) to a follow-up session
