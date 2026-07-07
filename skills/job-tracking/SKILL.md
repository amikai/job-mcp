---
name: job-tracking
description: Track saved job searches over time and report only new postings since the last run. Use when the user wants to watch for new jobs at a company or for a query, asks "anything new since last time", or wants to create, run, list, or remove saved job-search watches.
---

# Job Tracking

Persist named job searches ("watches") and, on each run, report only the
postings that were not seen before. Search execution follows the `job-search`
skill; this skill adds the state handling around it.

## State file

All state lives in `~/.openings-mcp/watchlist.md`. Create the directory and
file on first use. One section per watch:

```markdown
## watch: nvidia-go-backend
- created: 2026-07-07
- recipe: search_jobs_by_company(company=nvidia, query="golang backend", location=Taiwan)
- strategy: precise
- last-run: 2026-07-07
- seen: JR1988783, JR1990211
```

- `recipe` is the exact tool call to re-run, written as
  `tool_name(param=value, ...)`.
- `seen` is a comma-separated list of job ids this watch has already
  reported or baselined.
- Watch names are kebab-case and unique within the file.

## Operations

**Create a watch.** Run one full search first, following the `job-search`
skill — this validates that the recipe actually returns sensible results.
Show the results to the user, then write the watch section with every
returned job id in `seen` as the baseline. Nothing from the first run is
"new".

**Run watches** (one by name, or all). For each watch: re-execute the recipe
exactly as recorded, page through all results, and diff the returned job ids
against `seen`. Report only the jobs whose ids are new, in the same table
format the `job-search` skill uses. Then append the new ids to `seen` and
update `last-run`. If a run returns nothing new, say so explicitly — silence
looks like a failed run.

**List watches.** Read the file and show each watch's name, recipe, last-run
date, and seen-count.

**Remove a watch.** Delete its section from the file after confirming the
name with the user.

## Rules

- A job is new if and only if its job id is absent from `seen`. Never use
  `posted_at` to decide newness — its meaning differs across sites; treat it
  as display-only.
- Never edit a recipe silently. If a recipe errors (e.g. a filter value no
  longer exists), show the error and ask the user whether to update the
  watch.
- "Run all watches" pairs well with scheduled tasks; configuring schedules is
  outside this skill.
