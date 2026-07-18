GET https://jobs.apple.com/api/v1/CSRFToken
Accept: application/json

HTTP 200
[Captures]
csrf_token: header "x-apple-csrf-token"
[Asserts]
header "x-apple-csrf-token" matches "^[0-9a-f]{64}$"
cookie "jssid" exists

# A second-page, newest-first U.S. search verifies that paging, sorting, and
# country filtering are all evaluated server-side.
POST https://jobs.apple.com/api/v1/search
Accept: application/json
Content-Type: application/json
x-apple-csrf-token: {{csrf_token}}
{
  "query": "camera",
  "filters": {
    "locations": [
      "postLocation-USA"
    ]
  },
  "page": 2,
  "locale": "en-us",
  "sort": "newest",
  "format": {
    "longDate": "MMMM D, YYYY",
    "mediumDate": "MMM D, YYYY"
  }
}

HTTP 200
[Asserts]
header "Content-Type" contains "application/json"
jsonpath "$.res.totalRecords" isInteger
jsonpath "$.res.searchResults" count == 20
