// Package middleware provides HTTP transport middleware, rate limiting, caching,
// response shaping, and structured logging for the xurrent-mcp server.
package middleware

import (
	"context"
	"log/slog"
	"os"
	"strings"
)

// Logger returns the configured slog.Logger. Defaults to JSON to stderr at info level.
// Set LOG_LEVEL=debug, info, warn, error. Set LOG_FORMAT=text for human-readable output.
func Logger() *slog.Logger {
	level := slog.LevelInfo
	switch strings.ToLower(strings.TrimSpace(os.Getenv("LOG_LEVEL"))) {
	case "debug":
		level = slog.LevelDebug
	case "info":
		level = slog.LevelInfo
	case "warn", "warning":
		level = slog.LevelWarn
	case "error":
		level = slog.LevelError
	}
	opts := &slog.HandlerOptions{
		Level: level,
		// Redact Authorization headers and token-like values.
		ReplaceAttr: func(_ []string, a slog.Attr) slog.Attr {
			if a.Key == "authorization" || a.Key == "token" || strings.Contains(strings.ToLower(a.Key), "token") {
				return slog.String(a.Key, "[REDACTED]")
			}
			return a
		},
	}
	var h slog.Handler
	if strings.ToLower(strings.TrimSpace(os.Getenv("LOG_FORMAT"))) == "text" {
		h = slog.NewTextHandler(os.Stderr, opts)
	} else {
		h = slog.NewJSONHandler(os.Stderr, opts)
	}
	return slog.New(h)
}

// Logf is a helper for test-friendly logging without a full logger.
func Logf(ctx context.Context, format string, args ...interface{}) {
	slog.DebugContext(ctx, "", "msg", format, "args", args)
}
