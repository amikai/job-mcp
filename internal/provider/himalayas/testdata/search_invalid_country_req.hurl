# Invalid filter values return 400 with {"ok":false,"errors":"..."} — a
# different error shape from the documented {"error":"..."} object.
GET https://himalayas.app/jobs/api/search
[QueryStringParams]
country: Narnia

HTTP 400
