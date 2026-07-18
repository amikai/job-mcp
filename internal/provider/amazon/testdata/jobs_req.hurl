# Unfiltered keyword search with offset pagination.
GET https://www.amazon.jobs/en/search.json
[QueryStringParams]
base_query: software engineer
offset: 0
result_limit: 2
sort: relevant

HTTP 200

[Asserts]
jsonpath "$.error" == null
jsonpath "$.hits" > 0
jsonpath "$.jobs" count == 2
