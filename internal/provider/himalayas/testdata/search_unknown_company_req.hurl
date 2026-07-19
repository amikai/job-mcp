# An unrecognized company slug is rejected as a 400 invalid filter, not an
# empty 200 result.
GET https://himalayas.app/jobs/api/search
[QueryStringParams]
company: this-company-does-not-exist-xyz

HTTP 400
