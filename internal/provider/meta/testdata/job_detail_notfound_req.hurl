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
variables: {"renderLoggedInView":false,"requisitionID":"999999999999999","viewasUserID":null}
doc_id: 27371134039243725

HTTP 200
# Meta's /graphql answers JSON with `Content-Type: text/html; charset="utf-8"`;
# hurl rejects the quoted charset and cannot decode the body, so asserts use raw bytes.
[Asserts]
bytes startsWith hex,7b226461746122;
bytes contains hex,227863705f7265717569736974696f6e5f6a6f625f6465736372697074696f6e223a6e756c6c;
