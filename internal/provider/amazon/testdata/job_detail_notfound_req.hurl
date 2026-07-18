# Unknown ids are a soft not-found: HTTP 200 with zero hits and an empty page.
GET https://www.amazon.jobs/en/search.json
[QueryStringParams]
base_query: 9999999999
offset: 0
result_limit: 10
sort: relevant

HTTP 200

[Asserts]
jsonpath "$.error" == null
jsonpath "$.hits" == 0
jsonpath "$.jobs" count == 0
