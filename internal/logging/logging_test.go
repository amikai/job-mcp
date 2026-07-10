package logging

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestErrorLoggingMiddleware(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&buf, nil))

	middleware := ErrorLoggingMiddleware(logger)
	dummyHandler := func(ctx context.Context, method string, req mcp.Request) (mcp.Result, error) {
		res := &mcp.CallToolResult{
			IsError: true,
			Content: []mcp.Content{&mcp.TextContent{Text: "api lookup failed"}},
		}
		return res, errors.New("handler level failure")
	}

	wrapped := middleware(dummyHandler)
	_, err := wrapped(t.Context(), "test_method", nil)
	require.Error(t, err)

	logOutput := buf.String()
	assert.Contains(t, logOutput, "tool call error")
	assert.Contains(t, logOutput, "error=\"api lookup failed\"")
	assert.Contains(t, logOutput, "MCP protocol handler error")
	assert.Contains(t, logOutput, "error=\"handler level failure\"")
}

func TestErrorLoggingMiddlewareSuccessLogsNothing(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&buf, nil))

	middleware := ErrorLoggingMiddleware(logger)
	dummyHandler := func(ctx context.Context, method string, req mcp.Request) (mcp.Result, error) {
		return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: "ok"}}}, nil
	}

	wrapped := middleware(dummyHandler)
	_, err := wrapped(t.Context(), "test_method", nil)
	require.NoError(t, err)
	assert.Empty(t, buf.String())
}
