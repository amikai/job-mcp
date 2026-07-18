# The API has no separate detail operation. A numeric id_icims is an exact
# searchable term; the client verifies the returned id before accepting it.
GET https://www.amazon.jobs/en/search.json
[QueryStringParams]
base_query: 3164253
offset: 0
result_limit: 10
sort: relevant

HTTP 200

[Asserts]
jsonpath "$.error" == null
jsonpath "$.hits" == 1
jsonpath "$.jobs[0].id_icims" == "3164253"
