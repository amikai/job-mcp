# Jobicy Provider — Plan

Scope for this session: stages 1–5 only (through the debug CLI). MCP
wiring (stage 6+) is deliberately deferred at the user's request — the
handoff notes at the bottom describe what remains.

Jobicy (jobicy.com) is a remote-jobs board with an official, free,
no-auth API ([Jobicy/remote-jobs-api](https://github.com/Jobicy/remote-jobs-api)),
so this is a dedicated-tools provider (single site, no tenant roster).

## API shape (verified live 2026-07-19, apiVersion 2.2.14)

One data endpoint, `GET https://jobicy.com/api/v2/remote-jobs`, with
query params `count` (1–100, default 100), `geo`, `industry` (slugs),
and `tag` (free-text over title+description). The same path doubles as
the taxonomy endpoint via `?get=locations` / `?get=industries`, which
return the valid `geo`/`industry` slugs.

Classification: **full dump with server-side soft search** — every list
row already carries the complete HTML `jobDescription`, and there is no
detail-by-id endpoint upstream. Search-then-detail collapses into the
one call; the eventual MCP surface can serve detail from a search row
or skip a detail tool entirely (decide at stage 6).

Observed drift from the official README (spec is written from captured
traffic, not the README):

- `id` is a number, not a string; `jobSlug` exists but is undocumented.
- `jobIndustry` and `jobType` are arrays of strings (docs claim string),
  with HTML entities in values (`DevOps &amp; Infrastructure`).
- Salary fields (`salaryMin/Max/Currency/Period`) appear only on ~40%
  of rows — optional, absent otherwise.
- Zero matches: HTTP **404** with the normal envelope plus
  `statusCode: 404, success: false` and `jobs: []`.
- Invalid `geo`/`industry` slug: HTTP **400** with a small
  `{"success": false, "error": "..."}` body.
- Usage terms: credit Jobicy, link applications to the original job
  `url`, poll at most hourly; listings surface with a 6-hour delay.

## Stages (executed 2026-07-19)

1. ~~Spec hunt~~ — official docs found (README above); no official OpenAPI.
2. ~~Recon + fixtures~~ — hurl+JSON pairs: happy path, filtered search
   (geo+industry+tag), empty result (404), invalid filter (400),
   locations, industries; `hurl --test` green live.
3. ~~OpenAPI + client~~ — minimal `openapi.yaml` from traffic; the
   query-routed taxonomy variants are modeled as a `oneOf` on the single
   operation's 200, which ogen turns into a field-discriminated sum
   type; zero-match 404 decodes as the jobs envelope, and 400s surface
   through ogen's convenient-error path. In `OPENAPI_SPECS`;
   `make validate-openapi` OK.
4. ~~Provider package~~ — `mocksrv.go` (query-dispatch replay of the six
   fixtures) + `client_test.go` (all four search outcomes + both
   taxonomies) + `doc.go`.
5. ~~Debug CLI~~ — `cmd/jobicy` (ff/v4): `search`, `locations`,
   `industries` subcommands (no `detail` — nothing upstream to call);
   verified live including 404-empty, 400, flag-range, and stray-arg
   paths.

## Handoff (not done here)

6. MCP surface: `internal/openingsmcp/jobicy.go` (`RegisterJobicy` +
   tests), wire client in `newServer`, serverInstructions, live stdio
   smoke test.
7. Roster curation — n/a (dedicated-tools provider).
8. Docs — README job-board list.
