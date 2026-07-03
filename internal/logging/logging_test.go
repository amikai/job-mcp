package logging

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"strings"
	"sync"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/assert"
)

func TestIOLogger_Read(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug}))

	input := "hello json-rpc"
	reader := strings.NewReader(input)
	ioLogger := NewIOLogger(reader, nil, logger)

	p := make([]byte, 50)
	n, err := ioLogger.Read(p)
	assert.NoError(t, err)
	assert.Equal(t, len(input), n)
	assert.Equal(t, input, string(p[:n]))

	logOutput := buf.String()
	assert.Contains(t, logOutput, "[stdin]: received bytes")
	assert.Contains(t, logOutput, "count=14")
	assert.Contains(t, logOutput, "data=\"hello json-rpc\"")
}

func TestIOLogger_Write(t *testing.T) {
	var logBuf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&logBuf, &slog.HandlerOptions{Level: slog.LevelDebug}))

	var writeBuf bytes.Buffer
	ioLogger := NewIOLogger(nil, &writeBuf, logger)

	data := []byte("response payload")
	n, err := ioLogger.Write(data)
	assert.NoError(t, err)
	assert.Equal(t, len(data), n)
	assert.Equal(t, "response payload", writeBuf.String())

	logOutput := logBuf.String()
	assert.Contains(t, logOutput, "[stdout]: sending bytes")
	assert.Contains(t, logOutput, "count=16")
	assert.Contains(t, logOutput, "data=\"response payload\"")
}

func TestContextErrors(t *testing.T) {
	ctx := ContextWithErrors(context.Background())
	assert.Nil(t, GetErrors(ctx))

	err1 := errors.New("first error")
	err2 := errors.New("second error")

	AddErrorToContext(ctx, err1)
	AddErrorToContext(ctx, err2)

	errs := GetErrors(ctx)
	assert.Len(t, errs, 2)
	assert.Equal(t, err1, errs[0])
	assert.Equal(t, err2, errs[1])
}

func TestContextErrorsConcurrency(t *testing.T) {
	ctx := ContextWithErrors(context.Background())
	var wg sync.WaitGroup

	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func(val int) {
			defer wg.Done()
			AddErrorToContext(ctx, errors.New("concurrent error"))
		}(i)
	}

	wg.Wait()
	errs := GetErrors(ctx)
	assert.Len(t, errs, 50)
}

func TestErrorLoggingMiddleware(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug}))

	middleware := ErrorLoggingMiddleware(logger)
	dummyHandler := func(ctx context.Context, method string, req mcp.Request) (mcp.Result, error) {
		AddErrorToContext(ctx, errors.New("api lookup failed"))
		return nil, errors.New("handler level failure")
	}

	wrapped := middleware(dummyHandler)
	_, err := wrapped(context.Background(), "test_method", nil)
	assert.Error(t, err)

	logOutput := buf.String()
	assert.Contains(t, logOutput, "upstream client error")
	assert.Contains(t, logOutput, "error=\"api lookup failed\"")
	assert.Contains(t, logOutput, "MCP protocol handler error")
	assert.Contains(t, logOutput, "error=\"handler level failure\"")
}
