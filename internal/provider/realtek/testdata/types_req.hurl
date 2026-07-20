# Job category list used to populate the job_type_id filter.
GET https://recruit.realtek.com/Job/GetAllTypeList
User-Agent: Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/126.0.0.0 Safari/537.36

HTTP 200
[Asserts]
jsonpath "$.Pass" == true
jsonpath "$.Data[0].JobTypeId" exists
jsonpath "$.Data[0].JobType" exists
