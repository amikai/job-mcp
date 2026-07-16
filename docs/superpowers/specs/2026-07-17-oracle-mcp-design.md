# Oracle Recruiting Cloud MCP Wiring — Design

## Goal

Expose the existing Oracle Recruiting Cloud provider through the unified
`search_jobs_by_company`, `get_filters_by_company`, and
`get_job_detail_by_company` MCP tools.

## Adapter identity and slugs

Oracle is a multi-company ATS, so it implements `internal/ats.Adapter`.
Curated companies use `host/site_number` as their slug because several Oracle
Fusion hosts serve multiple public career sites. The combination is already
validated as unique by the provider roster.

Recognized curated careers URLs fold back to that roster slug. A valid
Candidate Experience URL outside the roster becomes a canonical careers URL
slug. Search, filters, and detail rediscover the page's internal site number,
keeping the MCP calls stateless.

## Search and filters

Oracle search is server-side with a unified page size of 20. Query maps to the
Candidate Experience keyword. Filters are presented as facet display labels,
then resolved to Oracle facet IDs with a one-result facet probe before the real
search. Location fuzzy-matches the location facet; `remote` maps to the
workplace-type facet.

## Detail

Detail maps Oracle's requisition record into the unified result. Description,
responsibilities, qualifications, and organization boilerplate are converted
from HTML to plain text and joined into one readable body.

## Wiring

Register the adapter in the MCP server, careers URL guidance, and
`verify-companies`. Add focused fixture-backed adapter tests, full registry
tests, README provider documentation, and a live MCP smoke test.
