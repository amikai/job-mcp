# iCIMS Career Portal OpenAPI — Design

## Context

iCIMS is a multi-company ATS. Public careers sites typically live at
`careers-<slug>.icims.com` (also `uscareers-<slug>.icims.com` and other
`<tenant>.icims.com` shapes). The partner-gated Talent Cloud API at
developer.icims.com requires Partner Program credentials and is unusable for
anonymous cross-company search.

There is no public JSON jobs API. Search and detail are server-rendered HTML
served from the iframe careers portal (`in_iframe=1`), the same SSR pattern
as SuccessFactors Career Site Builder (`internal/provider/successfactors`).

## Verified wire behavior (2026-07-15)

`GET https://<career-host>/jobs/search?ss=1&pr={page}&in_iframe=1`:

- requires no authentication or cookies;
- returns `200 text/html` for a live tenant;
- returns `404` for an unknown `careers-*` host;
- paginates with zero-based `pr`; page size is **tenant-configured** (observed
  20 on Advantage Solutions / Audacy, 50 on Peraton / 360care);
- advertises `Page X of Y` in `.iCIMS_PagingBatch` for total page count;
- each posting is an `<li class="iCIMS_JobCardItem">` with title link
  `/jobs/{id}/{slug}/job`, location, and optional summary;
- free-text title/description search: `searchKeyword`;
- location: `searchLocation` expects the encoded `<option value>` from the
  portal select (e.g. `12781-12827-Austin`), not free text such as `Austin`.

`GET https://<career-host>/jobs/{id}/{slug}/job?in_iframe=1`:

- `{slug}` is cosmetic — any non-empty path segment resolves the same
  posting from `{id}` alone (verified with `x` and `foo`);
- returns `200` with `application/ld+json` `JobPosting` (title, description
  HTML, employmentType, datePosted, jobLocation, hiringOrganization);
- unknown/expired IDs return **HTTP 410** and render the job list, not a
  dedicated error page (no JobPosting JSON-LD).

Live verification: BusPatrol, Peraton, Advantage Solutions (asm), 360care,
Amer Sports, Audacy.

## Spec shape

`internal/provider/icims/openapi.yaml` documents two HTML operations (no
ogen-generated client — same hand-written client approach as
successfactors):

```text
GET /jobs/search?ss=1&pr=&in_iframe=1&searchKeyword=&searchLocation=
  -> 200 HTML | 404

GET /jobs/{id}/{slug}/job?in_iframe=1
  -> 200 HTML (JSON-LD JobPosting) | 410
```

## Family

**Server-side search** with tenant-variable page size. The ATS adapter maps
unified `pageSize` (20) onto one or two upstream `pr` pages by discovering
the page size from the first response, then slicing.

## Out of scope

Partner Talent Cloud API, application submission, HTML iframe-wrapper pages
without `in_iframe=1`, and non-portal vanity domains that only embed iCIMS
via third-party frames.
