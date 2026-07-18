# A board exercising the list feed's value variety: all three locationType
# codes (0 on-site, 1 remote, 2 hybrid), null location.city, and rows whose
# atsLocation carries real values instead of all-null.
GET https://curtinmaritime.bamboohr.com/careers/list
Accept: application/json

HTTP 200
