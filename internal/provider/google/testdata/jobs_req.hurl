# Search jobs, with every documented query parameter filled in — including
# repeated keys for the multi-value filters (location, target_level, degree,
# employment_type, company) — see ../openapi.yaml paths./jobs/results.get.
GET https://www.google.com/about/careers/applications/jobs/results
User-Agent: Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/124.0.0.0 Safari/537.36
Accept: text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8
Accept-Language: en-US,en;q=0.9
[Options]
output: jobs_rsp.html
[QueryStringParams]
q: software engineer
location: Taiwan
location: Singapore
has_remote: true
target_level: MID
target_level: ADVANCED
skills: Python
degree: MASTERS
degree: PHD
employment_type: FULL_TIME
employment_type: PART_TIME
company: Google
company: DeepMind
company: YouTube
sort_by: date
page: 1

HTTP 200
