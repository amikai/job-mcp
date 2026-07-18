# Quirk: an unknown tenant subdomain is NOT a 404. BambooHR 302-redirects
# to its marketing site, so a client that follows redirects lands on an
# HTML page with HTTP 200.
GET https://this-company-does-not-exist-xyz.bamboohr.com/careers/list
Accept: application/json

HTTP 302
[Asserts]
header "Location" == "https://www.bamboohr.com"
