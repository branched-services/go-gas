// Package observability provides logging and metrics infrastructure.
// Logs are written to stdout as structured data (12-factor: treat logs as event streams).
package observability

import (
	"context"
	"log/slog"
	"os"
	"strings"
)

// LogContextKey is the type for context keys used in logging.
type LogContextKey string

const (
	// RequestIDKey is the context key for request IDs.
	RequestIDKey LogContextKey = "request_id"
)

// NewLogger creates a configured slog.Logger.
// Output is always stdout (12-factor compliant).
func NewLogger(level, format string) *slog.Logger {
	lvl := parseLevel(level)
	opts := &slog.HandlerOptions{
		Level:     lvl,
		AddSource: lvl == slog.LevelDebug,
	}

	var handler slog.Handler
	if strings.ToLower(format) == "json" {
		handler = slog.NewJSONHandler(os.Stdout, opts)
	} else {
		handler = slog.NewTextHandler(os.Stdout, opts)
	}

	return slog.New(handler)
}

func parseLevel(level string) slog.Level {
	switch strings.ToLower(level) {
	case "debug":
		return slog.LevelDebug
	case "warn", "warning":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

// WithContext returns a logger with fields extracted from context.
func WithContext(ctx context.Context, logger *slog.Logger) *slog.Logger {
	if reqID, ok := ctx.Value(RequestIDKey).(string); ok {
		logger = logger.With("request_id", reqID)
	}
	return logger
}

// Component returns a logger scoped to a specific component.
func Component(logger *slog.Logger, name string) *slog.Logger {
	return logger.With("component", name)
}
