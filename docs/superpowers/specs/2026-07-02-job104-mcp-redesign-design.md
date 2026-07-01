# job104 MCP Tool Redesign

Date: 2026-07-02
Status: Approved (design), pending implementation plan
Supersedes: the 104 sections of `2026-06-28-job-mcp-server-design.md`

## Goal

Redesign the `104_search_jobs` / `104_get_job_detail` tools in
`internal/jobmcp/job104.go` from scratch. The previous two approaches are both
rejected:

- **HEAD (alias approach):** a hand-picked 6-city enum with a second
  `taipei → Taipei` alias layer on top of `ids.go`, covering only a fraction of
  what the client supports and requiring manual maintenance of two vocabularies.
- **Deleted WIP (raw-code approach):** exposing 104's opaque codes
  (`area=6001001000`, `ro=1`) directly with a fully hand-written
  `jsonschema.Schema`, duplicating id→label knowledge in description prose and
  hand-rolled validators.

The new design exposes **human-readable labels generated from the canonical
maps in `internal/provider/job104/ids.go`**, with the JSON schema derived from
the input struct and enum lists patched in from those maps. One source of
truth per concern.

## Tool surface (unchanged)

| Tool | Purpose |
|------|---------|
| `104_search_jobs` | Keyword/filter search |
| `104_get_job_detail` | Full description for a jobNo |

Company tools and HTTP transport remain out of scope.

## `104_search_jobs` input

LLM-facing names, not 104 API names (`ro`, `order`, `s9` are internal):

```go
type job104SearchInput struct {
    Keyword string   `json:"keyword,omitempty"`
    Area    string   `json:"area,omitempty"`
    JobType string   `json:"job_type,omitempty"`
    Sort    string   `json:"sort,omitempty"`
    Remote  string   `json:"remote,omitempty"`
    Edu     []string `json:"edu,omitempty"`
    Shift   []string `json:"shift,omitempty"`
    Page    int      `json:"page,omitempty"`
}
```

| Field | Enum source | Values (labels = `ids.go` map keys) |
|-------|-------------|--------------------------------------|
| `area` | `job104.AreaIDs` | All ~77 regions: `Taipei`, `NewTaipei`, … `Tokyo`, `WestAfrica` |
| `job_type` | `job104.RoIDs` | `Full-time`, `Part-time`, `Senior`, `Dispatch` |
| `sort` | `job104.OrderIDs` | `Relevance`, `Newest` |
| `remote` | `job104.RemoteWorkIDs` | `Full`, `Partial` |
| `edu` (multi) | `job104.EduIDs` | `HighSchoolBelow` … `Doctorate` |
| `shift` (multi) | `job104.S9IDs` | `Day`, `Night`, `Graveyard`, `Holiday` |

Decisions:

- **No alias layer.** Enum values are exactly the `ids.go` map keys. Adding a
  label to `ids.go` makes it available in the tool with no further edits.
- **`keyword` is optional** — the 104 API supports filter-only browsing
  (e.g. area-only).
- **All six filters exposed.** The old spec deferred `edu`/`s9` because no
  label↔code maps existed; `EduIDs`/`S9IDs` exist now, so that restriction is
  gone.

## Schema generation

Base schema derived from the struct via `jsonschema.For[job104SearchInput]`
(field names, types, array-ness single-sourced from the struct — this is what
the SDK does itself when `InputSchema` is nil). Then enum lists are patched
onto the derived properties:

```go
p["area"].Enum     = labelEnum(job104.AreaIDs)      // sorted by code → stable output
p["job_type"].Enum = labelEnum(job104.RoIDs)
// sort/remote likewise; edu/shift patch .Items.Enum
```

`labelEnum` is a small generic helper (`map[string]T → []any` of keys, ordered
by the underlying code so schema output is deterministic and id-ordered).

Descriptions carry **semantics only** (e.g. the "soft filter — check each
result's jobRo" caveat, "multi-select OR'd together"). They must NOT restate
id=label tables — hand-copied tables are exactly how the RO/RemoteWork codes
went wrong in an earlier version.

Schema construction happens once at package init and panics on error (an
invariant of the source, not a runtime condition).

## Handler mapping

`job104ToRequest(in job104SearchInput) (job104.SearchJobsParams, error)`
translates labels to typed codes by map lookup, following the existing
`mapCodes` precedent in `tsmc.go`. Unknown label → error via `errorResult`.

The SDK already validates arguments against the input schema (including enums)
before invoking the handler, so handler-side lookup failure is
defense-in-depth plus a friendly error for direct (non-MCP) callers. There are
no hand-rolled validators and no `AllValues()` scans.

## Error handling

Unchanged: handler errors are reported to the model via the existing
`errorResult` helper (`IsError` result, not a protocol error).

## Testing

- Unit test for `job104ToRequest`: labels (including multi-select `edu`/
  `shift`) map to the expected typed `SearchJobsParams`; unknown label errors.
- Schema round-trip test over in-memory transports: list tools through a real
  client session, assert the search tool's schema exposes label enums (e.g.
  `area` contains `"Taipei"`, `job_type` contains `"Full-time"`) and does not
  expose raw codes or internal names (`ro`, `s9`).
- Existing `assertTools` registration test stays.

## Out of scope

- Raw 104 codes in the tool vocabulary.
- Hand-written full `jsonschema.Schema`.
- Alias/synonym layers on top of `ids.go` keys.
- 104 company tools; HTTP transport; other providers (tsmc.go untouched).
