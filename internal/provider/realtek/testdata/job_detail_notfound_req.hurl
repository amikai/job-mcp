# Quirk: an unknown JobOppId is NOT a 404. It's HTTP 200 with Pass:true and a
# Data object whose fields are all null (JobTitle == null signals not-found;
# the site's own JS checks exactly that).
GET https://recruit.realtek.com/Job/GetVacancyDetail?JobOppId=999999
User-Agent: Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/126.0.0.0 Safari/537.36

HTTP 200
[Asserts]
jsonpath "$.Pass" == true
jsonpath "$.Data.JobTitle" == null
