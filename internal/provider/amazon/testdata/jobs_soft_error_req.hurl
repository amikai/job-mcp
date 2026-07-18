# result_limit above 100 is a soft failure: HTTP 200, an error string, and
# jobs: null. The provider wrapper converts this payload into a Go error.
GET https://www.amazon.jobs/en/search.json
[QueryStringParams]
offset: 0
result_limit: 101
sort: relevant

HTTP 200

[Asserts]
jsonpath "$.error" == "Result limit cannot be greater than 100"
jsonpath "$.hits" == 0
jsonpath "$.jobs" == null
