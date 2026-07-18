# Category list. Envelope-reuse quirk: the categories arrive under the
# `jobs` key and job-count holds the category count (30 at capture time).
# A job's `category` field carries the `name` here, not the `slug`.
GET https://remotive.com/api/remote-jobs/categories

HTTP 200
[Asserts]
header "Content-Type" contains "application/json"
jsonpath "$.jobs" count > 0
jsonpath "$.jobs[0].id" isInteger
jsonpath "$.jobs[0].name" isString
jsonpath "$.jobs[0].slug" isString
