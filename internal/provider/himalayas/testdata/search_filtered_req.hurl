# Proves country/employment_type/seniority are real server-side filters:
# totalCount narrows from 97 (q=golang alone) to 18.
GET https://himalayas.app/jobs/api/search
[QueryStringParams]
q: golang
country: US
employment_type: Full Time
seniority: Senior

HTTP 200
