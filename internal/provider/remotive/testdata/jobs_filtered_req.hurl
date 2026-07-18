# Proves the documented query params (search, limit, category,
# company_name) are no-ops live: limit=3 still returns the full dump
# because Cloudflare's cache key ignores the query string (the captured
# body was byte-identical to jobs_rsp.json, cf-cache-status: HIT,
# 2026-07-19 — same for unknown company_name and nonexistent category
# probes, so there is no error shape either). No paired _rsp.json on
# purpose. If the count assert ever fails, upstream filtering started
# working and the provider's client-side filtering should be revisited.
GET https://remotive.com/api/remote-jobs
[QueryStringParams]
search: devops
limit: 3

HTTP 200
[Asserts]
jsonpath "$.jobs" count > 3
