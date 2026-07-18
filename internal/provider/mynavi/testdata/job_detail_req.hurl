# Job detail page. The full posting is in a schema.org JobPosting JSON-LD
# block: title, hiringOrganization, employmentType, industry,
# occupationalCategory, datePosted, validThrough, jobLocation[], baseSalary,
# experienceRequirements, workHours, jobBenefits, description, url.
# NOTE: postings expire (validThrough ~4 weeks out); when this ID goes 404,
# recapture with a fresh ID from jobs_req.hurl's first result.
GET https://tenshoku.mynavi.jp/jobinfo-348855-1-29-1/
accept: text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8
accept-language: ja,en;q=0.9
user-agent: Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/124.0.0.0 Safari/537.36

HTTP 200

[Asserts]
header "Content-Type" contains "text/html"
xpath "//script[@type='application/ld+json']" exists
body contains "\"@type\":\"JobPosting\""
