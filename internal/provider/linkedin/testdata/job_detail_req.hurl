# The guest jobs/view/{id} endpoint 999-blocks (authwall redirect) a cold
# request with no session. It works when the cookies set by a prior request
# to the same host (jobs-guest search or the public jobs/search page) are
# carried along, same as a browser would. Capture jobs_rsp.html first, then
# run this in the same cookie jar:
#   hurl --cookie-jar cookies.txt jobs_req.hurl job_detail_req.hurl
GET https://www.linkedin.com/jobs/view/4422697744
authority: www.linkedin.com
accept: text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,image/apng,*/*;q=0.8,application/signed-exchange;v=b3;q=0.7
accept-language: en-US,en;q=0.9
cache-control: max-age=0
upgrade-insecure-requests: 1
referer: https://www.linkedin.com/jobs/search?keywords=software%20engineer&location=Taiwan
user-agent: Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36
[Options]
location: true
output: job_detail_rsp.html

HTTP 200
