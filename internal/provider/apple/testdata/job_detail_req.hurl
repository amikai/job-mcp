GET https://jobs.apple.com/api/v1/jobDetails/200624996
Accept: application/json
locale: en-us
[QueryStringParams]
locale: en-us

HTTP 200
[Asserts]
header "Content-Type" contains "application/json"
jsonpath "$.res.positionId" == "200624996"
jsonpath "$.res.postingTitle" exists
jsonpath "$.res.description" exists
