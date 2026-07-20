# Full dump: every open vacancy in one response (~200 items), no pagination.
# Fields are zh-TW regardless of the lang cookie or ?lang= param.
GET https://recruit.realtek.com/Job/GetAllJobList
User-Agent: Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/126.0.0.0 Safari/537.36

HTTP 200
[Asserts]
jsonpath "$.Pass" == true
jsonpath "$.Data" count > 100
jsonpath "$.Data[0].JobOppId" exists
jsonpath "$.Data[0].JobTitle" exists
