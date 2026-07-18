# Quirk: zero matches is HTTP 404, but the body still carries the normal
# envelope (jobs: []) plus statusCode/success/message trailer fields.
GET https://jobicy.com/api/v2/remote-jobs?count=3&tag=zzzznomatchzzz

HTTP 404
