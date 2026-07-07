# Job-Search Skills Design

Date: 2026-07-07
Status: Approved design, pending implementation

## Problem

openings-mcp now exposes three families of tools: job-board tools (104, Cake,
LinkedIn), per-company tools (Google, NVIDIA, TSMC), and — since PR #87 — the
unified company-parameterized tools (`search_jobs_by_company`,
`get_filters_by_company`, `get_job_detail_by_company`) covering 258 companies.

Left to its own defaults, an LLM client uses these tools poorly:

- It treats the user's keywords as the only filter, missing jobs whose titles
  use different vocabulary for the same role (SRE / Site Reliability /
  Infrastructure Engineer).
- It never deliberately chooses between server-side filtering and fetching a
  broad result set to filter semantically itself.
- It fetches job details eagerly, burning tokens on postings that were never
  going to match.
- It does not know when to search a job board versus a company career site.

The fix is not more tools but guidance: Agent Skills that teach the client
**strategy selection**, not just tool sequencing.

## Decisions made during brainstorming

- **Deliverable**: Claude Code Agent Skills (SKILL.md) shipped in the repo,
  installed by copying; may be upgraded to a Claude Code plugin later once the
  content proves itself.
- **Coverage**: all tools — job boards, per-company tools, and the unified
  by-company tools.
- **Structure**: one main search skill plus a separate tracking skill. The
  search scenarios (single-company deep search, cross-company scan,
  profile-based matching) share one strategy-selection flow; tracking is
  separate because it reads and writes state.
- **Tracking state location**: user home directory (`~/.openings-mcp/`), since
  job hunting is not tied to any project checkout.
- **Skill style**: hybrid — a short decision tree for the two key decisions
  (venue, filter strategy), heuristic principles within each branch, plus a
  discipline section for token control.
- **Language**: skill content written in English.

## File layout

```
skills/
  job-search/SKILL.md      # main search skill
  job-tracking/SKILL.md    # tracking skill
```

README gains a "Skills (optional)" section: one-line description of each
skill, install instructions (copy into `~/.claude/skills/`), and a note that
they currently target Claude Code.

## Skill 1: `job-search`

**Trigger** (frontmatter description): the user asks to find jobs, search
openings, or check a company's postings, and openings-mcp tools are available.

### Step 0 — intent intake

Extract five things from the conversation: role/skill keywords, location,
target company (if any), hard constraints (remote, seniority), and desired
thoroughness (quick look vs exhaustive sweep). Ask at most one or two
questions, and only for the most load-bearing missing piece.

### Decision point 1 — venue

- **A specific company is named** → prefer the unified tools
  (`search_jobs_by_company`). The resolver fuzzy-matches names and suggests
  the closest supported companies on a miss. If the company is not in the
  roster, fall back to LinkedIn filtered by company name, or a dedicated
  per-company tool when one exists (Google, NVIDIA, TSMC).
- **No target company** → job boards: 104 + Cake for Taiwan-centric searches,
  LinkedIn for global ones.
- **The user wants to scan many companies** → loop over the unified tools,
  but confirm the company list and its size with the user first (token cost).

### Decision point 2 — filter strategy

- **Precise mode** (default): use it when the user's criteria map cleanly onto
  search parameters. When targeting a company site, call
  `get_filters_by_company` first, put hard constraints into structured
  filters, and keep the query parameter to role/technology terms only.
- **Broad-trawl mode**: switch when (a) title vocabulary is unreliable — the
  same role is named differently across companies, (b) the user explicitly
  asks for an exhaustive sweep, or (c) precise mode returns suspiciously few
  results. Loosen or drop the query, keep structured filters (especially
  location), page through summaries, and filter semantically by reading
  titles.
- **Escalation rule**: start precise, fall back to trawl when results look
  wrong — and tell the user which strategy is in play whenever it changes.

### Discipline rules

- Never put locations or employment types into the query string.
- When trawling, check `total_count` after page 1: above roughly 200 results
  (10 pages), narrow with structured filters or confirm with the user before
  paging on.
- Never call a get_job_detail tool during the trawl phase.
- Fetch details only when both hold: the filtered shortlist is at most ~10
  jobs, and the user's criteria require the posting body (tech stack, visa,
  remote policy).
- When results run short, fetch the next page; do not broaden the query and
  re-search.

### Presentation

A comparison table — title / company / location / posted date / URL / match
note — sorted by match quality. Close with the strategy used and the coverage
achieved (pages scanned, companies covered, anything skipped) so the user
knows where the blind spots are.

### Sub-scenario: profile-based matching

When the user provides a resume or a skills description, expand it into two or
three synonym keyword sets (SRE ↔ Site Reliability ↔ Infrastructure Engineer),
run a search round per set, deduplicate by job_id/URL, then score each result
against the profile in the final table.

## Skill 2: `job-tracking`

**State file**: `~/.openings-mcp/watchlist.md`, human-readable markdown, one
section per watch:

```markdown
## watch: nvidia-go-backend
- created: 2026-07-07
- recipe: search_jobs_by_company(company=nvidia, query="golang backend", location=Taiwan)
- strategy: precise
- last-run: 2026-07-07
- seen: JR1988783, JR1990211, ...
```

**Operations**:

- **Create watch**: run one full search via the job-search flow (validating
  the recipe), then record every returned job_id as the seen baseline.
- **Run watch** (one or all): re-execute the recipe, diff against the seen
  list, report only jobs whose IDs are new, then append the new IDs and update
  last-run.
- **List / remove watches**: read or edit the file directly.

**Newness criterion**: a job_id absent from the seen list. `posted_at` is
display-only — its semantics differ across ATSs and boards, so it is never
used to decide newness.

**Scheduling**: the skill notes that "run all watches" pairs well with Claude
Code's scheduled tasks, but configuring schedules is out of scope.

## README changes

Add a "Skills (optional)" section after the MCP install instructions: what
each skill does, the copy-to-`~/.claude/skills/` install step, and a note that
they currently target Claude Code.

## Verification

Skills are prompts, not code; there is no unit test to run. Acceptance means
executing four scenarios in Claude Code and checking that the tool-call
sequence matches the design:

1. Named company + concrete criteria → calls `get_filters_by_company` before
   searching precisely.
2. Vague role wording ("find me ops-ish jobs") → switches to broad trawl,
   pages through summaries, fetches no details.
3. Cross-company scan → confirms the company list before looping.
4. Tracking: create a watch, run it manually, confirm only new postings are
   reported.

## Out of scope

- Packaging as a Claude Code plugin (future upgrade path).
- MCP server-side prompts.
- Changes to the MCP tools themselves.
- Schedule configuration for tracking runs.
