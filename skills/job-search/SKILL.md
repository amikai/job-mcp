---
name: job-search
description: Search job openings effectively with openings-mcp tools by choosing where to search (job boards vs company career sites) and how to filter (precise server-side search vs broad trawl with semantic filtering). Use when the user wants to find jobs, browse openings, check a company's postings, or match openings against their resume or skills.
---

# Job Search Strategy

Turn a job-hunting request into deliberate tool use: pick the venue, pick the
filter strategy, keep token spend under control, and present results the user
can act on. The tools come from the openings-mcp MCP server; if they are not
available, say so instead of improvising.

## Step 0: Intake

Extract from the conversation:

1. Role / skill keywords
2. Location
3. Target company, if any
4. Hard constraints (remote, seniority, employment type)
5. Thoroughness: quick look or exhaustive sweep

Ask at most one or two questions, and only when a load-bearing piece is
missing. Do not interrogate the user about details a first search round can
answer.

## Decision 1: Where to search

| Situation | Venue |
|---|---|
| A specific company is named | `search_jobs_by_company`. The resolver fuzzy-matches names and slugs; on a miss its error suggests the closest supported companies — try those before giving up. |
| Named company not in the roster | A dedicated tool if one exists (`google_*`, `nvidia_*`, `tsmc_*`); otherwise LinkedIn with the company as a filter. |
| No target company, Taiwan-centric search | 104 and Cake. |
| No target company, global search | LinkedIn. |
| Scan many companies with the same criteria | Loop `search_jobs_by_company` over the list — but confirm the company list and its size with the user first; each company costs at least one request. |

## Decision 2: How to filter

**Precise mode — the default.** Use it when the user's criteria map cleanly
onto search parameters.

- Company career sites: call `get_filters_by_company` first. Put hard
  constraints (location, job family, employment type) into `filters` using the
  exact keys and values it returned. Keep `query` to role titles, skills, and
  technologies only.
- Job boards: same principle with their dedicated parameters.

**Broad-trawl mode.** Titles lie: the same role is called SRE, Site
Reliability Engineer, Infrastructure Engineer, or DevOps depending on the
company, so a keyword query silently drops matches. Switch to trawling when
any of these holds:

- The role the user wants goes by many names.
- The user asks for an exhaustive sweep.
- Precise mode returned suspiciously few results for a company of that size.

To trawl: loosen or drop `query`, keep structured filters (especially
location), page through the summaries, and judge each title yourself against
the user's actual intent — you are the filter now.

**Escalation rule:** start precise; fall back to trawl when results look
wrong. Tell the user whenever the strategy changes and why.

## Discipline

These rules control token spend. Do not break them.

- Never put locations or employment types in the query string — dedicated
  parameters exist for them.
- When trawling, check `total_count` after page 1. Above roughly 200 results,
  narrow with structured filters or confirm with the user before paging on.
- Never call a job-detail tool during a trawl.
- Fetch details only when both hold: the filtered shortlist is at most about
  10 jobs, and the user's criteria require the posting body (tech stack, visa
  sponsorship, remote policy, on-call expectations).
- When results run short, fetch the next page. Do not broaden the query and
  re-search — that re-fetches what you already saw.

## Presenting results

Show a table sorted by match quality: title, company, location, posted date,
URL, and a short match note. Always include the URL — it is how the user
applies. Close by stating which strategy ran and the coverage achieved (pages
scanned, companies covered, anything skipped) so the user knows where the
blind spots are.

## Profile matching

When the user provides a resume or a skills description:

1. Expand it into two or three synonym keyword sets (e.g. SRE / Site
   Reliability / Infrastructure Engineer).
2. Run one search round per set, following Decisions 1 and 2 above.
3. Deduplicate results by job id or URL.
4. Score each surviving job against the profile and say why it matches or
   falls short in the match-note column.
