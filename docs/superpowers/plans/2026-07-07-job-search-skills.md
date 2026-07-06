# Job-Search Skills Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Ship two Claude Code Agent Skills (`job-search`, `job-tracking`) in a new `skills/` directory plus a README install section, teaching MCP clients search-strategy selection per `docs/superpowers/specs/2026-07-07-job-search-skills-design.md`.

**Architecture:** Pure-markdown deliverables — no Go code changes. Each skill is a self-contained `SKILL.md` with YAML frontmatter (name + description for triggering) and a body organized as: short decision tree for the two key decisions, heuristics within branches, and a discipline section for token control. Tracking state lives in `~/.openings-mcp/watchlist.md` at runtime (created by the skill when used, not by this repo).

**Tech Stack:** Claude Code Agent Skills (SKILL.md format), GitHub-flavored markdown.

## Global Constraints

- All skill and README content in English.
- **Single commit at the end** — the user explicitly asked to commit once when everything is done. Do NOT commit per task; the per-task commit steps are replaced by a final commit task.
- Skill frontmatter: exactly `name` and `description` keys; `name` must equal the directory name.
- Tool names referenced by the skills (`search_jobs_by_company`, `get_filters_by_company`, `get_job_detail_by_company`) come from PR #87, which is not yet merged into this branch. That is fine — skills are prose and have no build dependency — but the PR that ships these skills should land after #87.
- Spec: `docs/superpowers/specs/2026-07-07-job-search-skills-design.md`. Where this plan and the spec disagree, the spec wins.

---

### Task 1: Create the `job-search` skill

**Files:**
- Create: `skills/job-search/SKILL.md`

**Interfaces:**
- Consumes: nothing from other tasks.
- Produces: the skill name `job-search` and the install path convention `skills/<name>/SKILL.md`, referenced verbatim by Task 3's README section. The "Discipline" section is referenced by Task 2's create-watch flow ("run one full search following the job-search skill").

- [ ] **Step 1: Write the file**

Create `skills/job-search/SKILL.md` with exactly this content:

````markdown
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
````

- [ ] **Step 2: Verify frontmatter and location**

Run: `sed -n '1,5p' skills/job-search/SKILL.md`
Expected output — first line `---`, a `name: job-search` line matching the directory name, a `description:` line containing both what it does and "Use when", closing `---`.

Run: `grep -c 'get_filters_by_company\|search_jobs_by_company' skills/job-search/SKILL.md`
Expected: a number ≥ 3 (the unified tools are referenced in both decision sections).

- [ ] **Step 3: Check against spec**

Re-read the "Skill 1: `job-search`" section of `docs/superpowers/specs/2026-07-07-job-search-skills-design.md` and confirm every element appears in the file: intake (5 items, ≤2 questions), venue table (5 rows), two filter modes with the three trawl triggers, escalation rule with user notification, all 5 discipline rules, presentation format with coverage statement, profile matching (synonym expansion → dedupe → score). Fix any gap before moving on. No commit yet (single commit at the end).

### Task 2: Create the `job-tracking` skill

**Files:**
- Create: `skills/job-tracking/SKILL.md`

**Interfaces:**
- Consumes: the `job-search` skill name from Task 1 (the create-watch flow tells the model to follow it).
- Produces: the skill name `job-tracking`, referenced by Task 3's README section; the state-file path `~/.openings-mcp/watchlist.md`.

- [ ] **Step 1: Write the file**

Create `skills/job-tracking/SKILL.md` with exactly this content:

````markdown
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
````

- [ ] **Step 2: Verify frontmatter and location**

Run: `sed -n '1,5p' skills/job-tracking/SKILL.md`
Expected output — first line `---`, `name: job-tracking` matching the directory name, `description:` containing "Use when", closing `---`.

Run: `grep -c 'watchlist.md\|posted_at' skills/job-tracking/SKILL.md`
Expected: a number ≥ 2 (state path and the newness rule are both present).

- [ ] **Step 3: Check against spec**

Re-read the "Skill 2: `job-tracking`" section of the spec and confirm: state file path `~/.openings-mcp/watchlist.md`, the four operations (create with baseline seeding, run with seen-diff, list, remove), and the newness criterion (job id absent from seen; `posted_at` display-only). Fix any gap. No commit yet.

### Task 3: README "Skills (optional)" section

**Files:**
- Modify: `README.md` (insert between the end of "Add the MCP server to your tool", line 80, and "## Disclaimer", line 82)

**Interfaces:**
- Consumes: skill names `job-search` and `job-tracking` and their paths `skills/<name>/SKILL.md` from Tasks 1–2.
- Produces: nothing consumed by later tasks.

- [ ] **Step 1: Insert the section**

Immediately before the `## Disclaimer` heading, insert:

````markdown
## Skills (optional)

If your client is Claude Code, this repo ships two [Agent Skills](https://code.claude.com/docs/en/skills)
that teach it to use the tools well — picking the right site, choosing
between precise filtering and a broad sweep, and keeping token spend down:

- **job-search** — strategy guide for one-off searches: single-company deep
  dives, cross-company scans, and matching openings against your resume.
- **job-tracking** — saved searches that report only postings that are new
  since the last run (state lives in `~/.openings-mcp/watchlist.md`).

Install by copying into your user skills directory:

```
git clone https://github.com/amikai/openings-mcp
cp -r openings-mcp/skills/job-search openings-mcp/skills/job-tracking ~/.claude/skills/
```

Then invoke naturally ("find me backend jobs at nvidia") or explicitly with
`/job-search` and `/job-tracking`. Other MCP clients can use the SKILL.md
files as plain prompt guidance.
````

- [ ] **Step 2: Verify README structure**

Run: `grep -n '^## ' README.md`
Expected: `Skills (optional)` appears between `Add the MCP server to your tool` and `Disclaimer`; all previously existing headings unchanged.

### Task 4: Acceptance scenarios and final commit

**Files:**
- No new files. Uses `skills/job-search/SKILL.md`, `skills/job-tracking/SKILL.md` from Tasks 1–2.

**Interfaces:**
- Consumes: both skills, the spec's Verification section.
- Produces: the single final commit.

- [ ] **Step 1: Self-review both skills as a fresh reader**

Read each SKILL.md top to bottom and check: no placeholder text, no reference to tools that do not exist (valid names: `search_jobs_by_company`, `get_filters_by_company`, `get_job_detail_by_company`, `104_*`, `cake_*`, `linkedin_*`, `google_*`, `nvidia_*`, `tsmc_*` — verify against the repo's tool registrations if unsure), no contradiction with the spec, and each file's guidance is executable by a model that has never seen this conversation.

- [ ] **Step 2: Walk the four spec scenarios on paper**

For each scenario in the spec's Verification section, trace which skill sections fire and confirm the expected tool-call sequence follows from the text alone:

1. "Find Go backend jobs at NVIDIA, Taiwan, remote OK" → job-search Decision 1 row 1, precise mode → `get_filters_by_company` before `search_jobs_by_company`.
2. "Find me ops-ish jobs at nvidia" → trawl trigger (a) → loose query, paging, no detail calls.
3. "Scan Stripe, Datadog, and Cloudflare for platform roles" → Decision 1 last row → confirm list with user before looping.
4. "Watch nvidia for new golang jobs" → job-tracking create (baseline seeding), then run → only-new reporting.

If a trace requires knowledge not written in the skill, amend the skill text.

- [ ] **Step 3: Live smoke test (requires user cooperation)**

The full acceptance test — installing to `~/.claude/skills/` and running the four scenarios in a fresh Claude Code session — touches the user's home directory and needs their MCP setup, so hand it to the user rather than doing it silently. Present the install command and the four scenario prompts from Step 2.

- [ ] **Step 4: Single final commit**

```bash
git add skills/ README.md docs/superpowers/specs/2026-07-07-job-search-skills-design.md docs/superpowers/plans/2026-07-07-job-search-skills.md
git commit -m "feat: add job-search and job-tracking agent skills

Co-Authored-By: Claude Fable 5 <noreply@anthropic.com>"
```

Only run this after the user has confirmed they are ready to commit.
