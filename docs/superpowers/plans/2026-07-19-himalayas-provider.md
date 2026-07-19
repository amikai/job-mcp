# Himalayas Provider — Plan

Executed 2026-07-19 in one session, following the integrate-new-provider
pipeline. **Stopped before the MCP surface on purpose** (user request:
"先不要 wire mcp") — see the handoff section below.

Surface decision: Himalayas is a single remote-jobs board (one API, no
tenants), so it takes dedicated `himalayas_*` MCP tools, not an ATS
adapter. No roster.

1. ~~Spec hunt~~ — official OpenAPI 3.1 spec at
   https://himalayas.app/docs/openapi.json; free public API, no auth.
2. ~~Recon + fixtures~~ — captured browse, keyword search, filtered
   search, company search, and both 400 shapes live (six hurl + JSON
   pairs); `hurl --test` green. The official schemas drift from the wire:
   timestamps are Unix seconds not ms, locationRestrictions is
   country-name strings not objects, timezoneRestrictions is numeric
   offsets not "UTC-5" strings, currency is nullable, and the search 400
   body is `{"ok":false,"errors":"..."}`.
3. ~~OpenAPI + generated client~~ — server-side search shape (search
   endpoint + full-feed browse endpoint; no detail endpoint — the feed
   carries complete HTML descriptions). Trimmed spec with all deviations
   and quirks documented in `internal/provider/himalayas/openapi.yaml`;
   ogen client generated; spec in `OPENAPI_SPECS`; `make
   validate-openapi` green.
4. ~~Provider package~~ — mocksrv + client tests covering browse,
   search, filters, company filter, both 400 quirks, fractional
   timezone offsets, and null salary/currency decoding.
5. ~~Debug CLI~~ — `cmd/himalayas` browse/search; verified live
   including 400 paths, positional-arg rejection, and flag validation.
6. **MCP surface — NOT DONE (deferred by user request).** Remaining
   work: `internal/openingsmcp/himalayas.go` with
   `himalayas_search_jobs` (+ a detail story: no upstream detail
   endpoint exists, so either surface the feed's inline description in
   search results or resolve a guid via a company-filtered search) +
   tests; wire the client in `newServer` (`cmd/openings-mcp`); server
   instructions + tool-list test; live stdio smoke test with real
   queries.
7. Roster curation — n/a (no roster; dedicated-tools provider).
8. Docs — README job-board list entry; pending with the MCP stage.
