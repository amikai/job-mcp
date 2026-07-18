# An unknown geo/industry slug is HTTP 400 with a small
# {"success": false, "error": "..."} body, unlike the enveloped 404 for
# zero matches.
GET https://jobicy.com/api/v2/remote-jobs?count=3&industry=not-a-real-industry

HTTP 400
