#!/bin/bash
BASE="https://boards-api.greenhouse.io/v1"
curl -s "$BASE/boards/safariai/jobs" | jq . > jobs_rsp.json
