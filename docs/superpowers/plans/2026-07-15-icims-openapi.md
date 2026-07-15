# iCIMS Career Portal OpenAPI — Implementation Plan

**Goal:** Add a hand-written client for iCIMS public career-portal HTML
search and detail pages, documented by a minimal OpenAPI spec.

**Architecture:** Per-host client calls SSR search and detail endpoints.
HTML is parsed with goquery; detail fields come primarily from schema.org
JSON-LD. Generated ogen code is not used (HTML responses, like
successfactors).

**Spec:** `docs/superpowers/specs/2026-07-15-icims-openapi-design.md`

## Tasks

- [x] Capture hurl + HTML fixtures under `internal/provider/icims/testdata/`.
- [x] Write `openapi.yaml`, `client.go`, `parse.go`, `mocksrv.go`, tests.
- [x] Seed `companies.yaml` with verified tenants.
- [x] Add debug CLI `cmd/icims`.
- [x] ATS adapter + registry wiring.
- [x] README / host-pattern docs.
