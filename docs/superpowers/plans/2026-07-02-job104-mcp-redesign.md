# job104 MCP Tool Redesign Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Rewrite `internal/jobmcp/job104.go` so the `104_search_jobs` tool exposes human-readable label enums generated from `internal/provider/job104/ids.go`, with the JSON schema derived from the input struct and enum lists patched in.

**Architecture:** The input struct carries LLM-facing field names (`job_type`, `sort`, `shift`). `jsonschema.For[job104SearchInput]` derives the base schema; enum lists are patched from the six canonical `ids.go` maps via a `labelEnum` helper. The handler translates labels to typed codes by map lookup (same pattern as `mapCodes` in `tsmc.go`).

**Tech Stack:** Go 1.26, `github.com/modelcontextprotocol/go-sdk/mcp`, `github.com/google/jsonschema-go/jsonschema`, ogen-generated client in `internal/provider/job104`.

**Spec:** `docs/superpowers/specs/2026-07-02-job104-mcp-redesign-design.md`

## Global Constraints

- Enum values are exactly the `ids.go` map keys (`Taipei`, `Full-time`, `Doctorate`, …) — no alias layer, no raw 104 codes in the tool vocabulary.
- Descriptions carry semantics only; never restate id=label tables.
- `keyword` is optional (all input fields are `omitempty`).
- Do not touch `tsmc.go`, `cmd/104`, or the generated `internal/provider/job104` code.
- Branch: `job104-mcp-redesign` (already checked out).
- Every commit message ends with `Co-Authored-By: Claude Fable 5 <noreply@anthropic.com>`.

---

### Task 1: Label-to-code request mapping

**Files:**
- Modify: `internal/jobmcp/job104.go` (replace struct, alias maps, and `job104ToRequest`; lines 11–83 of the current file)
- Test: `internal/jobmcp/job104_test.go` (replace `TestJob104ToRequest`)

**Interfaces:**
- Consumes: `job104.AreaIDs`, `RoIDs`, `OrderIDs`, `RemoteWorkIDs`, `EduIDs`, `S9IDs` (all `map[string]<typed code>` in `internal/provider/job104/ids.go`); ogen option constructors `job104.NewOptString`, `NewOptSearchJobsArea`, `NewOptSearchJobsRo`, `NewOptSearchJobsOrder`, `NewOptSearchJobsRemoteWork`, `NewOptInt`.
- Produces: `job104SearchInput` struct (JSON fields `keyword`, `area`, `job_type`, `sort`, `remote`, `edu`, `shift`, `page`); `job104ToRequest(job104SearchInput) (job104.SearchJobsParams, error)`; generic helpers `lookupCode[T any](field, label string, m map[string]T) (T, error)` and `lookupCodes[T any](field string, labels []string, m map[string]T) ([]T, error)`. Task 2 relies on the struct's json tags exactly as written here.

- [ ] **Step 1: Write the failing tests**

Replace `TestJob104ToRequest` in `internal/jobmcp/job104_test.go` with:

```go
func TestJob104ToRequest(t *testing.T) {
	in := job104SearchInput{
		Keyword: "golang",
		Area:    "Taipei",
		JobType: "Part-time",
		Sort:    "Newest",
		Remote:  "Full",
		Edu:     []string{"University", "Master"},
		Shift:   []string{"Day", "Holiday"},
		Page:    2,
	}
	got, err := job104ToRequest(in)
	require.NoError(t, err)

	want := job104.SearchJobsParams{
		Keyword:    job104.NewOptString("golang"),
		Area:       job104.NewOptSearchJobsArea(job104.AreaIDs["Taipei"]),
		Ro:         job104.NewOptSearchJobsRo(job104.SearchJobsRo2),
		Order:      job104.NewOptSearchJobsOrder(job104.SearchJobsOrder2),
		RemoteWork: job104.NewOptSearchJobsRemoteWork(job104.SearchJobsRemoteWork1),
		Page:       job104.NewOptInt(2),
		Edu:        []job104.SearchJobsEduItem{job104.SearchJobsEduItem4, job104.SearchJobsEduItem5},
		S9:         []job104.SearchJobsS9Item{job104.SearchJobsS9Item1, job104.SearchJobsS9Item8},
	}
	assert.Equal(t, want, got)
}

func TestJob104ToRequestEmpty(t *testing.T) {
	got, err := job104ToRequest(job104SearchInput{})
	require.NoError(t, err)
	assert.Equal(t, job104.SearchJobsParams{}, got)
}

func TestJob104ToRequestInvalidLabels(t *testing.T) {
	cases := []struct {
		name string
		in   job104SearchInput
		want string
	}{
		{"area", job104SearchInput{Area: "Mars"}, `invalid area "Mars"`},
		{"job_type", job104SearchInput{JobType: "full"}, `invalid job_type "full"`},
		{"sort", job104SearchInput{Sort: "newest"}, `invalid sort "newest"`},
		{"remote", job104SearchInput{Remote: "hybrid"}, `invalid remote "hybrid"`},
		{"edu", job104SearchInput{Edu: []string{"University", "PhD"}}, `invalid edu "PhD"`},
		{"shift", job104SearchInput{Shift: []string{"Midnight"}}, `invalid shift "Midnight"`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := job104ToRequest(tc.in)
			require.Error(t, err)
			assert.Contains(t, err.Error(), tc.want)
		})
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/jobmcp/ -run 'TestJob104ToRequest' -v`
Expected: compile error — `unknown field Edu in struct literal` (the old struct has no `Edu`/`Shift` fields). Compile failure counts as the failing state.

- [ ] **Step 3: Write the implementation**

In `internal/jobmcp/job104.go`, replace the struct, the three alias-map var blocks (`job104AreaAliases`, `job104JobTypeAliases`/`job104SortAliases`/`job104RemoteAliases`), and `job104ToRequest` with:

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

// lookupCode translates one human label to its typed code, erroring with the
// field name on unknown labels.
func lookupCode[T any](field, label string, m map[string]T) (T, error) {
	code, ok := m[label]
	if !ok {
		var zero T
		return zero, fmt.Errorf("invalid %s %q", field, label)
	}
	return code, nil
}

// lookupCodes is lookupCode over a multi-select field.
func lookupCodes[T any](field string, labels []string, m map[string]T) ([]T, error) {
	if len(labels) == 0 {
		return nil, nil
	}
	out := make([]T, 0, len(labels))
	for _, label := range labels {
		code, err := lookupCode(field, label, m)
		if err != nil {
			return nil, err
		}
		out = append(out, code)
	}
	return out, nil
}

func job104ToRequest(in job104SearchInput) (job104.SearchJobsParams, error) {
	var params job104.SearchJobsParams
	if in.Keyword != "" {
		params.Keyword = job104.NewOptString(in.Keyword)
	}
	if in.Area != "" {
		code, err := lookupCode("area", in.Area, job104.AreaIDs)
		if err != nil {
			return params, err
		}
		params.Area = job104.NewOptSearchJobsArea(code)
	}
	if in.JobType != "" {
		code, err := lookupCode("job_type", in.JobType, job104.RoIDs)
		if err != nil {
			return params, err
		}
		params.Ro = job104.NewOptSearchJobsRo(code)
	}
	if in.Sort != "" {
		code, err := lookupCode("sort", in.Sort, job104.OrderIDs)
		if err != nil {
			return params, err
		}
		params.Order = job104.NewOptSearchJobsOrder(code)
	}
	if in.Remote != "" {
		code, err := lookupCode("remote", in.Remote, job104.RemoteWorkIDs)
		if err != nil {
			return params, err
		}
		params.RemoteWork = job104.NewOptSearchJobsRemoteWork(code)
	}
	var err error
	if params.Edu, err = lookupCodes("edu", in.Edu, job104.EduIDs); err != nil {
		return params, err
	}
	if params.S9, err = lookupCodes("shift", in.Shift, job104.S9IDs); err != nil {
		return params, err
	}
	if in.Page > 0 {
		params.Page = job104.NewOptInt(in.Page)
	}
	return params, nil
}
```

Leave `job104DetailInput`, `RegisterJob104`, and the imports otherwise untouched (drop imports that become unused; `fmt` stays).

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/jobmcp/ -run 'TestJob104ToRequest' -v`
Expected: PASS (all three test functions, six invalid-label subtests).

Note: `TestRegisterJob104` still passes because `RegisterJob104` is unchanged; the SDK derives a temporary schema from the new struct until Task 2 supplies the explicit one.

- [ ] **Step 5: Commit**

```bash
git add internal/jobmcp/job104.go internal/jobmcp/job104_test.go
git commit -m "feat(104): label-based search input mapped via ids.go

Co-Authored-By: Claude Fable 5 <noreply@anthropic.com>"
```

---

### Task 2: Schema generation from struct + ids.go enums

**Files:**
- Modify: `internal/jobmcp/job104.go` (add `labelEnum`, `job104SearchSchema`; set `InputSchema` on the `104_search_jobs` tool in `RegisterJob104`)
- Test: `internal/jobmcp/job104_test.go` (add `TestJob104SearchJobsSchema`)

**Interfaces:**
- Consumes: `job104SearchInput` and its json tags from Task 1; the six `ids.go` maps; `jsonschema.For[T](*jsonschema.ForOptions) (*jsonschema.Schema, error)`; `jsonschema.Ptr`.
- Produces: package var `job104SearchInputSchema *jsonschema.Schema`; helper `labelEnum[T cmp.Ordered](m map[string]T) []any` (labels ordered by underlying code).

- [ ] **Step 1: Write the failing test**

Add to `internal/jobmcp/job104_test.go` (add `"context"` to imports):

```go
func TestJob104SearchJobsSchema(t *testing.T) {
	ctx := context.Background()
	server := mcp.NewServer(&mcp.Implementation{Name: "test", Version: "v0"}, nil)
	client, err := job104.NewClient("https://www.104.com.tw")
	require.NoError(t, err)
	RegisterJob104(server, client)

	serverTransport, clientTransport := mcp.NewInMemoryTransports()
	serverSession, err := server.Connect(ctx, serverTransport, nil)
	require.NoError(t, err)
	defer serverSession.Close()

	mcpClient := mcp.NewClient(&mcp.Implementation{Name: "test-client", Version: "v0"}, nil)
	clientSession, err := mcpClient.Connect(ctx, clientTransport, nil)
	require.NoError(t, err)
	defer clientSession.Close()

	res, err := clientSession.ListTools(ctx, nil)
	require.NoError(t, err)

	var searchTool *mcp.Tool
	for _, tool := range res.Tools {
		if tool.Name == "104_search_jobs" {
			searchTool = tool
			break
		}
	}
	require.NotNil(t, searchTool)

	schema, ok := searchTool.InputSchema.(map[string]any)
	require.True(t, ok)
	props, ok := schema["properties"].(map[string]any)
	require.True(t, ok)

	// LLM-facing names only — no 104 API names.
	for _, field := range []string{"keyword", "area", "job_type", "sort", "remote", "edu", "shift", "page"} {
		assert.Contains(t, props, field)
	}
	for _, field := range []string{"ro", "order", "remoteWork", "s9"} {
		assert.NotContains(t, props, field)
	}

	// Label enums, not raw codes.
	area := props["area"].(map[string]any)
	assert.Contains(t, area["enum"], "Taipei")
	assert.NotContains(t, area["enum"], "6001001000")
	assert.Len(t, area["enum"], len(job104.AreaIDs))

	jobType := props["job_type"].(map[string]any)
	assert.Equal(t, []any{"Full-time", "Part-time", "Senior", "Dispatch"}, jobType["enum"])

	sort := props["sort"].(map[string]any)
	assert.Equal(t, []any{"Relevance", "Newest"}, sort["enum"])

	remote := props["remote"].(map[string]any)
	assert.Equal(t, []any{"Full", "Partial"}, remote["enum"])

	edu := props["edu"].(map[string]any)
	eduItems := edu["items"].(map[string]any)
	assert.Equal(t, []any{"HighSchoolBelow", "HighSchool", "College", "University", "Master", "Doctorate"}, eduItems["enum"])

	shift := props["shift"].(map[string]any)
	shiftItems := shift["items"].(map[string]any)
	assert.Equal(t, []any{"Day", "Night", "Graveyard", "Holiday"}, shiftItems["enum"])
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/jobmcp/ -run TestJob104SearchJobsSchema -v`
Expected: FAIL — the SDK's auto-derived schema has the right property names but no `enum` on `area` (assertion `Contains area["enum"] "Taipei"` fails with enum being nil).

- [ ] **Step 3: Write the implementation**

In `internal/jobmcp/job104.go`, add `"cmp"` and `"slices"` to imports plus `"github.com/google/jsonschema-go/jsonschema"`, then add:

```go
// labelEnum lists m's labels ordered by their underlying code, so the
// generated schema is deterministic and follows 104's id order.
func labelEnum[T cmp.Ordered](m map[string]T) []any {
	labels := make([]string, 0, len(m))
	for label := range m {
		labels = append(labels, label)
	}
	slices.SortFunc(labels, func(a, b string) int { return cmp.Compare(m[a], m[b]) })
	out := make([]any, len(labels))
	for i, label := range labels {
		out[i] = label
	}
	return out
}

// job104SearchInputSchema is derived from job104SearchInput (field names and
// types single-sourced from the struct), with enum labels patched in from the
// canonical ids.go maps — descriptions carry semantics only, never id=label
// tables (hand-copied tables are how the RO/RemoteWork codes once went wrong).
var job104SearchInputSchema = job104SearchSchema()

func job104SearchSchema() *jsonschema.Schema {
	schema, err := jsonschema.For[job104SearchInput](nil)
	if err != nil {
		panic(err)
	}
	p := schema.Properties
	p["keyword"].Description = "Free-text keyword search."
	p["area"].Description = "City/region filter."
	p["area"].Enum = labelEnum(job104.AreaIDs)
	p["job_type"].Description = "Employment basis. Soft filter — verify each result's jobRo."
	p["job_type"].Enum = labelEnum(job104.RoIDs)
	p["sort"].Description = "Result order."
	p["sort"].Enum = labelEnum(job104.OrderIDs)
	p["remote"].Description = "Remote work. Soft filter — verify each result's remoteWorkType. Omit for on-site."
	p["remote"].Enum = labelEnum(job104.RemoteWorkIDs)
	p["edu"].Description = "Education levels, OR'd together."
	p["edu"].Items.Enum = labelEnum(job104.EduIDs)
	p["shift"].Description = "Shift types, OR'd together."
	p["shift"].Items.Enum = labelEnum(job104.S9IDs)
	p["page"].Description = "1-based page number."
	p["page"].Minimum = jsonschema.Ptr(1.0)
	return schema
}
```

In `RegisterJob104`, set the schema on the search tool:

```go
	mcp.AddTool(s, &mcp.Tool{
		Name:        "104_search_jobs",
		Description: "Search jobs on 104 (Taiwan's largest job board) by keyword, with optional area/job-type/remote/education/shift/sort filters.",
		InputSchema: job104SearchInputSchema,
	}, func(ctx context.Context, _ *mcp.CallToolRequest, in job104SearchInput) (*mcp.CallToolResult, any, error) {
```

(only the `Description` and added `InputSchema` line change; handler body stays).

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/jobmcp/ -run TestJob104SearchJobsSchema -v`
Expected: PASS.

- [ ] **Step 5: Run the full package and repo tests**

Run: `go test ./internal/jobmcp/ && go vet ./... && go test ./...`
Expected: all `ok` / clean. If `TestRegisterJob104` fails on schema resolution, the struct/schema mismatch is in the json tags — fix the tag, not the test.

- [ ] **Step 6: Commit**

```bash
git add internal/jobmcp/job104.go internal/jobmcp/job104_test.go
git commit -m "feat(104): derive search tool schema from struct, enums from ids.go

Co-Authored-By: Claude Fable 5 <noreply@anthropic.com>"
```

---

### Task 3: Verify against a live-ish smoke check and finish the branch

**Files:**
- No code changes expected; verification only.

**Interfaces:**
- Consumes: the finished `job104.go` from Tasks 1–2, `cmd/jobmcp` server binary.

- [ ] **Step 1: Build and list tools over stdio**

Run:
```bash
go build -o /tmp/jobmcp-smoke ./cmd/jobmcp && printf '%s\n%s\n' \
  '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-06-18","capabilities":{},"clientInfo":{"name":"smoke","version":"0"}}}' \
  '{"jsonrpc":"2.0","id":2,"method":"tools/list"}' | /tmp/jobmcp-smoke | head -c 2000
```
Expected: JSON responses; the `104_search_jobs` schema shows `"enum":["Taipei",...]` under `area` and fields `job_type`/`shift` present.

- [ ] **Step 2: Full suite once more**

Run: `go test ./... && go vet ./...`
Expected: all `ok`, vet clean.

- [ ] **Step 3: Use superpowers:finishing-a-development-branch**

Implementation complete — invoke that skill to decide merge/PR/cleanup with the user.
