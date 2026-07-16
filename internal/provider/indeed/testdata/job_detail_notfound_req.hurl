POST https://apis.indeed.com/graphql
Host: apis.indeed.com
content-type: application/json
indeed-api-key: 161092c2017b5bbab13edb12461a62d5a833871e7cad6d9d475304573de67ac8
accept: application/json
indeed-locale: en-US
user-agent: Mozilla/5.0 (iPhone; CPU iPhone OS 16_6_1 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Mobile/15E148 Indeed App 193.1
indeed-app-info: appv=193.1; appid=com.indeed.jobsearch; osv=16.6.1; os=ios; dtype=phone
indeed-co: TW
```graphql
query GetJobDetail {
  jobData(jobKeys: ["0000000000000000"]) {
    results {
      job {
        key
        title
      }
    }
  }
}
```

HTTP 200

[Asserts]
header "Content-Type" contains "application/json"
jsonpath "$.data.jobData.results" count == 0
