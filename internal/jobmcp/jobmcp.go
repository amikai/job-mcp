// Package jobmcp adapts the internal job-board clients into MCP tools.
package jobmcp

import (
	"context"

	"github.com/amikai/job-mcp/internal/logging"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// errorResult reports a failure to the model without aborting the tool call,
// and records the error in the context for log auditing.
func errorResult(ctx context.Context, err error) *mcp.CallToolResult {
	logging.AddErrorToContext(ctx, err)
	return &mcp.CallToolResult{
		IsError: true,
		Content: []mcp.Content{&mcp.TextContent{Text: err.Error()}},
	}
}
