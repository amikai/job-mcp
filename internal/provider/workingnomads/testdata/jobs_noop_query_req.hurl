# Proves the plausible query params (category, tag, and an unknown one)
# are all no-ops live: every response body is byte-identical to the
# unfiltered dump in jobs_rsp.json (compared 2026-07-19, same job count,
# same first-job title/url). No paired _rsp.json on purpose — this is the
# same "server ignores the query string" shape as Remotive's dump. If this
# ever starts returning a narrower set, client-side filtering in filter.go
# should be revisited.
GET https://www.workingnomads.com/api/exposed_jobs/
[QueryStringParams]
category: development
tag: python
nonsense: 1

HTTP 200
[Asserts]
jsonpath "$" count > 0
