# Full dump of the free API's job list: the response is the whole board
# (job-count == total-job-count, 42 jobs at capture time), full HTML
# descriptions inline, served from a Cloudflare cache with a multi-hour
# age. There is no pagination and no detail endpoint — detail lookup means
# finding the id in this dump.
GET https://remotive.com/api/remote-jobs

HTTP 200
[Asserts]
header "Content-Type" contains "application/json"
jsonpath "$['job-count']" isInteger
jsonpath "$['total-job-count']" isInteger
jsonpath "$.jobs" count > 0
jsonpath "$.jobs[0].id" isInteger
jsonpath "$.jobs[0].title" isString
jsonpath "$.jobs[0].company_name" isString
jsonpath "$.jobs[0].description" isString
