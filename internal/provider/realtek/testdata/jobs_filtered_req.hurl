# Server-side filtered search. Form fields: keyword (substring, matches
# title/requirement), job_location (display name, not id), xp (MINIMUM years
# of experience: N returns Exp >= N, 0 returns only Exp 0/-1, -1 = no limit),
# job_type_id. Omitted/empty fields mean "any". Quirk: title keyword hits
# bypass the other filters (soft filter) — see openapi.yaml.
POST https://recruit.realtek.com/Job/GetFilterList
User-Agent: Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/126.0.0.0 Safari/537.36
[Form]
keyword: verification
xp: -1

HTTP 200
[Asserts]
jsonpath "$.Pass" == true
jsonpath "$.Data" count > 0
