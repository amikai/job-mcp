# Amazon Jobs Provider — Design

## Context

Amazon Jobs is a single public careers site, not a multi-tenant ATS. It gets
dedicated `amazon_search_jobs` and `amazon_get_job_detail` MCP tools.

No official OpenAPI specification or developer documentation was found. Live
recon on 2026-07-19 confirmed the unauthenticated JSON endpoint used by the
site:

```
GET https://www.amazon.jobs/en/search.json
```

It works outside a browser without cookies or special headers.

## API surface

The endpoint performs server-side keyword/filter search and offset pagination.
The integration exposes the verified parameters `base_query`,
`normalized_country_code[]`, `normalized_city_name[]`, `category[]`,
`business_category[]`, `schedule_type_id[]`, `sort`, `offset`, and
`result_limit`.

`sort` accepts `relevant` and `recent`. The upstream silently treats unknown
sort values as relevance, so the generated enum and wrapper reject them.
`result_limit` is capped at 100. Exceeding it is an unusual soft failure:
Amazon returns HTTP 200 with `error` set and `jobs: null`; the wrapper turns
that payload into a Go error.

Every search hit carries the full description, basic qualifications, preferred
qualifications, apply URL, and public `job_path`. There is no separate JSON
detail endpoint. The numeric `id_icims` field is both the public posting ID in
`job_path` and an exact searchable term. `Client.JobDetail` therefore calls
the search endpoint with that ID and accepts only a result whose `id_icims`
matches exactly. An unknown ID is a normal zero-hit HTTP 200 response and maps
to `ErrJobNotFound`.

The opaque UUID `id` is retained in the generated model but is not exposed as
the MCP job ID. HTML fragments in description and qualification fields are
converted to plain text at the MCP and CLI presentation boundaries.

## Implementation

- Minimal reverse-engineered OpenAPI 3.1 spec plus ogen-generated client in
  `internal/provider/amazon`.
- A small handwritten wrapper supplies defaults, validates pagination, handles
  the upstream soft-error payload, and implements exact-ID detail lookup.
- Real search, filtered-search, detail, and not-found response fixtures are
  replayed by `NewMockServer`; matching hurl files replay the live contract.
- `cmd/amazon` provides `search` and `detail` subcommands with text and JSON
  output.
- Dedicated MCP tools return compact summaries from search and plain-text full
  descriptions from detail.

## Validation

The provider is complete when OpenAPI validation/code generation, provider
tests, CLI tests, MCP tests, the repository test suite, live hurl requests, and
live stdio MCP search/detail calls all pass.
