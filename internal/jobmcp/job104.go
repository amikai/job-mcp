package jobmcp

import (
	"context"
	"fmt"

	"github.com/amikai/job-mcp/internal/provider/job104"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type job104SearchInput struct {
	Keyword string `json:"keyword" jsonschema:"search keyword, required"`
	Area    string `json:"area,omitempty" jsonschema:"city filter; one of: taipei, new_taipei, taoyuan, taichung, tainan, kaohsiung"`
	JobType string `json:"job_type,omitempty" jsonschema:"employment basis; one of: full, part"`
	Sort    string `json:"sort,omitempty" jsonschema:"result order; one of: newest, relevance"`
	Remote  string `json:"remote,omitempty" jsonschema:"remote work; one of: partial, full (there is no explicit 'none' value — omit this field for that)"`
	Page    int    `json:"page,omitempty" jsonschema:"1-based page number"`
}

type job104DetailInput struct {
	JobCode string `json:"job_code" jsonschema:"104 job code (jobNo), required"`
}

// Areas keyed by the MCP tool's public "taipei"/"new_taipei"/... vocabulary,
// pointing at job104.AreaIDs (which uses "Taipei"/"NewTaipei"/... labels).
var job104AreaAliases = map[string]string{
	"taipei":     "Taipei",
	"new_taipei": "NewTaipei",
	"taoyuan":    "Taoyuan",
	"taichung":   "Taichung",
	"tainan":     "Tainan",
	"kaohsiung":  "Kaohsiung",
}

// JobType/Sort/Remote keyed by the MCP tool's public "full"/"part"/...
// vocabulary, pointing at job104.RoIDs/OrderIDs/RemoteWorkIDs (which use
// "Full-time"/"Newest"/... labels) — the canonical id values live only
// there, not duplicated here (that duplication is exactly how RO/RemoteWork
// ended up wrong in an earlier version of this file).
var (
	job104JobTypeAliases = map[string]string{"full": "Full-time", "part": "Part-time"}
	job104SortAliases    = map[string]string{"newest": "Newest", "relevance": "Relevance"}
	job104RemoteAliases  = map[string]string{"partial": "Partial", "full": "Full"}
)

func job104ToRequest(in job104SearchInput) (job104.SearchJobsParams, error) {
	var params job104.SearchJobsParams
	if in.Keyword != "" {
		params.Keyword = job104.NewOptString(in.Keyword)
	}
	if in.Area != "" {
		label, ok := job104AreaAliases[in.Area]
		if !ok {
			return params, fmt.Errorf("invalid area %q (want taipei|new_taipei|taoyuan|taichung|tainan|kaohsiung)", in.Area)
		}
		params.Area = job104.NewOptSearchJobsArea(job104.AreaIDs[label])
	}
	if in.JobType != "" {
		label, ok := job104JobTypeAliases[in.JobType]
		if !ok {
			return params, fmt.Errorf("invalid job_type %q (want full|part)", in.JobType)
		}
		params.Ro = job104.NewOptSearchJobsRo(job104.RoIDs[label])
	}
	if in.Sort != "" {
		label, ok := job104SortAliases[in.Sort]
		if !ok {
			return params, fmt.Errorf("invalid sort %q (want newest|relevance)", in.Sort)
		}
		params.Order = job104.NewOptSearchJobsOrder(job104.OrderIDs[label])
	}
	if in.Remote != "" {
		label, ok := job104RemoteAliases[in.Remote]
		if !ok {
			return params, fmt.Errorf("invalid remote %q (want partial|full)", in.Remote)
		}
		params.RemoteWork = job104.NewOptSearchJobsRemoteWork(job104.RemoteWorkIDs[label])
	}
	if in.Page > 0 {
		params.Page = job104.NewOptInt(in.Page)
	}
	return params, nil
}

// RegisterJob104 registers the 104 search and job-detail tools.
func RegisterJob104(s *mcp.Server, c *job104.Client) {
	mcp.AddTool(s, &mcp.Tool{
		Name:        "104_search_jobs",
		Description: "Search jobs on 104 (Taiwan's largest job board) by keyword, with optional city/job-type/remote/sort filters.",
	}, func(ctx context.Context, _ *mcp.CallToolRequest, in job104SearchInput) (*mcp.CallToolResult, any, error) {
		params, err := job104ToRequest(in)
		if err != nil {
			return errorResult(err), nil, nil
		}
		resp, err := c.SearchJobs(ctx, params)
		if err != nil {
			return errorResult(err), nil, nil
		}
		return nil, resp, nil
	})

	mcp.AddTool(s, &mcp.Tool{
		Name:        "104_get_job_detail",
		Description: "Get the full job description for a 104 job code (jobNo from search results).",
	}, func(ctx context.Context, _ *mcp.CallToolRequest, in job104DetailInput) (*mcp.CallToolResult, any, error) {
		resp, err := c.GetJobDetail(ctx, job104.GetJobDetailParams{JobCode: in.JobCode})
		if err != nil {
			return errorResult(err), nil, nil
		}
		detail, ok := resp.(*job104.JobDetailResponse)
		if !ok {
			return errorResult(fmt.Errorf("job detail %s returned %T", in.JobCode, resp)), nil, nil
		}
		return nil, detail, nil
	})
}
