# Registry multi-match and result merging

## Problem

`ats.Registry` kept two maps (`bySlug`, `byName`) and failed startup when a
normalized slug or display name collided across adapters. This blocked the
real roster situation of one company listed on more than one ATS (e.g. a main
board plus a regional board on another provider). `Resolve` also returned a
single `(Adapter, slug)`, so even if collisions were allowed, callers could
only ever reach one of the colliding entries.

## Decision summary

- **Slug = globally unique public address; name = display alias that may
  collide.** A slug collision across any two rosters is a curation bug and
  fails startup; a display-name collision is the multi-match feature.
- Multi-match is always merged — the registry does not try to distinguish
  "same company on two ATSes" from "two companies sharing a name".
- Errors are never swallowed and never name the ATS. The unified tools'
  abstraction promise (clients never learn which ATS serves a company) holds
  in every path; the slug is the only source identifier exposed, and it is
  already public vocabulary.
- Fan-out is fail-fast for search and filters: any adapter error aborts the
  request with a `company "<slug>": ...` wrapped error through the normal
  MCP error channel. No warnings side-channel in outputs.

## Design

### 1. Registry data structure

```go
bySlug map[string]registryEntry   // key: normalize(slug); globally unique
byName map[string][]registryEntry // key: normalize(display name); collisions append
```

Build rules:

- A slug key collision — within one adapter or across adapters — fails
  startup loudly, as before this change.
- Name keys append; order under a key follows adapter registration order.
- The company count in the "unknown company" teaching error is `len(bySlug)`.

`slugs []slugEntry` (suggestion index) is unchanged.

### 2. Resolve semantics

```go
type ResolvedCompany struct {
    Adapter Adapter
    Slug    string
}

func (r *Registry) Resolve(company string) ([]ResolvedCompany, error)
```

For a normalized key, the result is the slug hit (if any) followed by the
same-key name hits, deduplicated by slug:

- `"nvidia-jp"` (slug only) → exactly that entry.
- `"NVIDIA Corp"` (shared name) → every roster listing that name.
- `"stripe"` where A is `{slug: stripe, name: Stripe}` and B is
  `{slug: stripe-jp, name: Stripe}` → A then B. A pure slug-priority rule
  was rejected here: it would capture the shared name key (slug and name
  normalize identically in the common roster shape) and the merge would
  never trigger for exactly the same-company-on-two-ATSes case this design
  exists for.
- Careers-URL fallback is unchanged and returns a single-element slice.
- Full miss → the existing teaching error. A nil error implies ≥1 match.

### 3. Tool behavior

- **search_jobs_by_company**: call `Search` on each resolved entry with the
  same params; concatenate `jobs` in entry order, sum `total_count`, take the
  max `total_pages`, pass `page` through. Any adapter error aborts the whole
  request with `company "<slug>": <err>` — the LLM can retry per slug.
- **get_filters_by_company**: single match keeps the flat `filters` map.
  Multi-match returns per-slug sections instead of a flat union, so filter
  values are never mixed across sources and the strict-validation adapters
  (e.g. Eightfold rejects a whole request over one unknown value) cannot be
  fed another source's values by construction:

  ```json
  { "sources": [
      { "company": "acme",    "filters": { "team": ["ML", "Web"] } },
      { "company": "acme-jp", "filters": { "team": ["Web", "Hardware"] } } ] }
  ```

  Fail-fast like search.
- **get_job_detail_by_company**: the job_id belongs to exactly one entry; try
  each in order, return the first success. Per-entry errors here are expected
  probes, not signal — all-fail returns them joined, each wrapped with its
  slug.

### 4. Non-goals

- No same-company-vs-collision heuristic — always merge name matches.
- No warnings field and no partial-failure tolerance for search/filters —
  errors travel the error channel or not at all.
- No adapter names in any client-visible string.
- No concurrent fan-out — multi-match is rare and usually 2 entries;
  sequential until proven slow.

### 5. Testing

- Registry: slug hit exact; shared-name multi-match ordered; slug+same-key
  name union (Stripe case) deduplicates; cross-adapter and within-adapter
  slug collisions fail startup; name collisions don't; careers URL single;
  teaching error count.
- Company tools, with two fake adapters sharing a name: search merging
  (concat / sum / max), search fail-fast with slug-prefixed error, filters
  flat when single match, per-slug sections when multi, filters fail-fast,
  detail first-success, detail all-fail joined slug-prefixed errors.
