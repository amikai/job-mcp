# Full dump of the public jobs feed: the whole board in one response, full
# HTML descriptions inline, no pagination (34 jobs at capture time). There
# is no separate detail endpoint — see detail_not_found_req.hurl.
GET https://www.workingnomads.com/api/exposed_jobs/

HTTP 200
[Asserts]
header "Content-Type" == "application/json"
jsonpath "$" count > 0
jsonpath "$[0].url" isString
jsonpath "$[0].title" isString
jsonpath "$[0].description" isString
jsonpath "$[0].company_name" isString
jsonpath "$[0].category_name" isString
jsonpath "$[0].tags" isString
jsonpath "$[0].location" isString
jsonpath "$[0].pub_date" isString
