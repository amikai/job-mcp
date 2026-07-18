# A real tenant whose hosted board has no public postings: still HTTP 200
# with totalCount 0 and an empty result array, indistinguishable from a
# company that just has nothing open.
GET https://zapier.bamboohr.com/careers/list
Accept: application/json

HTTP 200
