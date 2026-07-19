POST https://www.metacareers.com/graphql
User-Agent: Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/126.0.0.0 Safari/537.36
X-FB-LSD: openings-mcp
Origin: https://www.metacareers.com
Referer: https://www.metacareers.com/jobsearch/
Sec-Fetch-Dest: empty
Sec-Fetch-Mode: cors
Sec-Fetch-Site: same-origin
[Form]
lsd: openings-mcp
variables: {"isLoggedIn":false,"search_input":{"q":"engineer","divisions":[],"offices":["Singapore"],"roles":[],"leadership_levels":[],"saved_jobs":[],"saved_searches":[],"sub_teams":[],"teams":[],"is_leadership":false,"is_remote_only":false,"sort_by_new":false,"page":1,"results_per_page":null},"viewasUserID":null}
doc_id: 27506805582236862

HTTP 200
# Meta's /graphql answers JSON with `Content-Type: text/html; charset="utf-8"`;
# hurl rejects the quoted charset and cannot decode the body, so asserts use raw bytes.
[Asserts]
bytes startsWith hex,7b226461746122;
bytes contains hex,6a6f625f7365617263685f776974685f66656174757265645f6a6f6273;
bytes contains hex,53696e6761706f7265;
