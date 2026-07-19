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
variables: {"renderLoggedInView":false,"requisitionID":"1063741453022215","viewasUserID":null}
doc_id: 27371134039243725

HTTP 200
# Meta's /graphql answers JSON with `Content-Type: text/html; charset="utf-8"`;
# hurl rejects the quoted charset and cannot decode the body, so asserts use raw bytes.
[Asserts]
bytes startsWith hex,7b226461746122;
bytes contains hex,226964223a223130363337343134353330323232313522;
bytes contains hex,6d696e696d756d5f7175616c696669636174696f6e73;
bytes contains hex,726573706f6e736962696c6974696573;
