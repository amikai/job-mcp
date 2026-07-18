# Amazon Jobs Provider — Plan

Design: [specs/2026-07-19-amazon-provider-design.md](../specs/2026-07-19-amazon-provider-design.md)

Executed 2026-07-19 in one session, following the integrate-new-provider
pipeline:

1. ~~Spec hunt and recon~~ — verified the undocumented public JSON search
   endpoint, every exposed filter, pagination limits, soft errors, and exact-ID
   lookup. No official API specification or developer documentation exists.
2. ~~Capture fixtures~~ — real happy, filtered, detail, not-found, and soft
   error hurl/response pairs; all five live replays pass.
3. ~~Client~~ — minimal OpenAPI spec, ogen generation, and a handwritten
   wrapper for defaults, validation, soft errors, and exact-ID detail lookup.
4. ~~Provider package~~ — fixture-replaying mock server and provider tests.
5. ~~Debug CLI~~ — `cmd/amazon` `search` and `detail` subcommands; both were
   checked live with JSON output, with text rendering covered by unit tests.
6. ~~MCP surface~~ — dedicated `amazon_search_jobs` and
   `amazon_get_job_detail` tools, registered in the real server and verified
   through live stdio requests (two searches plus one full detail lookup).
7. ~~Docs~~ — README provider list and server tool-selection instructions.
8. ~~Verification~~ — formatting, hurl lint, OpenAPI validation, ogen
   regeneration, vet, focused tests, and the full Go test suite pass. The
   repository-wide hurl run passed 113/114 files; the sole failure is an
   unrelated existing JOIN fixture whose live URL now redirects with 301.

Roster curation is not applicable because Amazon Jobs is a single careers
site.
