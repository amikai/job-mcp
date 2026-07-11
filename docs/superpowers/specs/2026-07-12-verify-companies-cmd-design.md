# verify-companies cmd design

Date: 2026-07-12
Issue: #91 — "verify tenant identifiers are still valid"

## Purpose

A standalone CLI, `cmd/verify-companies`, that sweeps the four curated ATS
rosters (`internal/provider/{ashby,greenhouse,lever,workday}/companies.yaml`,
~550 entries) and verifies each entry by running a real search through the
unified `internal/ats` adapters — the same code path the MCP server serves.
Each entry's report line includes its total job count, so the sweep doubles
as a roster health report (a valid tenant with 0 jobs is visible at a
glance). It prints a per-entry report and exits non-zero when entries have
gone stale, so it works both as a manual audit tool and in CI.

## Verification call

For every roster entry, call that adapter's
`Search(ctx, slug, ats.SearchParams{Page: 1})`:

- **ashby / greenhouse / lever** — full-dump adapters: one GET of the whole
  board, `TotalCount` = every listed job.
- **workday** — server-side search: one POST of page 1, `TotalCount` from
  the API.

Classification:

| Outcome                                                             | Status  |
|---------------------------------------------------------------------|---------|
| Search succeeds                                                     | OK (with job count) |
| Upstream HTTP 404 — all providers; HTTP 422 — workday (bad tenant)  | INVALID |
| Anything else: timeout, 5xx, other 4xx, network or decode error     | ERROR   |

INVALID detection unwraps the error chain for typed status-code errors:
lever and workday surface `*<pkg>.ErrorResponseStatusCode` (matched through
an `interface{ error; GetStatusCode() int }` target so the cmd needs no
provider imports), with ogen's `validate.UnexpectedStatusCodeError` as
fallback. The ashby and greenhouse adapters translate their typed 404
responses into `"... not found upstream"` errors before any status code is
visible, so those two are matched on that message. ERROR means
indeterminate, likely transient — not proof the entry is stale.

## Structure

- `cmd/verify-companies/main.go` only. No test file.
- CLI built with `ff/v4`, matching the other `cmd/` tools.
- The cmd depends only on `internal/ats`: adapters are constructed with the
  same base URLs as `cmd/openings-mcp/main.go`, rosters come from each
  adapter's `Roster()`, and verification goes through `Search()`. The
  provider packages are not imported directly.
- Entries fan out through a bounded worker pool.

## Flags

- `--provider` — comma-separated subset of `ashby,greenhouse,lever,workday`;
  default all four.
- `--timeout` — per-request timeout, default 300s (the largest full-dump
  boards, e.g. Palantir and Veeva on lever, take minutes to download; 60s
  produced false ERRORs).
- `--concurrency` — worker pool size, default 8.
- `--format` — `text` (default) or `json`.

## Output and exit code

Text format: one line per entry, `STATUS  provider  company  slug  jobs
detail`, grouped by provider (`jobs` is the total job count for OK entries,
blank otherwise; `detail` is the error message for non-OK entries), followed
by a summary of counts per status. JSON format: one object
`{"results": [...], "summary": {"ok": N, "invalid": N, "error": N}}` where
each result carries provider, company, slug, status, job count, and detail.

Exit codes: any INVALID → 1; no INVALID but any ERROR → 2; all OK → 0.
