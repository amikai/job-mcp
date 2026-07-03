// Package jobmcp adapts the internal job-board clients into MCP tools.
package jobmcp

import (
	"github.com/amikai/job-mcp/internal/provider/cake"
	"github.com/amikai/job-mcp/internal/provider/google"
	"github.com/amikai/job-mcp/internal/provider/job104"
	"github.com/amikai/job-mcp/internal/provider/nvidia"
	"github.com/amikai/job-mcp/internal/provider/tsmc"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// serverInstructions carries the cross-tool guidance for host LLMs: provider
// routing and the shared search→detail flow. Per-tool behavior stays in each
// tool's description.
const serverInstructions = `job-mcp exposes job-search tools for five job boards: 104 and Cake.me (both Taiwan-centric), plus the official careers sites of Google, NVIDIA, and TSMC.

Tool selection:
- When the user names a site or company, use that provider's tools. Otherwise search 104 and Cake.me for jobs in Taiwan, and the company careers tools for roles at Google, NVIDIA, or TSMC.
- Every provider follows the same two-step flow: <provider>_search_jobs returns summaries carrying an identifier (job code, ID, or path), and <provider>_get_job_detail exchanges that identifier for the full posting. Identifiers are provider-specific and not interchangeable.

Context management:
- Search results are paginated; fetch additional pages rather than broadening the query.
- Fetch job details only for postings you intend to present.`

// NewServer builds the job-mcp server with every provider's tools registered.
func NewServer(c104 *job104.Client, cCake *cake.Client, cNvidia *nvidia.Client, cTsmc *tsmc.Client, cGoogle *google.Client) *mcp.Server {
	server := mcp.NewServer(&mcp.Implementation{Name: "job-mcp"}, &mcp.ServerOptions{Instructions: serverInstructions})
	RegisterJob104(server, c104)
	RegisterCake(server, cCake)
	RegisterNvidia(server, cNvidia)
	RegisterTsmc(server, cTsmc)
	RegisterGoogle(server, cGoogle)
	return server
}

// errorResult reports a failure to the model without aborting the tool call.
func errorResult(err error) *mcp.CallToolResult {
	return &mcp.CallToolResult{
		IsError: true,
		Content: []mcp.Content{&mcp.TextContent{Text: err.Error()}},
	}
}
