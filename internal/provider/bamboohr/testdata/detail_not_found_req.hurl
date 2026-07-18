# An unknown job id on a real tenant 404s with a structured JSON error
# body, unlike the list endpoint's 302-to-marketing-site behavior for
# unknown tenants.
GET https://giatecscientific.bamboohr.com/careers/999999/detail
Accept: application/json

HTTP 404
