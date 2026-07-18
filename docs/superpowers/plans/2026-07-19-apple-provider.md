# Apple Jobs Provider — Plan

Design: [specs/2026-07-19-apple-provider-design.md](../specs/2026-07-19-apple-provider-design.md)

Executed 2026-07-19 in one session, following the
`integrate-new-provider` pipeline:

1. ~~Spec hunt~~ — confirmed that Apple publishes no Jobs API or OpenAPI
   document; documented the production-site findings in the design notes.
2. ~~Recon + fixtures~~ — recovered the `/api/v1` calls from the production
   bundle and browser flow, replayed them outside the browser, probed paging,
   filtering, CSRF, and not-found behavior, then captured real hurl/JSON pairs.
   Four hurl files (six live requests) pass.
3. ~~OpenAPI + generated client~~ — wrote the minimal three-operation spec,
   added the ogen directive and Makefile entry, generated into the `api`
   subpackage, and validated the spec. Regeneration is clean.
4. ~~Provider package~~ — added the session-aware wrapper, strict TLS mock
   server, and fixture-replaying tests, including the exact search envelope.
5. ~~Debug CLI~~ — added ff/v4 `search` and `detail` subcommands with
   text/JSON output, validation, and tests; verified both live.
6. ~~MCP surface~~ — added and tested `apple_search_jobs` and
   `apple_get_job_detail`, wired the client into `cmd/openings-mcp`, and ran a
   real stdio search-to-detail smoke test. The live search returned 11 Taiwan
   jobs and detail succeeded with returned position ID `200624996`.
7. Roster curation — not applicable because Apple is a dedicated site.
8. ~~Docs and validation~~ — updated README and server instructions; ran hurl
   formatting/lint, OpenAPI validation, gofmt, unit tests, vet/lint where
   available, race tests, and final diff checks.
