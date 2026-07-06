# LinkedIn MCP Provider Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

> **Historical note — the shipped code deviates from this plan.** The plan
> was executed as written, then three follow-up refactors changed the final
> shape; the code snippets below are the original drafts, not the shipped
> versions. The deviations: `distance` and `easy_apply` are not exposed
> (Distance was removed from `JobsRequest` entirely as unverified);
> `company_ids` is a JSON array, not a comma-separated string;
> `looks_remote` was renamed back to `remote` to match the provider's field
> name; the detail output gained a `url` field; and the tool/schema
> descriptions were trimmed, with rate-limit remedy text moved into the
> client's 429/999 error strings. The design spec
> (`docs/superpowers/specs/2026-07-06-linkedin-mcp-provider-design.md`) is
> kept in sync with the shipped code; trust it over the snippets here.

**Goal:** Wire the existing `internal/provider/linkedin` package into the `openings-mcp` MCP server as `linkedin_search_jobs` and `linkedin_get_job_detail` tools.

**Architecture:** Add `internal/openingsmcp/linkedin.go` following the exact pattern already used for tsmc/nvidia/etc. (hand-written JSON input schema with label enums, MCP↔HTTP conversion functions, `RegisterLinkedin`), then wire a shared `linkedin.Client` into `cmd/openings-mcp/main.go`.

**Tech Stack:** Go, `github.com/modelcontextprotocol/go-sdk/mcp`, `github.com/stretchr/testify`.

## Global Constraints

- Tool names: `linkedin_search_jobs`, `linkedin_get_job_detail` (provider-prefix + `_search_jobs`/`_get_job_detail`, matching every other provider).
- No field in the search input schema is `required` — LinkedIn's search accepts an entirely empty query.
- `start` is exposed as LinkedIn's own raw zero-based offset (not abstracted into a 1-based `page`), per the approved design.
- Both tool descriptions must state the rate-limit caution: a session is typically cut off around the 10th consecutive search request with a plain HTTP 429 carrying no `Retry-After` hint.
- No typed upstream error (unlike nvidia/job104's openapi-generated clients) — `linkedin.Client` errors are already descriptive strings and pass straight through `errorResult(err)`, same as tsmc/google.
- No client-side rate-limit throttling/backoff added to `linkedin.Client` itself — out of scope per the design.
- `looks_remote` in both outputs must carry a `jsonschema` description flagging it as a keyword heuristic, not a native LinkedIn field.
- `CompanyLogo` from `linkedin.JobDetailResponse` is dropped from the MCP detail output (no other provider's detail output carries a logo field).

---

### Task 1: Search input schema and MCP→HTTP request conversion

**Files:**
- Create: `internal/openingsmcp/linkedin.go`
- Test: `internal/openingsmcp/linkedin_test.go`

**Interfaces:**
- Consumes: `linkedin.JobsRequest` (fields `Keywords, Location, Distance, WorkplaceType, JobType, EasyApply, CompanyIDs []string, PostedWithinSeconds, Start`, all in `internal/provider/linkedin/client.go`), `linkedin.WorkplaceTypeIDs`, `linkedin.JobTypeIDs` (both `map[string]string`, in `internal/provider/linkedin/ids.go`), `mustSchema` (in `internal/openingsmcp/helper.go`).
- Produces: `linkedinSearchInputSchema *jsonschema.Schema`, `type linkedinSearchInput struct{...}`, `func linkedinMCPToHTTPRequest(in *linkedinSearchInput) (*linkedin.JobsRequest, error)` — task 3 registers this against the tool, task 2 does not use it.

- [ ] **Step 1: Write the failing tests**

Create `internal/openingsmcp/linkedin_test.go`:

```go
package openingsmcp

import (
	"testing"

	"github.com/amikai/openings-mcp/internal/provider/linkedin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLinkedinMCPToHTTPRequest(t *testing.T) {
	in := linkedinSearchInput{
		Keyword:       "software engineer",
		Location:      "Taiwan",
		Distance:      25,
		WorkplaceType: "Remote",
		JobType:       "Full-time",
		EasyApply:     true,
		CompanyIDs:    "1441, 162479",
		PostedWithin:  "Past week",
		Start:         10,
	}
	got, err := linkedinMCPToHTTPRequest(&in)
	require.NoError(t, err)

	want := &linkedin.JobsRequest{
		Keywords:            "software engineer",
		Location:            "Taiwan",
		Distance:            25,
		WorkplaceType:       linkedin.WorkplaceRemote,
		JobType:             linkedin.JobTypeFullTime,
		EasyApply:           true,
		CompanyIDs:          []string{"1441", "162479"},
		PostedWithinSeconds: 604800,
		Start:               10,
	}
	assert.Equal(t, want, got)
}

func TestLinkedinMCPToHTTPRequestMinimal(t *testing.T) {
	got, err := linkedinMCPToHTTPRequest(&linkedinSearchInput{})
	require.NoError(t, err)
	assert.Equal(t, &linkedin.JobsRequest{}, got)
}

func TestLinkedinMCPToHTTPRequestInvalidLabels(t *testing.T) {
	cases := []struct {
		name string
		in   linkedinSearchInput
		want string
	}{
		{"workplace_type", linkedinSearchInput{WorkplaceType: "Space"}, `invalid workplace_type "Space"`},
		{"job_type", linkedinSearchInput{JobType: "Volunteer"}, `invalid job_type "Volunteer"`},
		{"posted_within", linkedinSearchInput{PostedWithin: "Past year"}, `invalid posted_within "Past year"`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := linkedinMCPToHTTPRequest(&tc.in)
			require.Error(t, err)
			assert.Contains(t, err.Error(), tc.want)
		})
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/openingsmcp/... -run TestLinkedinMCPToHTTPRequest -v`
Expected: FAIL — `linkedinSearchInput`, `linkedinMCPToHTTPRequest` undefined (package doesn't compile yet).

- [ ] **Step 3: Write the implementation**

Create `internal/openingsmcp/linkedin.go`:

```go
package openingsmcp

import (
	"fmt"
	"strings"

	"github.com/amikai/openings-mcp/internal/provider/linkedin"
)

var linkedinSearchInputRawSchema = []byte(`{
	"type": "object",
	"properties": {
		"keyword": {
			"type": "string",
			"description": "Free-text search query matched against job title, company, and description."
		},
		"location": {
			"type": "string",
			"description": "Free-text location filter. LinkedIn searches globally; there is no separate country-code parameter."
		},
		"distance": {
			"type": "integer",
			"description": "Search radius in miles around location.",
			"minimum": 0
		},
		"workplace_type": {
			"type": "string",
			"description": "Workplace type filter.",
			"enum": ["On-site", "Remote", "Hybrid"]
		},
		"job_type": {
			"type": "string",
			"description": "Job type filter.",
			"enum": ["Full-time", "Part-time", "Contract", "Temporary", "Internship"]
		},
		"easy_apply": {
			"type": "boolean",
			"description": "Only jobs with LinkedIn Easy Apply."
		},
		"company_ids": {
			"type": "string",
			"description": "Comma-separated LinkedIn numeric company IDs. IDs are opaque and must be resolved from a company's public page or a prior search response, not guessed."
		},
		"posted_within": {
			"type": "string",
			"description": "Only jobs posted within this window.",
			"enum": ["Past day", "Past week", "Past month"]
		},
		"start": {
			"type": "integer",
			"description": "Zero-based result offset; default 0. The endpoint always returns exactly 10 cards per call regardless of this value, so paging through results must increment start by exactly 10 each call (0, 10, 20, ...) to avoid gaps. Do not mimic a real browser's 25-per-step scroll traffic, which skips 10 of every 25 positions this endpoint can return.",
			"minimum": 0
		}
	},
	"additionalProperties": false
}`)

// linkedinSearchInputSchema is hand-written JSON kept aligned with
// openapi.yaml's searchJobs parameters: human labels instead of the site's
// raw form-field codes (workplace_type/job_type map back via ids.go;
// posted_within maps back via linkedinPostedWithinSeconds below).
var linkedinSearchInputSchema = mustSchema(linkedinSearchInputRawSchema)

type linkedinSearchInput struct {
	Keyword       string `json:"keyword,omitempty"`
	Location      string `json:"location,omitempty"`
	Distance      int    `json:"distance,omitempty"`
	WorkplaceType string `json:"workplace_type,omitempty"`
	JobType       string `json:"job_type,omitempty"`
	EasyApply     bool   `json:"easy_apply,omitempty"`
	CompanyIDs    string `json:"company_ids,omitempty"`
	PostedWithin  string `json:"posted_within,omitempty"`
	Start         int    `json:"start,omitempty"`
}

// linkedinPostedWithinSeconds maps a human label to the seconds value
// linkedin.JobsRequest.PostedWithinSeconds expects (f_TPR=r{n} on the wire).
var linkedinPostedWithinSeconds = map[string]int{
	"Past day":   86400,
	"Past week":  604800,
	"Past month": 2592000,
}

func linkedinMCPToHTTPRequest(in *linkedinSearchInput) (*linkedin.JobsRequest, error) {
	req := &linkedin.JobsRequest{
		Keywords:  in.Keyword,
		Location:  in.Location,
		Distance:  in.Distance,
		EasyApply: in.EasyApply,
		Start:     in.Start,
	}

	if in.WorkplaceType != "" {
		id, ok := linkedin.WorkplaceTypeIDs[in.WorkplaceType]
		if !ok {
			return nil, fmt.Errorf("invalid workplace_type %q", in.WorkplaceType)
		}
		req.WorkplaceType = id
	}

	if in.JobType != "" {
		id, ok := linkedin.JobTypeIDs[in.JobType]
		if !ok {
			return nil, fmt.Errorf("invalid job_type %q", in.JobType)
		}
		req.JobType = id
	}

	if in.CompanyIDs != "" {
		for _, id := range strings.Split(in.CompanyIDs, ",") {
			if id = strings.TrimSpace(id); id != "" {
				req.CompanyIDs = append(req.CompanyIDs, id)
			}
		}
	}

	if in.PostedWithin != "" {
		seconds, ok := linkedinPostedWithinSeconds[in.PostedWithin]
		if !ok {
			return nil, fmt.Errorf("invalid posted_within %q", in.PostedWithin)
		}
		req.PostedWithinSeconds = seconds
	}

	return req, nil
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/openingsmcp/... -run TestLinkedinMCPToHTTPRequest -v`
Expected: PASS (all three test functions).

- [ ] **Step 5: Commit**

```bash
git add internal/openingsmcp/linkedin.go internal/openingsmcp/linkedin_test.go
git commit -m "feat(openingsmcp): add LinkedIn search input schema and request conversion"
```

---

### Task 2: Response/detail types and MCP output conversion

**Files:**
- Modify: `internal/openingsmcp/linkedin.go` (append)
- Modify: `internal/openingsmcp/linkedin_test.go` (append)

**Interfaces:**
- Consumes: `linkedin.JobsResponse{Jobs []linkedin.Job}`, `linkedin.Job{ID, Title, Company, CompanyURL, Location, PostedDate string; Remote bool}`, `linkedin.JobDetailResponse{ID, Title, Company, Location, Posted, SeniorityLevel, EmploymentType, JobFunction, Industries, Description, CompanyLogo, ApplyURL string; Remote bool}` (all in `internal/provider/linkedin/client.go`).
- Produces: `type linkedinSearchOutput struct{ Data []linkedinJobSummary }`, `type linkedinJobSummary struct{...}`, `func linkedinJobURL(id string) string`, `func linkedinHTTPToMCPResponse(resp *linkedin.JobsResponse) *linkedinSearchOutput`, `type linkedinDetailInput struct{ JobID string }`, `type linkedinDetailOutput struct{...}`, `func linkedinHTTPToMCPDetail(detail *linkedin.JobDetailResponse) *linkedinDetailOutput` — task 3 registers these against the two tools.

- [ ] **Step 1: Write the failing tests**

Append to `internal/openingsmcp/linkedin_test.go`:

```go
func TestLinkedinHTTPToMCPResponse(t *testing.T) {
	in := linkedin.JobsResponse{
		Jobs: []linkedin.Job{
			{ID: "1", Title: "t1", Company: "c1", CompanyURL: "cu1", Location: "l1", PostedDate: "d1", Remote: true},
			{ID: "2", Title: "t2"},
		},
	}
	got := linkedinHTTPToMCPResponse(&in)

	want := &linkedinSearchOutput{
		Data: []linkedinJobSummary{
			{ID: "1", Title: "t1", Company: "c1", CompanyURL: "cu1", Location: "l1", PostedDate: "d1", LooksRemote: true, URL: "https://www.linkedin.com/jobs/view/1"},
			{ID: "2", Title: "t2", URL: "https://www.linkedin.com/jobs/view/2"},
		},
	}
	assert.Equal(t, want, got)
}

func TestLinkedinHTTPToMCPDetail(t *testing.T) {
	in := linkedin.JobDetailResponse{
		ID:             "7",
		Title:          "t",
		Company:        "c",
		Location:       "l",
		Posted:         "p",
		SeniorityLevel: "sl",
		EmploymentType: "et",
		JobFunction:    "jf",
		Industries:     "ind",
		Description:    "desc",
		CompanyLogo:    "logo-url",
		ApplyURL:       "apply-url",
		Remote:         true,
	}
	got := linkedinHTTPToMCPDetail(&in)

	// CompanyLogo has no corresponding output field: it's intentionally
	// dropped, so it must not appear anywhere in want.
	want := &linkedinDetailOutput{
		ID:             "7",
		Title:          "t",
		Company:        "c",
		Location:       "l",
		Posted:         "p",
		SeniorityLevel: "sl",
		EmploymentType: "et",
		JobFunction:    "jf",
		Industries:     "ind",
		Description:    "desc",
		ApplyURL:       "apply-url",
		LooksRemote:    true,
	}
	assert.Equal(t, want, got)
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/openingsmcp/... -run TestLinkedinHTTPToMCP -v`
Expected: FAIL — `linkedinSearchOutput`, `linkedinJobSummary`, `linkedinHTTPToMCPResponse`, `linkedinDetailOutput`, `linkedinHTTPToMCPDetail` undefined.

- [ ] **Step 3: Write the implementation**

Append to `internal/openingsmcp/linkedin.go`:

```go
type linkedinSearchOutput struct {
	Data []linkedinJobSummary `json:"data"`
}

type linkedinJobSummary struct {
	ID          string `json:"id" jsonschema:"Numeric LinkedIn job ID; pass to linkedin_get_job_detail's job_id param."`
	Title       string `json:"title"`
	Company     string `json:"company,omitempty"`
	CompanyURL  string `json:"company_url,omitempty"`
	Location    string `json:"location,omitempty"`
	PostedDate  string `json:"posted_date,omitempty"`
	LooksRemote bool   `json:"looks_remote,omitempty" jsonschema:"Keyword heuristic (title/location substring match for 'remote'/'work from home'/'wfh'), not a field LinkedIn provides. False does not mean confirmed on-site."`
	URL         string `json:"url,omitempty" jsonschema:"Public job posting URL."`
}

func linkedinJobURL(id string) string {
	if id == "" {
		return ""
	}
	return "https://www.linkedin.com/jobs/view/" + id
}

func linkedinHTTPToMCPResponse(resp *linkedin.JobsResponse) *linkedinSearchOutput {
	out := &linkedinSearchOutput{Data: make([]linkedinJobSummary, 0, len(resp.Jobs))}
	for _, j := range resp.Jobs {
		out.Data = append(out.Data, linkedinJobSummary{
			ID:          j.ID,
			Title:       j.Title,
			Company:     j.Company,
			CompanyURL:  j.CompanyURL,
			Location:    j.Location,
			PostedDate:  j.PostedDate,
			LooksRemote: j.Remote,
			URL:         linkedinJobURL(j.ID),
		})
	}
	return out
}

type linkedinDetailInput struct {
	JobID string `json:"job_id" jsonschema:"Numeric LinkedIn job ID (id from linkedin_search_jobs results, e.g. 4422697744)."`
}

type linkedinDetailOutput struct {
	ID             string `json:"id"`
	Title          string `json:"title"`
	Company        string `json:"company,omitempty"`
	Location       string `json:"location,omitempty"`
	Posted         string `json:"posted,omitempty"`
	SeniorityLevel string `json:"seniority_level,omitempty"`
	EmploymentType string `json:"employment_type,omitempty"`
	JobFunction    string `json:"job_function,omitempty"`
	Industries     string `json:"industries,omitempty"`
	Description    string `json:"description,omitempty" jsonschema:"Full job description as plain text."`
	ApplyURL       string `json:"apply_url,omitempty" jsonschema:"External ATS apply URL; absent for LinkedIn Easy Apply postings."`
	LooksRemote    bool   `json:"looks_remote,omitempty" jsonschema:"Keyword heuristic over title/location only (not the full description), not a field LinkedIn provides. False does not mean confirmed on-site."`
}

func linkedinHTTPToMCPDetail(detail *linkedin.JobDetailResponse) *linkedinDetailOutput {
	return &linkedinDetailOutput{
		ID:             detail.ID,
		Title:          detail.Title,
		Company:        detail.Company,
		Location:       detail.Location,
		Posted:         detail.Posted,
		SeniorityLevel: detail.SeniorityLevel,
		EmploymentType: detail.EmploymentType,
		JobFunction:    detail.JobFunction,
		Industries:     detail.Industries,
		Description:    detail.Description,
		ApplyURL:       detail.ApplyURL,
		LooksRemote:    detail.Remote,
	}
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/openingsmcp/... -run TestLinkedinHTTPToMCP -v`
Expected: PASS (both test functions).

- [ ] **Step 5: Commit**

```bash
git add internal/openingsmcp/linkedin.go internal/openingsmcp/linkedin_test.go
git commit -m "feat(openingsmcp): add LinkedIn response/detail conversion"
```

---

### Task 3: Register the tools and end-to-end test coverage

**Files:**
- Modify: `internal/openingsmcp/linkedin.go` (append)
- Modify: `internal/openingsmcp/linkedin_test.go` (append)

**Interfaces:**
- Consumes: `linkedin.Client` with `Jobs(ctx, *JobsRequest) (*JobsResponse, error)` and `JobDetail(ctx, jobID string) (*JobDetailResponse, error)` (in `internal/provider/linkedin/client.go`); `linkedin.NewMockServer() *httptest.Server` (in `internal/provider/linkedin/mocksrv.go`); `errorResult(err error) *mcp.CallToolResult` (in `internal/openingsmcp/openingsmcp.go`); `assertTools(t, server, names...)` (in `internal/openingsmcp/openingsmcp_test.go`); `findTool(tools, name)` (in `internal/openingsmcp/job104_test.go`); everything from Tasks 1–2.
- Produces: `func RegisterLinkedin(s *mcp.Server, c *linkedin.Client)` — Task 4 calls this from `cmd/openings-mcp/main.go`.

- [ ] **Step 1: Write the failing tests**

Append to `internal/openingsmcp/linkedin_test.go` (add `"context"`, `"encoding/json"`, `"github.com/modelcontextprotocol/go-sdk/mcp"` to imports):

```go
func testLinkedinMCPClientServer(t *testing.T) (*mcp.ClientSession, *mcp.ServerSession) {
	server := mcp.NewServer(&mcp.Implementation{Name: "test", Version: "v0"}, nil)
	srv := linkedin.NewMockServer()
	t.Cleanup(srv.Close)
	client := linkedin.NewClient(srv.URL, srv.Client())
	RegisterLinkedin(server, client)

	serverTransport, clientTransport := mcp.NewInMemoryTransports()
	serverSession, err := server.Connect(t.Context(), serverTransport, nil)
	require.NoError(t, err)
	t.Cleanup(func() {
		serverSession.Close()
	})

	mcpClient := mcp.NewClient(&mcp.Implementation{Name: "test-client", Version: "v0"}, nil)
	clientSession, err := mcpClient.Connect(t.Context(), clientTransport, nil)
	require.NoError(t, err)
	t.Cleanup(func() {
		clientSession.Close()
	})
	return clientSession, serverSession
}

func TestRegisterLinkedin(t *testing.T) {
	server := mcp.NewServer(&mcp.Implementation{Name: "test", Version: "v0"}, nil)

	client := linkedin.NewClient("https://www.linkedin.com", nil)
	RegisterLinkedin(server, client)

	assertTools(t, server, "linkedin_search_jobs", "linkedin_get_job_detail")
}

func TestLinkedinSearchJobsE2E(t *testing.T) {
	clientSession, _ := testLinkedinMCPClientServer(t)

	res, err := clientSession.ListTools(t.Context(), nil)
	require.NoError(t, err)

	tool := findTool(res.Tools, "linkedin_search_jobs")
	require.NotNil(t, tool)

	schema, ok := tool.InputSchema.(map[string]any)
	require.True(t, ok)

	want := map[string]any{
		"type": "object",
		"properties": map[string]any{
			"keyword": map[string]any{
				"type":        "string",
				"description": "Free-text search query matched against job title, company, and description.",
			},
			"location": map[string]any{
				"type":        "string",
				"description": "Free-text location filter. LinkedIn searches globally; there is no separate country-code parameter.",
			},
			"distance": map[string]any{
				"type":        "integer",
				"description": "Search radius in miles around location.",
				"minimum":     float64(0),
			},
			"workplace_type": map[string]any{
				"type":        "string",
				"description": "Workplace type filter.",
				"enum":        []any{"On-site", "Remote", "Hybrid"},
			},
			"job_type": map[string]any{
				"type":        "string",
				"description": "Job type filter.",
				"enum":        []any{"Full-time", "Part-time", "Contract", "Temporary", "Internship"},
			},
			"easy_apply": map[string]any{
				"type":        "boolean",
				"description": "Only jobs with LinkedIn Easy Apply.",
			},
			"company_ids": map[string]any{
				"type":        "string",
				"description": "Comma-separated LinkedIn numeric company IDs. IDs are opaque and must be resolved from a company's public page or a prior search response, not guessed.",
			},
			"posted_within": map[string]any{
				"type":        "string",
				"description": "Only jobs posted within this window.",
				"enum":        []any{"Past day", "Past week", "Past month"},
			},
			"start": map[string]any{
				"type":        "integer",
				"description": "Zero-based result offset; default 0. The endpoint always returns exactly 10 cards per call regardless of this value, so paging through results must increment start by exactly 10 each call (0, 10, 20, ...) to avoid gaps. Do not mimic a real browser's 25-per-step scroll traffic, which skips 10 of every 25 positions this endpoint can return.",
				"minimum":     float64(0),
			},
		},
		"additionalProperties": false,
	}
	assert.Equal(t, want, schema)

	callRes, err := clientSession.CallTool(t.Context(), &mcp.CallToolParams{
		Name: "linkedin_search_jobs",
		Arguments: map[string]any{
			"keyword":        "software engineer",
			"location":       "Taiwan",
			"workplace_type": "Remote",
			"job_type":       "Full-time",
			"posted_within":  "Past week",
			"start":          0,
		},
	})
	require.NoError(t, err)
	require.False(t, callRes.IsError)

	data, err := json.Marshal(callRes.StructuredContent)
	require.NoError(t, err)
	var got linkedinSearchOutput
	require.NoError(t, json.Unmarshal(data, &got))

	want2 := &linkedinSearchOutput{
		Data: []linkedinJobSummary{
			{ID: "4422697744", Title: "Software Engineer", Company: "BoostDraft", CompanyURL: "https://www.linkedin.com/company/boostdraft", Location: "Taiwan", PostedDate: "2026-06-03", URL: "https://www.linkedin.com/jobs/view/4422697744"},
			{ID: "4430577683", Title: "Software Engineer, Apps, Pixel", Company: "Google", CompanyURL: "https://www.linkedin.com/company/google", Location: "Banqiao District, New Taipei City, Taiwan", PostedDate: "2026-06-22", URL: "https://www.linkedin.com/jobs/view/4430577683"},
			{ID: "4435540496", Title: "Software Engineer", Company: "Mphasis", CompanyURL: "https://in.linkedin.com/company/mphasis", Location: "Taipei, Taipei City, Taiwan", PostedDate: "2026-07-01", URL: "https://www.linkedin.com/jobs/view/4435540496"},
			{ID: "4409906484", Title: "Software Engineer (Taipei)", Company: "Nitra", CompanyURL: "https://www.linkedin.com/company/nitrahq", Location: "Taipei, Taipei City, Taiwan", PostedDate: "2026-05-07", URL: "https://www.linkedin.com/jobs/view/4409906484"},
			{ID: "4430941394", Title: "(f2pool) Software Engineer - Back-end / Full-stack", Company: "stakefish", CompanyURL: "https://vg.linkedin.com/company/stakefish", Location: "Taiwan", PostedDate: "2026-05-25", URL: "https://www.linkedin.com/jobs/view/4430941394"},
			{ID: "4435420998", Title: "Full Stack Software Engineer", Company: "MediaTek", CompanyURL: "https://tw.linkedin.com/company/mediatek", Location: "Hsinchu, Taiwan, Taiwan", PostedDate: "2026-07-03", URL: "https://www.linkedin.com/jobs/view/4435420998"},
			{ID: "4425114186", Title: "Software Engineer - Dajia, Taichung City, Taiwan", Company: "Winbro", CompanyURL: "https://www.linkedin.com/company/winbro", Location: "Dajia District, Taichung City, Taiwan", PostedDate: "2026-05-12", URL: "https://www.linkedin.com/jobs/view/4425114186"},
			{ID: "4435546265", Title: "Software Engineer", Company: "Mphasis", CompanyURL: "https://in.linkedin.com/company/mphasis", Location: "Taipei, Taipei City, Taiwan", PostedDate: "2026-07-01", URL: "https://www.linkedin.com/jobs/view/4435546265"},
			{ID: "4435541354", Title: "Software Engineer", Company: "Mphasis", CompanyURL: "https://in.linkedin.com/company/mphasis", Location: "Taipei, Taipei City, Taiwan", PostedDate: "2026-07-01", URL: "https://www.linkedin.com/jobs/view/4435541354"},
			{ID: "4401701902", Title: "Senior Software Engineer / Lead Software Engineer", Company: "BoostDraft", CompanyURL: "https://www.linkedin.com/company/boostdraft", Location: "Taiwan", PostedDate: "2026-04-14", URL: "https://www.linkedin.com/jobs/view/4401701902"},
		},
	}
	assert.Equal(t, want2, &got)
}

func TestLinkedinSearchJobsInvalidEnumE2E(t *testing.T) {
	clientSession, _ := testLinkedinMCPClientServer(t)

	// A value outside a property's enum is rejected by the SDK's
	// input-schema validation before the handler runs, as an IsError tool
	// result.
	callRes, err := clientSession.CallTool(t.Context(), &mcp.CallToolParams{
		Name:      "linkedin_search_jobs",
		Arguments: map[string]any{"workplace_type": "valueNotInEnum"},
	})
	require.NoError(t, err)
	require.True(t, callRes.IsError)
	text, ok := callRes.Content[0].(*mcp.TextContent)
	require.True(t, ok)
	assert.Contains(t, text.Text, `validating /properties/workplace_type: enum`)
}

func TestLinkedinGetJobDetailE2E(t *testing.T) {
	clientSession, _ := testLinkedinMCPClientServer(t)

	callRes, err := clientSession.CallTool(t.Context(), &mcp.CallToolParams{
		Name:      "linkedin_get_job_detail",
		Arguments: map[string]any{"job_id": "4422697744"},
	})
	require.NoError(t, err)
	require.False(t, callRes.IsError)

	data, err := json.Marshal(callRes.StructuredContent)
	require.NoError(t, err)
	var got linkedinDetailOutput
	require.NoError(t, json.Unmarshal(data, &got))

	// The fixture's description is a very large bilingual blob; asserted via
	// assert.Contains below rather than pinned verbatim, same as
	// internal/provider/linkedin/parse_test.go's TestParseDetailHTML.
	description := got.Description
	got.Description = ""

	want := linkedinDetailOutput{
		ID:             "4422697744",
		Title:          "Software Engineer",
		Company:        "BoostDraft",
		Location:       "Taiwan",
		Posted:         "1 month ago",
		SeniorityLevel: "Entry level",
		EmploymentType: "Full-time",
		JobFunction:    "Other",
		Industries:     "IT Services and IT Consulting",
	}
	assert.Equal(t, want, got)
	assert.Contains(t, description, "BoostDraft is a software engineering company")
	assert.Contains(t, description, "Fluent in coding with C#")
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/openingsmcp/... -run 'TestRegisterLinkedin|TestLinkedinSearchJobsE2E|TestLinkedinSearchJobsInvalidEnumE2E|TestLinkedinGetJobDetailE2E' -v`
Expected: FAIL — `RegisterLinkedin` undefined.

- [ ] **Step 3: Write the implementation**

Append to `internal/openingsmcp/linkedin.go` (add `"context"` and `"github.com/modelcontextprotocol/go-sdk/mcp"` to the import block):

```go
// RegisterLinkedin registers the LinkedIn search and job-detail tools.
func RegisterLinkedin(s *mcp.Server, c *linkedin.Client) {
	mcp.AddTool(s, &mcp.Tool{
		Name:        "linkedin_search_jobs",
		Description: "Search jobs on LinkedIn's public guest job-search surface by keyword/location, with optional workplace-type/job-type/easy-apply/company/posted-within filters. Caution: LinkedIn rate-limits aggressively -- a single session is typically cut off around the 10th consecutive search request with a plain HTTP 429 carrying no Retry-After hint. Page conservatively (start in steps of 10) and back off on your own schedule rather than retrying immediately after a 429.",
		Annotations: &mcp.ToolAnnotations{Title: "Search LinkedIn jobs", ReadOnlyHint: true},
		InputSchema: linkedinSearchInputSchema,
	}, func(ctx context.Context, _ *mcp.CallToolRequest, in *linkedinSearchInput) (*mcp.CallToolResult, *linkedinSearchOutput, error) {
		req, err := linkedinMCPToHTTPRequest(in)
		if err != nil {
			return errorResult(err), nil, nil
		}
		res, err := c.Jobs(ctx, req)
		if err != nil {
			return errorResult(err), nil, nil
		}
		return nil, linkedinHTTPToMCPResponse(res), nil
	})

	mcp.AddTool(s, &mcp.Tool{
		Name:        "linkedin_get_job_detail",
		Description: "Get the full job description and criteria for a LinkedIn job by numeric ID (id from linkedin_search_jobs results). Caution: this is the most block-prone endpoint -- a cold request can return HTTP 999 (bot-suspected authwall) -- and it shares the same session-wide rate-limit budget as search, so avoid fetching details for many jobs in one session.",
		Annotations: &mcp.ToolAnnotations{Title: "Get LinkedIn job details", ReadOnlyHint: true},
	}, func(ctx context.Context, _ *mcp.CallToolRequest, in *linkedinDetailInput) (*mcp.CallToolResult, *linkedinDetailOutput, error) {
		res, err := c.JobDetail(ctx, in.JobID)
		if err != nil {
			return errorResult(err), nil, nil
		}
		return nil, linkedinHTTPToMCPDetail(res), nil
	})
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/openingsmcp/... -v`
Expected: PASS (every test in the package, including Tasks 1–2's tests).

- [ ] **Step 5: Commit**

```bash
git add internal/openingsmcp/linkedin.go internal/openingsmcp/linkedin_test.go
git commit -m "feat(openingsmcp): register linkedin_search_jobs and linkedin_get_job_detail tools"
```

---

### Task 4: Wire the LinkedIn client into the MCP server binary

**Files:**
- Modify: `cmd/openings-mcp/main.go`
- Modify: `cmd/openings-mcp/main_test.go`

**Interfaces:**
- Consumes: `linkedin.NewClient(baseURL string, httpClient *http.Client) *linkedin.Client`, `openingsmcp.RegisterLinkedin(s *mcp.Server, c *linkedin.Client)` (from Task 3).
- Produces: `newServer` gains a `cLinkedin *linkedin.Client` parameter (6th positional, before `logger`) — this is the last task, nothing downstream depends on it.

- [ ] **Step 1: Update `cmd/openings-mcp/main.go` imports and `serverInstructions`**

In `cmd/openings-mcp/main.go`, update the import block (add `"net/http/cookiejar"` and the linkedin provider):

```go
import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"log/slog"
	"net/http"
	"net/http/cookiejar"
	"os"
	"time"

	"github.com/peterbourgon/ff/v4"
	"github.com/peterbourgon/ff/v4/ffhelp"

	"github.com/amikai/openings-mcp/internal/openingsmcp"
	"github.com/amikai/openings-mcp/internal/logging"
	"github.com/amikai/openings-mcp/internal/provider/cake"
	"github.com/amikai/openings-mcp/internal/provider/google"
	"github.com/amikai/openings-mcp/internal/provider/job104"
	"github.com/amikai/openings-mcp/internal/provider/linkedin"
	"github.com/amikai/openings-mcp/internal/provider/nvidia"
	"github.com/amikai/openings-mcp/internal/provider/tsmc"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)
```

Replace the `serverInstructions` constant with:

```go
const serverInstructions = `openings-mcp exposes job-search tools for six job boards: 104 and Cake.me (both Taiwan-centric) and LinkedIn (global), plus the official careers sites of Google, NVIDIA, and TSMC.

Tool selection:
- When the user names a site or company, use that provider's tools.
- When the user has no target in mind, offer them the provider choices; if they don't pick one, start with the job boards (104, Cake.me, and LinkedIn) rather than a single company's careers site.

Query construction:
- Listen carefully to the user's stated criteria and map each one onto a search parameter when a matching parameter exists; enforce criteria the parameters cannot express by filtering the results yourself.
- Keep the keyword parameter to role titles, skills, or technologies. Location, job type, seniority, and other constraints go in their dedicated parameters, never embedded in the keyword string.
- Every provider follows the same search-then-detail flow: <provider>_search_jobs returns summaries carrying an identifier (job code, ID, or path), and <provider>_get_job_detail exchanges that identifier for the full posting. Identifiers are provider-specific and not interchangeable. The detail step is conditional, not automatic: when a summary from the search step fails the user's criteria, drop it and never call get_job_detail for it.

Context management:
- Search results are paginated; fetch additional pages rather than broadening the query.
- After filtering, fetch details when both hold: the user's criteria include something summaries can't answer (tech stack, remote policy, overtime culture, education requirements written in the posting body, etc.), and the filtered set is small enough to fetch economically (roughly 5-10 postings). If either condition fails, present summaries and let the user decide whether to go deeper.`
```

- [ ] **Step 2: Construct the LinkedIn client in `runWithTransport` and pass it to `newServer`**

In `cmd/openings-mcp/main.go`, in `runWithTransport`, after the `cGoogle := google.NewClient(...)` line, add:

```go
	jarLinkedin, _ := cookiejar.New(nil)
	cLinkedin := linkedin.NewClient("https://www.linkedin.com", &http.Client{Timeout: 30 * time.Second, Jar: jarLinkedin})
```

Change the `newServer` call on the next line from:

```go
	server := newServer(c104, cCake, cNvidia, cTsmc, cGoogle, logger)
```

to:

```go
	server := newServer(c104, cCake, cNvidia, cTsmc, cGoogle, cLinkedin, logger)
```

- [ ] **Step 3: Update `newServer`'s signature and body**

Change:

```go
func newServer(c104 *job104.Client, cCake *cake.Client, cNvidia *nvidia.Client, cTsmc *tsmc.Client, cGoogle *google.Client, logger *slog.Logger) *mcp.Server {
	server := mcp.NewServer(&mcp.Implementation{Name: "openings-mcp", Version: version}, &mcp.ServerOptions{Instructions: serverInstructions, Logger: logger})
	server.AddReceivingMiddleware(logging.ErrorLoggingMiddleware(logger))
	openingsmcp.RegisterJob104(server, c104)
	openingsmcp.RegisterCake(server, cCake)
	openingsmcp.RegisterNvidia(server, cNvidia)
	openingsmcp.RegisterTsmc(server, cTsmc)
	openingsmcp.RegisterGoogle(server, cGoogle)
	return server
}
```

to:

```go
func newServer(c104 *job104.Client, cCake *cake.Client, cNvidia *nvidia.Client, cTsmc *tsmc.Client, cGoogle *google.Client, cLinkedin *linkedin.Client, logger *slog.Logger) *mcp.Server {
	server := mcp.NewServer(&mcp.Implementation{Name: "openings-mcp", Version: version}, &mcp.ServerOptions{Instructions: serverInstructions, Logger: logger})
	server.AddReceivingMiddleware(logging.ErrorLoggingMiddleware(logger))
	openingsmcp.RegisterJob104(server, c104)
	openingsmcp.RegisterCake(server, cCake)
	openingsmcp.RegisterNvidia(server, cNvidia)
	openingsmcp.RegisterTsmc(server, cTsmc)
	openingsmcp.RegisterGoogle(server, cGoogle)
	openingsmcp.RegisterLinkedin(server, cLinkedin)
	return server
}
```

- [ ] **Step 4: Update `cmd/openings-mcp/main_test.go`**

Add `"github.com/amikai/openings-mcp/internal/provider/linkedin"` to the import block.

In `TestServerListsJobTools`, after the `cGoogle := google.NewClient(...)` line, add:

```go
	cLinkedin := linkedin.NewClient("https://www.linkedin.com", http.DefaultClient)
```

Change:

```go
	server := newServer(c104, cCake, cNvidia, cTsmc, cGoogle, slog.New(slog.NewTextHandler(io.Discard, nil)))
```

to:

```go
	server := newServer(c104, cCake, cNvidia, cTsmc, cGoogle, cLinkedin, slog.New(slog.NewTextHandler(io.Discard, nil)))
```

Add `"linkedin_search_jobs"` and `"linkedin_get_job_detail"` to the expected tool-name list:

```go
	for _, name := range []string{
		"104_search_jobs",
		"104_get_job_detail",
		"cake_search_jobs",
		"cake_get_job_detail",
		"nvidia_search_jobs",
		"nvidia_get_job_detail",
		"tsmc_search_jobs",
		"tsmc_get_job_detail",
		"google_search_jobs",
		"google_get_job_detail",
		"linkedin_search_jobs",
		"linkedin_get_job_detail",
	} {
		assert.Contains(t, got, name)
	}
```

- [ ] **Step 5: Run the test to verify it passes**

Run: `go test ./cmd/openings-mcp/... -run TestServerListsJobTools -v`
Expected: PASS.

- [ ] **Step 6: Build and run the full test suite**

Run: `go build ./... && go test ./...`
Expected: build succeeds; all tests PASS (including the pre-existing `TestRunWithTransportTreatsStdinEOFAsCleanExit`, which is unaffected).

- [ ] **Step 7: Commit**

```bash
git add cmd/openings-mcp/main.go cmd/openings-mcp/main_test.go
git commit -m "feat(openings-mcp): wire LinkedIn provider into the MCP server"
```
