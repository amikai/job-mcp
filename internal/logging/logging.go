package logging

import (
	"context"
	"io"
	"log/slog"
	"sync"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// IOLogger wraps an io.Reader and io.Writer to log raw data being read and written.
type IOLogger struct {
	reader io.Reader
	writer io.Writer
	logger *slog.Logger
}

// NewIOLogger creates a new IOLogger instance.
func NewIOLogger(r io.Reader, w io.Writer, logger *slog.Logger) *IOLogger {
	return &IOLogger{
		reader: r,
		writer: w,
		logger: logger,
	}
}

// Read reads data from the underlying io.Reader and logs it.
func (l *IOLogger) Read(p []byte) (n int, err error) {
	if l.reader == nil {
		return 0, io.EOF
	}
	n, err = l.reader.Read(p)
	if n > 0 {
		l.logger.Info("[stdin]: received bytes", "count", n, "data", string(p[:n]))
	}
	return n, err
}

// Write writes data to the underlying io.Writer and logs it.
func (l *IOLogger) Write(p []byte) (n int, err error) {
	if l.writer == nil {
		return 0, io.ErrClosedPipe
	}
	l.logger.Info("[stdout]: sending bytes", "count", len(p), "data", string(p))
	return l.writer.Write(p)
}

// Close closes the underlying reader and writer if they support it.
func (l *IOLogger) Close() error {
	var errReader, errWriter error
	if closer, ok := l.reader.(io.Closer); ok {
		errReader = closer.Close()
	}
	if closer, ok := l.writer.(io.Closer); ok {
		errWriter = closer.Close()
	}
	if errReader != nil {
		return errReader
	}
	return errWriter
}

type errorKey struct{}

// CtxErrors stores a thread-safe list of errors in the request context.
type CtxErrors struct {
	mu   sync.Mutex
	errs []error
}

// ContextWithErrors initializes the context with an error tracking structure.
func ContextWithErrors(ctx context.Context) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	return context.WithValue(ctx, errorKey{}, &CtxErrors{})
}

// AddErrorToContext appends a detailed error to the tracking structure in the context.
func AddErrorToContext(ctx context.Context, err error) {
	if ctx == nil {
		return
	}
	if val, ok := ctx.Value(errorKey{}).(*CtxErrors); ok {
		val.mu.Lock()
		val.errs = append(val.errs, err)
		val.mu.Unlock()
	}
}

// GetErrors retrieves the accumulated errors from the context.
func GetErrors(ctx context.Context) []error {
	if ctx == nil {
		return nil
	}
	if val, ok := ctx.Value(errorKey{}).(*CtxErrors); ok {
		val.mu.Lock()
		defer val.mu.Unlock()
		if len(val.errs) == 0 {
			return nil
		}
		errsCopy := make([]error, len(val.errs))
		copy(errsCopy, val.errs)
		return errsCopy
	}
	return nil
}

// ErrorLoggingMiddleware returns an MCP middleware that configures the context
// to capture upstream API errors, logs them upon request completion, and logs any protocol errors.
func ErrorLoggingMiddleware(logger *slog.Logger) mcp.Middleware {
	return func(next mcp.MethodHandler) mcp.MethodHandler {
		return func(ctx context.Context, method string, request mcp.Request) (mcp.Result, error) {
			ctx = ContextWithErrors(ctx)
			res, err := next(ctx, method, request)

			if errs := GetErrors(ctx); len(errs) > 0 {
				for _, e := range errs {
					logger.Error("upstream client error", "method", method, "error", e)
				}
			}

			if err != nil {
				logger.Error("MCP protocol handler error", "method", method, "error", err)
			}

			return res, err
		}
	}
}
