#!/bin/bash
# Captures Ashby public job-board fixtures from real boards:
# - `browserbase` (small - 5 jobs at capture time - but exercises
#   secondaryLocations, streetAddress, compensation tiers, and null tier
#   titles)
# - `weaviate` (4 jobs at capture time; its jobs carry null isRemote and
#   null workplaceType, which the official docs claim are always present)
BASE="https://api.ashbyhq.com/posting-api/job-board/browserbase"
curl -s "$BASE" | jq . > board_rsp.json
curl -s "$BASE?includeCompensation=true" | jq . > board_comp_rsp.json
curl -s "https://api.ashbyhq.com/posting-api/job-board/weaviate?includeCompensation=true" | jq . > board_nulls_rsp.json
