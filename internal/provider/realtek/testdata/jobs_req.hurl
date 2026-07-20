# Dump of open vacancies in one response, no pagination — capped at 200 rows.
# The cap is real: disjoint xp=0/xp=1 GetFilterList buckets union to 254
# distinct vacancies while this endpoint returns exactly 200 (2026-07-21).
# Fields are zh-TW regardless of the lang cookie or ?lang= param.
GET https://recruit.realtek.com/Job/GetAllJobList
User-Agent: Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/126.0.0.0 Safari/537.36

HTTP 200
[Asserts]
jsonpath "$.Pass" == true
jsonpath "$.Data" count > 100
jsonpath "$.Data[0].JobOppId" exists
jsonpath "$.Data[0].JobTitle" exists
