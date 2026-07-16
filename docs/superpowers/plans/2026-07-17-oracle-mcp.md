# Oracle Recruiting Cloud MCP Wiring — Implementation Plan

**Goal:** Make the existing Oracle Recruiting Cloud provider reachable through
the unified company MCP tools.

**Spec:** `docs/superpowers/specs/2026-07-17-oracle-mcp-design.md`

- [x] Implement `internal/ats.OracleAdapter`.
- [x] Cover roster slugs, careers URL parsing, search pagination, filters,
  detail conversion, and teaching errors with fixture-backed tests.
- [x] Register Oracle in `newATSRegistry`.
- [x] Add the Oracle careers URL shape to registry guidance.
- [x] Add Oracle to `cmd/verify-companies`.
- [x] Update the README provider list.
- [x] Run the full test/build/vet suite.
- [x] Smoke-test search and detail through the real MCP stdio path.
