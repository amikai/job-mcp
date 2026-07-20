# Detail keyed by JobOppId (the id used in JobDetail?jobid= links), NOT JobId.
# Only JobTitle, Degree, Exp, Location, Requirement are populated; the other
# fields come back null.
GET https://recruit.realtek.com/Job/GetVacancyDetail?JobOppId=18
User-Agent: Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/126.0.0.0 Safari/537.36

HTTP 200
[Asserts]
jsonpath "$.Pass" == true
jsonpath "$.Data.JobTitle" exists
jsonpath "$.Data.Requirement" exists
