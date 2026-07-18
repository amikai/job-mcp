# company= takes canonical Himalayas company slugs only; every returned job
# carries companySlug=leland.
GET https://himalayas.app/jobs/api/search
[QueryStringParams]
company: leland

HTTP 200
