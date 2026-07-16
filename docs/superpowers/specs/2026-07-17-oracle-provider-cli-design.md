# Oracle Recruiting Cloud Provider and CLI — Design

## Goal

Turn the generated Oracle Candidate Experience client into an end-to-end
provider path that supports curated companies and direct public careers URLs,
discovers the internal site number and API origin when needed, and supports
search, facets, and detail through a debug CLI.

This stage deliberately does not modify the Oracle company roster. The CLI
reads the existing roster through `--company`, while also accepting a direct
Candidate Experience careers URL for uncurated sites.

## Provider shape

`internal/provider/oracle/site.go` adds:

- `DiscoverSite`, which fetches the public Candidate Experience HTML and reads
  the `<base>` element's `data-apibaseurl` and `data-sitenumber`;
- fallback discovery for older themes that omit those attributes, using the
  final page origin and `/sites/<site>` path segment;
- `SiteClient`, which binds the generated ogen client to one discovered site;
- typed search requests, standard facet names, compact job summaries, and
  detail records;
- stable construction and validation of Oracle ADF `finder` expressions;
- `ErrJobNotFound` for Oracle's HTTP 200 plus empty `items` detail response.

The generated client does not emit an `Accept` header. Without one, Oracle can
return `application/vnd.oracle.adf.resourcecollection+json`, which the trimmed
generated decoder does not accept. `SiteClient` wraps its HTTP transport to
send `Accept: application/json`, matching the captured Hurl requests.

## CLI

`cmd/oracle` uses `ff/v4` and aligns with the other multi-company provider
CLIs. `--company` accepts either a curated company name or a Candidate
Experience careers URL. URL discovery remains an internal provider operation
rather than a separate public CLI command.

Commands:

- `companies`: list curated Oracle companies and their careers URLs;
- `search`: keyword search, absolute-offset pagination, and repeatable
  `--filter name=id` facet filters;
- `facets`: retrieve all standard facets and live counts;
- `detail`: retrieve and render one posting's public description sections.

Every command supports text and JSON output. For direct URL input, discovery
and the API request share one caller-controlled timeout.

## Validation

- fixture-backed discovery, search, facet, detail, and missing-detail tests;
- CLI tests for curated company resolution, company listing, and the full
  careers-page discovery plus API flow against `NewMockServer`;
- live Mayo Clinic checks for discovery, keyword search, facets, location
  filtering, detail, and missing detail;
- live KPMG check for the legacy theme fallback and search.
