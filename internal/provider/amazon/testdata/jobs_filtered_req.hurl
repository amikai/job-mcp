# ISO-3 country filtering is server-side: every returned job is in Taiwan.
GET https://www.amazon.jobs/en/search.json
[QueryStringParams]
base_query: software engineer
normalized_country_code[]: TWN
offset: 0
result_limit: 2
sort: relevant

HTTP 200

[Asserts]
jsonpath "$.error" == null
jsonpath "$.hits" > 0
jsonpath "$.jobs" count == 2
jsonpath "$.jobs[0].country_code" == "TWN"
jsonpath "$.jobs[1].country_code" == "TWN"
