# BambooHR Provider — Plan

Executed 2026-07-18 (Claude session hit rate limit mid-adapter; finished in a
follow-up session), following the integrate-new-provider pipeline:

1. ~~Spec hunt~~ — BambooHR's official API at documentation.bamboohr.com is a
   separate, authenticated product. Public careers boards use undocumented
   endpoints under `{subdomain}.bamboohr.com`; the contract was written from
   captured traffic against live boards.
2. ~~Recon + fixtures~~ — live-probed several tenants. Endpoints:
   `GET /careers/list` (full dump, sparse summaries) and
   `GET /careers/{id}/detail` (full posting). Unknown tenants 302-redirect to
   the marketing site rather than returning an error status. Captured list /
   list-empty / list-variety / detail / detail-nulls / detail-not-found /
   unknown-tenant fixtures; hurl requests live under `testdata/`.
3. ~~OpenAPI + client~~ — full-dump shape (no pagination, no server-side
   search) with a separate detail endpoint, like Greenhouse/Recruitee. Minimal
   `openapi.yaml` over the two endpoints; ogen client. Nullability taken from
   real responses (nullable location fields, optional description HTML, etc.).
4. ~~Provider package~~ — mocksrv + client tests; seed roster of 5
   live-verified boards (Aroa Biosurgery, Ashtead Technology, Concept2,
   Curtin Maritime, Giatec Scientific). `WorkModeLabel` maps locationType
   `"0"|"1"|"2"` → On-site / Remote / Hybrid.
5. ~~Debug CLI~~ — `cmd/bamboohr` companies/search/get (ff/v4); verified live
   including detail and companies listing.
6. ~~MCP surface~~ — `internal/ats/bamboohr.go` dump-style adapter
   (`searchDump`); redirect-blocking HTTP client so 302 tenants surface as
   "not found upstream". Registered in the server, careers-host patterns
   (`*.bamboohr.com`), and verify-companies. Live stdio smoke: search on 3
   roster companies, careers-URL input, filters, and detail for job 201 on
   Curtin Maritime — all through the real server.
7. Roster curation — seed only; bulk discovery left for a later
   discover-companies session.
8. ~~Docs~~ — README ATS list; this plan + design spec.
