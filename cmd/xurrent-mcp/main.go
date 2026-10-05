// xurrent-mcp is an MCP (Model Context Protocol) server providing CRUD access
// to the Xurrent ITSM platform REST API.
//
// It follows a "layered + query" pattern: a small set of powerful tools rather than
// one-per-endpoint. All mutations go through a draft → confirm safety flow.
//
// Transport modes:
//
//	MCP_TRANSPORT=stdio (default) — for local Claude Desktop / Cursor
//	MCP_TRANSPORT=http   — streamable HTTP for Docker / remote deployments
//
// Environment:
//
//	Required: XURRENT_TOKEN, XURRENT_ACCOUNT
//	Optional: XURRENT_API_BASE, XURRENT_LANGUAGE, XURRENT_ALLOW_MUTATIONS,
//	         XURRENT_HTTP_MIN_INTERVAL, LOG_LEVEL, PORT
package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/xurrent/xurrent-mcp/internal/server"
)

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	srv, err := server.New(nil)
	if err != nil {
		slog.Error("failed to create server", "error", err)
		os.Exit(1)
	}
	defer srv.Shutdown()

	if err := srv.Run(ctx); err != nil {
		slog.Error("server stopped with error", "error", err)
		os.Exit(1)
	}
}
