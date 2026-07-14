POST https://apis.indeed.com/graphql
Host: apis.indeed.com
content-type: application/json
indeed-api-key: 161092c2017b5bbab13edb12461a62d5a833871e7cad6d9d475304573de67ac8
accept: application/json
indeed-locale: en-US
accept-language: en-US,en;q=0.9
user-agent: Mozilla/5.0 (iPhone; CPU iPhone OS 16_6_1 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Mobile/15E148 Indeed App 193.1
indeed-app-info: appv=193.1; appid=com.indeed.jobsearch; osv=16.6.1; os=ios; dtype=phone
indeed-co: TW
{
  "query": "query GetJobData { jobSearch(what: \"software engineer\", location: {where: \"Taipei\", radius: 25, radiusUnit: MILES}, limit: 5, sort: RELEVANCE) { pageInfo { nextCursor } results { trackingKey job { key title datePublished location { countryCode admin1Code city formatted { long } } compensation { estimated { currencyCode baseSalary { unitOfWork range { ... on Range { min max } } } } baseSalary { unitOfWork range { ... on Range { min max } } } currencyCode } attributes { key label } employer { relativeCompanyPageUrl name } } } } }"
}

HTTP 200

[Asserts]
header "Content-Type" contains "application/json"
jsonpath "$.data.jobSearch.results" count > 0
