#!/bin/bash
BASE="https://boards-api.greenhouse.io/v1"
curl -s "$BASE/boards/anthropic/jobs/4461450008?questions=true&pay_transparency=true" | jq . > job_detail_full_rsp.json
