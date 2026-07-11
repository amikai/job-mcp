// Package logging provides MCP server middleware for request logging and
// panic recovery.
package logging

import (
	"context"
	"log/slog"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// LoggingMiddleware returns an MCP middleware that logs request handling at
// levels matching severity: each request's start and completion (with
// duration) at debug, tool results flagged as errors and protocol-level
// handler errors at error. The logger's level decides which entries appear.
func LoggingMiddleware(logger *slog.Logger) mcp.Middleware {
	return func(next mcp.MethodHandler) mcp.MethodHandler {
		return func(ctx context.Context, method string, request mcp.Request) (mcp.Result, error) {
			logger.Debug("request received", "method", method)
			start := time.Now()

			res, err := next(ctx, method, request)

			logger.Debug("request completed", "method", method, "duration", time.Since(start))

			if r, ok := res.(*mcp.CallToolResult); ok && r.IsError {
				for _, c := range r.Content {
					if tc, ok := c.(*mcp.TextContent); ok {
						logger.Error("tool call error", "method", method, "error", tc.Text)
					}
				}
			}

			if err != nil {
				logger.Error("MCP protocol handler error", "method", method, "error", err)
			}

			return res, err
		}
	}
}
