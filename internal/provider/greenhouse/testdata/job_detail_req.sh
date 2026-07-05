#!/bin/bash
BASE="https://boards-api.greenhouse.io/v1"
curl -s "$BASE/boards/anthropic/jobs/4461450008" | jq . > job_detail_rsp.json
