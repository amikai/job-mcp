# Location list; GetFilterList takes the display name (JobLocation), not the id.
GET https://recruit.realtek.com/Job/GetAllLocationList
User-Agent: Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/126.0.0.0 Safari/537.36

HTTP 200
[Asserts]
jsonpath "$.Pass" == true
jsonpath "$.Data[0].JobLocationId" exists
jsonpath "$.Data[0].JobLocation" exists
