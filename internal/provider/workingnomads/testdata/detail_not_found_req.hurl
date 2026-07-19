# Proves there is no per-job endpoint under this API namespace: appending
# a job's numeric id (as it appears in a job's outbound "url" field, e.g.
# /job/go/1734670/) to the exposed_jobs path 404s. This is why
# Client.Detail resolves an id against a fresh full-dump fetch instead of
# a dedicated request.
GET https://www.workingnomads.com/api/exposed_jobs/1734670/

HTTP 404
