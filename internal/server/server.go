package server

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	openapiclient "github.com/bab3l/go-xurrent"

	"github.com/xurrent/xurrent-mcp/internal/client"
	"github.com/xurrent/xurrent-mcp/internal/middleware"
	"github.com/xurrent/xurrent-mcp/internal/otel"
)

// Server wraps the MCP server with its dependencies.
type Server struct {
	Server     *mcp.Server
	APIClient  *openapiclient.APIClient // go-xurrent client; nil when credentials unavailable
	Config     *client.EnvConfig
	Shaper     *middleware.ResponseShaper
	Cache      *middleware.Cache
	Logger     *slog.Logger
	DraftStore *DraftStore

	tracingShutdown func()
	metricsShutdown func()
}

// New creates a new xurrent-mcp server. If cfg is nil, reads from env
// but does not require credentials — tools that need API access return
// helpful error messages when the client is nil.
func New(cfg *client.EnvConfig) (*Server, error) {
	if cfg == nil {
		cfg, _ = client.ConfigFromEnv()
		if cfg == nil {
			cfg = &client.EnvConfig{Account: "unconfigured"}
		}
	}

	logger := middleware.Logger()

	// Optional OTEL.
	tracingShutdown, _ := otel.InitTracing()
	metricsShutdown, _ := otel.InitMetrics()

	// Try to create an API client; tolerate missing credentials.
	apiClient, apiErr := client.NewAPIClient(cfg)
	if apiErr != nil {
		logger.Warn("API client unavailable — tools that query Xurrent will return guidance, not data",
			"error", apiErr)
	}

	opts := &mcp.ServerOptions{
		Logger: logger,
		Instructions: `Xurrent ITSM MCP Server — use xurrent_discover to see available entities,
xurrent_query to fetch real records, and xurrent_create/update/delete
(with xurrent_draft for safety) to modify.

Authentication: set XURRENT_TOKEN and XURRENT_ACCOUNT in the environment.
Without credentials, discovery/validation/collision tools still work.`,
	}

	srv := &Server{
		Server:          mcp.NewServer(&mcp.Implementation{Name: "xurrent-mcp", Version: "0.3.0"}, opts),
		APIClient:       apiClient,
		Config:          cfg,
		Shaper:          middleware.NewResponseShaper(),
		Cache:           middleware.NewCache(5 * time.Minute),
		Logger:          logger,
		DraftStore:      NewDraftStore(5 * time.Minute),
		tracingShutdown: tracingShutdown,
		metricsShutdown: metricsShutdown,
	}

	srv.registerTools()
	srv.registerResources()
	srv.registerPrompts()

	logger.Info("xurrent-mcp server created",
		"account", cfg.Account,
		"mutations", cfg.AllowMutations,
		"api_client", apiClient != nil,
	)

	return srv, nil
}

// Shutdown cleans up resources.
func (s *Server) Shutdown() {
	if s.Cache != nil {
		s.Cache.Stop()
	}
	if s.tracingShutdown != nil {
		s.tracingShutdown()
	}
	if s.metricsShutdown != nil {
		s.metricsShutdown()
	}
	s.Logger.Info("xurrent-mcp shutdown")
}

// TransportMode returns the transport mode from environment.
func TransportMode() string {
	mode := strings.ToLower(strings.TrimSpace(os.Getenv("MCP_TRANSPORT")))
	if mode == "" {
		mode = "stdio"
	}
	return mode
}

// HTTPAddr returns the listen address.
func HTTPAddr() string {
	port := strings.TrimSpace(os.Getenv("PORT"))
	if port == "" {
		port = "8080"
	}
	return ":" + port
}

// Run starts the MCP server on the configured transport.
func (s *Server) Run(ctx context.Context) error {
	mode := TransportMode()
	s.Logger.Info("starting xurrent-mcp", "transport", mode)

	switch mode {
	case "http":
		return s.runHTTP(ctx)
	default:
		return s.runStdio(ctx)
	}
}

func (s *Server) runStdio(ctx context.Context) error {
	return s.Server.Run(ctx, &mcp.StdioTransport{})
}

func (s *Server) runHTTP(ctx context.Context) error {
	handler := mcp.NewStreamableHTTPHandler(func(_ *http.Request) *mcp.Server {
		return s.Server
	}, nil)

	mux := http.NewServeMux()
	mux.Handle("/mcp", handler)

	// /health — lightweight liveness check (no API call).
	mux.HandleFunc("/health", s.handleHealthLiveness)
	// /healthz — full readiness check (verifies token viability against Xurrent API).
	mux.HandleFunc("/healthz", s.handleHealthReadiness)

	addr := HTTPAddr()
	s.Logger.Info("HTTP server listening", "addr", addr)
	httpSrv := &http.Server{Addr: addr, Handler: mux}

	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = httpSrv.Shutdown(shutdownCtx)
	}()

	return httpSrv.ListenAndServe()
}

// requireClient returns the API client or an error explaining how to configure it.
func (s *Server) requireClient() (*openapiclient.APIClient, error) {
	if s.APIClient == nil {
		return nil, fmt.Errorf("Xurrent API client not configured — set XURRENT_TOKEN and XURRENT_ACCOUNT environment variables. " +
			"For Docker: docker run -e XURRENT_TOKEN=... -e XURRENT_ACCOUNT=... " +
			"For Claude Desktop/Cursor: set these in your shell or launcher config. " +
			"For HTTP clients: include Authorization: Bearer <token> and X-Xurrent-Account: <account> headers.")
	}
	return s.APIClient, nil
}

// handleHealthLiveness returns 200 as long as the process is alive.
func (s *Server) handleHealthLiveness(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(`{"status":"ok","uptime":"live"}`))
}

// handleHealthReadiness verifies the Xurrent API is reachable and the token is viable.
// It calls GET /v1/me — the cheapest authenticated endpoint. Returns 200 on success,
// 503 when the API call fails (token invalid, network down, rate limited), with
// diagnostic detail in the response body.
func (s *Server) handleHealthReadiness(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	// Check 1: Is the API client configured?
	hasClient := s.APIClient != nil
	hasToken := strings.TrimSpace(os.Getenv("XURRENT_TOKEN")) != ""
	hasAccount := strings.TrimSpace(os.Getenv("XURRENT_ACCOUNT")) != ""

	// Check 2: If configured, verify token viability via /v1/me.
	if hasClient {
		ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
		defer cancel()

		body, resp, err := s.APIClient.GetCollectionJSON(ctx, "/v1/me", url.Values{})

		if err != nil {
			s.Logger.Warn("health check: API unreachable", "error", err)
			w.WriteHeader(http.StatusServiceUnavailable)
			_, _ = fmt.Fprintf(w, `{"status":"unhealthy","reason":"api_unreachable","error":%q}`+"\n", err.Error())
			return
		}
		defer resp.Body.Close()

		if resp.StatusCode >= 400 {
			statusStr := "token_invalid"
			if resp.StatusCode == 429 {
				statusStr = "rate_limited"
			}
			errBody := string(body)
			if len(errBody) > 200 {
				errBody = errBody[:200]
			}
			s.Logger.Warn("health check: token rejected", "status", resp.StatusCode)
			w.WriteHeader(http.StatusServiceUnavailable)
			_, _ = fmt.Fprintf(w, `{"status":"unhealthy","reason":%q,"http_status":%d,"detail":%q}`+"\n",
				statusStr, resp.StatusCode, errBody)
			return
		}

		// Parse the response to extract identity.
		var me map[string]any
		personName := "unknown"
		personEmail := "unknown"
		if json.Unmarshal(body, &me) == nil {
			if n, ok := me["name"].(string); ok {
				personName = n
			}
			if e, ok := me["primary_email"].(string); ok {
				personEmail = e
			}
		}

		// Check 3: Rate limit headers (informational).
		limit := resp.Header.Get("X-Ratelimit-Limit")
		remaining := resp.Header.Get("X-Ratelimit-Remaining")

		w.WriteHeader(http.StatusOK)
		_, _ = fmt.Fprintf(w, `{"status":"ok","authenticated":true,"account":%q,"person":%q,"email":%q,"rate_limit":{"limit":%q,"remaining":%q}}`+"\n",
			s.Config.Account, personName, personEmail, limit, remaining)
		return
	}

	// No client configured — report what's missing.
	reasons := []string{}
	if !hasToken {
		reasons = append(reasons, "missing XURRENT_TOKEN")
	}
	if !hasAccount {
		reasons = append(reasons, "missing XURRENT_ACCOUNT")
	}
	s.Logger.Warn("health check: no API client configured", "reasons", reasons)
	w.WriteHeader(http.StatusServiceUnavailable)
	_, _ = fmt.Fprintf(w, `{"status":"unhealthy","reason":"not_configured","missing":%q}`+"\n", reasons)
}
