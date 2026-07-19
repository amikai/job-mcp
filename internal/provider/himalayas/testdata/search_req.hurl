# Search results are fuzzy-relevance ranked: even a keyword with no exact
# match returns fallback results, so q= alone never yields totalCount 0.
GET https://himalayas.app/jobs/api/search
[QueryStringParams]
q: golang
sort: recent

HTTP 200
