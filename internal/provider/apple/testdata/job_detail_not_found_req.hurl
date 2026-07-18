GET https://jobs.apple.com/api/v1/jobDetails/999999999
Accept: application/json
locale: en-us
[QueryStringParams]
locale: en-us

HTTP 404
[Asserts]
header "Content-Type" contains "application/json"
jsonpath "$.error" == "jobsite.general.serviceError"
