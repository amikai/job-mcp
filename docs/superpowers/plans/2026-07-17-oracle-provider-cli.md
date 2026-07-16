# Oracle Recruiting Cloud Provider and CLI — Implementation Plan

**Goal:** Add careers-page discovery, site-bound provider operations, and a
debug CLI without changing `companies.yaml`.

**Spec:** `docs/superpowers/specs/2026-07-17-oracle-provider-cli-design.md`

## Tasks

- [x] Capture a real Candidate Experience careers-page fixture.
- [x] Parse modern `data-apibaseurl` / `data-sitenumber` metadata.
- [x] Support legacy themes that expose only the Candidate Experience base
  path.
- [x] Implement a site-bound client for search, standard facets, filters, and
  job detail.
- [x] Force JSON response negotiation for the generated client.
- [x] Implement `cmd/oracle` with discover, search, facets, and detail
  commands.
- [x] Add fixture-backed provider and CLI end-to-end tests.
- [x] Verify discovery and API operations against modern and legacy live
  Oracle sites.
- [ ] Add the ATS adapter and MCP registration in a later stage.

