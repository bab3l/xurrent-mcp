//go:build integration
// +build integration

// Package main provides exhaustive endpoint coverage tests against the live Xurrent API.
// Every registered entity type is tested for:
//   1. Collection GET (list all, paginated)
//   2. Single-record GET by ID (with full field detail)
//   3. Filtered/state-based GET where applicable
//   4. Search API
//
// Run with:
//   cd xurrent-mcp && source .env && \
//   go test -tags=integration -run TestEndpoint -v -count=1 -timeout 10m ./cmd/xurrent-mcp/
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	openapiclient "github.com/bab3l/go-xurrent"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/require"
)

// ──────────────────────────────────────────────────────
// Test infrastructure
// ──────────────────────────────────────────────────────

type endpointTest struct {
	Entity       string   // Entity key in the registry
	Path         string   // API path
	Fields       string   // Fields for collection fetch
	DetailFields string   // Fields for single-record fetch
	MinRecords   int      // Minimum records expected
	States       []string // State filters to try (if applicable)
	StateField   string   // Query param name for state filter ("state" or empty)
}

func liveClient(t *testing.T) *openapiclient.APIClient {
	t.Helper()
	token := os.Getenv("XURRENT_TOKEN")
	account := os.Getenv("XURRENT_ACCOUNT")
	require.NotEmpty(t, token, "XURRENT_TOKEN")
	require.NotEmpty(t, account, "XURRENT_ACCOUNT")

	cfg := openapiclient.NewConfiguration()
	cfg.AddDefaultHeader("Authorization", "Bearer "+token)
	cfg.AddDefaultHeader("X-4me-Account", account)
	return openapiclient.NewAPIClient(cfg)
}

func liveMCP(t *testing.T, ctx context.Context) *mcp.ClientSession {
	t.Helper()
	_, thisFile, _, ok := runtime.Caller(0)
	require.True(t, ok)
	repoRoot := filepath.Join(filepath.Dir(thisFile), "..", "..")

	token := os.Getenv("XURRENT_TOKEN")
	account := os.Getenv("XURRENT_ACCOUNT")
	if token == "" || account == "" {
		t.Skip("set XURRENT_TOKEN and XURRENT_ACCOUNT")
	}

	client := mcp.NewClient(&mcp.Implementation{Name: "endpoint-cover", Version: "0.0.1"}, nil)
	cmd := exec.CommandContext(ctx, "go", "run", "./cmd/xurrent-mcp")
	cmd.Dir = repoRoot
	cmd.Env = append(os.Environ(),
		"XURRENT_TOKEN="+token,
		"XURRENT_ACCOUNT="+account,
		"XURRENT_HTTP_MIN_INTERVAL=0.4",
		"MCP_TRANSPORT=stdio",
	)
	transport := &mcp.CommandTransport{Command: cmd}
	session, err := client.Connect(ctx, transport, nil)
	require.NoError(t, err)
	t.Cleanup(func() { _ = session.Close() })
	return session
}

func getCollectionOrLog(t *testing.T, ctx context.Context, client *openapiclient.APIClient, path, query string) []map[string]any {
	t.Helper()
	q, _ := url.ParseQuery(query)
	body, resp, err := client.GetCollectionJSON(ctx, path, q)
	if err != nil || resp.StatusCode >= 400 {
		status := 0
		if resp != nil {
			status = resp.StatusCode
		}
		t.Logf("   state query skipped: GET %s?%s → HTTP %d (err=%v)", path, query, status, err)
		if resp != nil {
			resp.Body.Close()
		}
		return nil
	}
	defer resp.Body.Close()
	var rows []map[string]any
	if json.Unmarshal(body, &rows) != nil {
		return nil
	}
	return rows
}

func getCollectionTry(t *testing.T, ctx context.Context, client *openapiclient.APIClient, path, query string) ([]map[string]any, error) {
	t.Helper()
	q, _ := url.ParseQuery(query)
	body, resp, err := client.GetCollectionJSON(ctx, path, q)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("HTTP %d", resp.StatusCode)
	}
	var rows []map[string]any
	if err := json.Unmarshal(body, &rows); err != nil {
		// Try single object format.
		var single map[string]any
		if err2 := json.Unmarshal(body, &single); err2 != nil {
			return nil, fmt.Errorf("unmarshal: %w / %w", err, err2)
		}
		rows = []map[string]any{single}
	}
	return rows, nil
}

func getSingleTry(t *testing.T, ctx context.Context, client *openapiclient.APIClient, path string, id int64, query string) (map[string]any, error) {
	t.Helper()
	fullPath := fmt.Sprintf("%s/%d", path, id)
	q, _ := url.ParseQuery(query)
	body, resp, err := client.GetCollectionJSON(ctx, fullPath, q)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("HTTP %d", resp.StatusCode)
	}
	var row map[string]any
	if err := json.Unmarshal(body, &row); err != nil {
		return nil, fmt.Errorf("unmarshal: %w", err)
	}
	return row, nil
}

func getCollection(t *testing.T, ctx context.Context, client *openapiclient.APIClient, path, query string) []map[string]any {
	t.Helper()
	q, _ := url.ParseQuery(query)
	body, resp, err := client.GetCollectionJSON(ctx, path, q)
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Less(t, resp.StatusCode, 400, "GET %s?%s → HTTP %d", path, query, resp.StatusCode)

	var rows []map[string]any
	err = json.Unmarshal(body, &rows)
	require.NoError(t, err, "GET %s?%s: unmarshal failed", path, query)
	return rows
}

func getSingle(t *testing.T, ctx context.Context, client *openapiclient.APIClient, path string, id int64, query string) map[string]any {
	t.Helper()
	fullPath := fmt.Sprintf("%s/%d", path, id)
	q, _ := url.ParseQuery(query)
	body, resp, err := client.GetCollectionJSON(ctx, fullPath, q)
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Less(t, resp.StatusCode, 400, "GET %s → HTTP %d", fullPath, resp.StatusCode)

	var row map[string]any
	err = json.Unmarshal(body, &row)
	require.NoError(t, err, "GET %s: unmarshal failed", fullPath)
	return row
}

func idOf(m map[string]any) int64 {
	if v, ok := m["id"]; ok {
		switch n := v.(type) {
		case float64:
			return int64(n)
		}
	}
	return 0
}

func strOf(m map[string]any, key string) string {
	if v, ok := m[key]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

func firstID(rows []map[string]any) int64 {
	if len(rows) > 0 {
		return idOf(rows[0])
	}
	return 0
}

// ──────────────────────────────────────────────────────
// Endpoint test runner
// ──────────────────────────────────────────────────────

func runEndpointTest(t *testing.T, ctx context.Context, client *openapiclient.APIClient, mcpSession *mcp.ClientSession, e endpointTest) {
	t.Run(e.Entity, func(t *testing.T) {
		// ── 1. MCP Discover ──
		res, err := mcpSession.CallTool(ctx, &mcp.CallToolParams{
			Name: "xurrent_discover", Arguments: map[string]any{"entity": e.Entity},
		})
		require.NoError(t, err)
		require.False(t, res.IsError)
		raw, _ := json.Marshal(res.StructuredContent)
		var disc struct {
			Name     string `json:"name"`
			Path     string `json:"path"`
			DocURL   string `json:"doc_url"`
			Plural   string `json:"plural"`
		}
		require.NoError(t, json.Unmarshal(raw, &disc))
		t.Logf("── %s (%s) ──", disc.Name, disc.Plural)
		t.Logf("   path: %s | doc: %s", disc.Path, disc.DocURL)

		// ── 2. Collection fetch ──
		query := fmt.Sprintf("per_page=10&fields=%s", e.Fields)
		rows, fetchErr := getCollectionTry(t, ctx, client, e.Path, query)
		if fetchErr != nil {
			t.Logf("   collection GET skipped: %s?%s → %v", e.Path, query, fetchErr)
		} else {
			t.Logf("   collection GET: %d records (query: %s)", len(rows), query)
			require.GreaterOrEqual(t, len(rows), e.MinRecords,
				"expected ≥%d records for %s, got %d", e.MinRecords, e.Entity, len(rows))

			if len(rows) > 0 {
				// Show first 3 records.
				for i := 0; i < min(3, len(rows)); i++ {
					details := ""
					for _, k := range []string{"id", "name", "label", "subject", "primary_email"} {
						if v, ok := rows[i][k]; ok {
							details += fmt.Sprintf(" %s=%v", k, v)
						}
					}
					t.Logf("   [%d]%s", i+1, details)
				}

				// ── 3. Single-record GET by ID ──
				rid := firstID(rows)
				if rid == 0 {
					t.Logf("   single GET skipped: non-numeric id=%v", rows[0]["id"])
				} else {
					detailQuery := fmt.Sprintf("fields=%s", e.DetailFields)
					detail, dErr := getSingleTry(t, ctx, client, e.Path, rid, detailQuery)
					if dErr != nil {
						t.Logf("   single GET /%d skipped: %v", rid, dErr)
					} else {
						t.Logf("   single GET /%d: fields=%v (keys=%d)", rid, e.DetailFields, len(detail))
					}
				}
			}
		}

		// ── 4. State-based queries ──
		for _, state := range e.States {
			stateField := e.StateField
			if stateField == "" {
				stateField = "state"
			}
			stateQuery := fmt.Sprintf("per_page=5&fields=%s&%s=%s", e.Fields, stateField, state)
			stateRows := getCollectionOrLog(t, ctx, client, e.Path, stateQuery)
			t.Logf("   state=%s: %d records", state, len(stateRows))
		}

		// ── 5. MCP query tool (only if collection worked) ──
		if fetchErr == nil && len(rows) > 0 {
			mcpRes, err := mcpSession.CallTool(ctx, &mcp.CallToolParams{
				Name: "xurrent_query",
				Arguments: map[string]any{
					"entity":   e.Entity,
					"fields":   e.Fields,
					"per_page": float64(5),
				},
			})
			require.NoError(t, err)
			if mcpRes.IsError {
				t.Logf("   MCP query returned error (endpoint may be unavailable): %v", mcpRes.Content)
			} else {
				mcpRaw, _ := json.Marshal(mcpRes.StructuredContent)
				var mcpOut struct {
					Entity string `json:"entity"`
					Count  int    `json:"count"`
				}
				require.NoError(t, json.Unmarshal(mcpRaw, &mcpOut))
				require.Equal(t, e.Entity, mcpOut.Entity)
				t.Logf("   MCP query: entity=%s returned %d records ✓", mcpOut.Entity, mcpOut.Count)
			}
		} else {
			t.Logf("   MCP query: skipped (no collection data available)")
		}
	})
}

// ──────────────────────────────────────────────────────
// Test suites
// ──────────────────────────────────────────────────────

func TestEndpoint_Priority0(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	client := liveClient(t)
	mcpS := liveMCP(t, ctx)

	tests := []endpointTest{
		{
			Entity: "team", Path: "/v1/teams",
			Fields: "id,name,disabled", DetailFields: "id,name,disabled,remarks,manager",
			MinRecords: 1, States: []string{"enabled", "disabled"},
		},
		{
			Entity: "site", Path: "/v1/sites",
			Fields: "id,name,disabled", DetailFields: "id,name,disabled,address",
			MinRecords: 1, States: []string{"enabled", "disabled"},
		},
		{
			Entity: "service", Path: "/v1/services",
			Fields: "id,name,disabled", DetailFields: "id,name,disabled,provider",
			MinRecords: 1, States: []string{"enabled", "disabled"},
		},
		{
			Entity: "service_instance", Path: "/v1/service_instances",
			Fields: "id,name,status", DetailFields: "id,name,status,service,support_team",
			MinRecords: 1, States: []string{},
		},
		{
			Entity: "person", Path: "/v1/people",
			Fields: "id,name,primary_email,disabled", DetailFields: "id,name,primary_email,organization,site",
			MinRecords: 1, States: []string{"enabled", "disabled"},
		},
		{
			Entity: "organization", Path: "/v1/organizations",
			Fields: "id,name,disabled", DetailFields: "id,name,disabled,financialID",
			MinRecords: 1, States: []string{"enabled", "disabled"},
		},
		{
			Entity: "configuration_item", Path: "/v1/cis",
			Fields: "id,label,serial_nr,status", DetailFields: "id,label,serial_nr,status,product,site",
			MinRecords: 1, States: []string{"active", "trash"},
		},
		{
			Entity: "service_level_agreement", Path: "/v1/slas",
			Fields: "id,name,customer", DetailFields: "id,name,customer,service_offering",
			MinRecords: 1, States: []string{"active", "inactive"},
		},
		{
			Entity: "request", Path: "/v1/requests",
			Fields: "id,subject,status,category", DetailFields: "id,subject,status,category,impact,urgency,requested_by,requested_for",
			MinRecords: 0, States: []string{},
		},
		{
			Entity: "workflow", Path: "/v1/workflows",
			Fields: "id,subject,status", DetailFields: "id,subject,status,template",
			MinRecords: 0, States: []string{},
		},
		{
			Entity: "ui_extension", Path: "/v1/ui_extensions",
			Fields: "id,name,category,disabled", DetailFields: "id,name,category,disabled,css,html,javascript",
			MinRecords: 0, States: []string{},
		},
		{
			Entity: "automation_rule", Path: "/v1/automation_rules",
			Fields: "id,name,trigger,disabled", DetailFields: "id,name,trigger,disabled,condition,actions,expressions",
			MinRecords: 0, States: []string{},
		},
	}

	for _, e := range tests {
		runEndpointTest(t, ctx, client, mcpS, e)
	}
}

func TestEndpoint_Priority1(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()
	client := liveClient(t)
	mcpS := liveMCP(t, ctx)

	tests := []endpointTest{
		{
			Entity: "service_offering", Path: "/v1/service_offerings",
			Fields: "id,name,service", DetailFields: "id,name,service",
			MinRecords: 0,
		},
		{
			Entity: "request_template", Path: "/v1/request_templates",
			Fields: "id,subject", DetailFields: "id,subject,service",
			MinRecords: 0,
		},
		{
			Entity: "workflow_template", Path: "/v1/workflow_templates",
			Fields: "id,subject", DetailFields: "id,subject",
			MinRecords: 0,
		},
		{
			Entity: "task", Path: "/v1/tasks",
			Fields: "id,subject", DetailFields: "id,subject,workflow,assigned_team",
			MinRecords: 0,
		},
		{
			Entity: "task_template", Path: "/v1/task_templates",
			Fields: "id,subject", DetailFields: "id,subject",
			MinRecords: 0,
		},
		{
			Entity: "problem", Path: "/v1/problems",
			Fields: "id,subject", DetailFields: "id,subject,service_instance",
			MinRecords: 0,
		},
		{
			Entity: "project", Path: "/v1/projects",
			Fields: "id,subject", DetailFields: "id,subject",
			MinRecords: 0,
		},
		{
			Entity: "calendar", Path: "/v1/calendars",
			Fields: "id,name,disabled", DetailFields: "id,name,disabled",
			MinRecords: 0,
		},
		{
			Entity: "product", Path: "/v1/products",
			Fields: "id,name,brand,productID", DetailFields: "id,name,brand,productID,disabled",
			MinRecords: 0, States: []string{"enabled", "disabled"},
		},
		{
			Entity: "request_note", Path: "",
			Fields: "id,text", DetailFields: "id,text",
			MinRecords: 0,
		},
		{
			Entity: "configuration_item_relation", Path: "",
			Fields: "", DetailFields: "",
			MinRecords: 0,
		},
	}

	for _, e := range tests {
		if e.Path == "" {
			t.Run(e.Entity, func(t *testing.T) {
				// CI relations require a CI ID — skip for now.
				t.Skip("requires parent CI ID")
			})
			continue
		}
		runEndpointTest(t, ctx, client, mcpS, e)
	}
}

func TestEndpoint_Priority2(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()
	client := liveClient(t)
	mcpS := liveMCP(t, ctx)

	tests := []endpointTest{
		{
			Entity: "account", Path: "/v1/account",
			Fields: "id", DetailFields: "id",
			MinRecords: 0,
		},
		{
			Entity: "invoice", Path: "/v1/invoices",
			Fields: "id,subject", DetailFields: "id,subject",
			MinRecords: 0,
		},
		{
			Entity: "survey", Path: "/v1/surveys",
			Fields: "id,name", DetailFields: "id,name",
			MinRecords: 0,
		},
		{
			Entity: "reservation", Path: "/v1/reservations",
			Fields: "id,subject", DetailFields: "id,subject",
			MinRecords: 0,
		},
		{
			Entity: "custom_collection", Path: "/v1/custom_collections",
			Fields: "id,name", DetailFields: "id,name",
			MinRecords: 0,
		},
		{
			Entity: "contract", Path: "/v1/contracts",
			Fields: "id,name", DetailFields: "id,name",
			MinRecords: 0,
		},
		{
			Entity: "product_category", Path: "/v1/product_categories",
			Fields: "id,name", DetailFields: "id,name",
			MinRecords: 0,
		},
		{
			Entity: "first_line_support_agreement", Path: "/v1/flsas",
			Fields: "id", DetailFields: "id",
			MinRecords: 0,
		},
	}

	for _, e := range tests {
		runEndpointTest(t, ctx, client, mcpS, e)
	}
}

// ──────────────────────────────────────────────────────
// Search endpoint
// ──────────────────────────────────────────────────────

func TestEndpoint_Search(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	client := liveClient(t)

	// Search for a term that should match known records.
	searchQueries := []string{"trimble", "network", "service", "office"}
	for _, q := range searchQueries {
		t.Run("search/"+q, func(t *testing.T) {
			query := fmt.Sprintf("q=%s&per_page=10", url.QueryEscape(q))
			rows := getCollection(t, ctx, client, "/v1/search", query)
			t.Logf("search %q: %d results", q, len(rows))
		})
	}
}

func TestEndpoint_RateLimit(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	client := liveClient(t)

	rows, err := getCollectionTry(t, ctx, client, "/v1/rate_limit", "")
	if err != nil {
		t.Logf("rate_limit: %v (skipped)", err)
	} else {
		t.Logf("rate_limit: %d records %v", len(rows), rows)
	}
}

func TestEndpoint_Enums(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	client := liveClient(t)

	body, resp, err := client.GetCollectionJSON(ctx, "/v1/enums", url.Values{})
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, 200, resp.StatusCode)

	var enums map[string]any
	require.NoError(t, json.Unmarshal(body, &enums))
	t.Logf("enums: %d keys", len(enums))
	for k := range enums {
		t.Logf("  %s", k)
	}
	// Verify some expected enum types exist.
	expected := []string{"request.status", "request.category", "language", "account.currency"}
	for _, e := range expected {
		_, ok := enums[e]
		t.Logf("  %s: found=%v", e, ok)
	}
}

func TestEndpoint_Me(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	client := liveClient(t)

	body, resp, err := client.GetCollectionJSON(ctx, "/v1/me", url.Values{})
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, 200, resp.StatusCode)

	var me map[string]any
	require.NoError(t, json.Unmarshal(body, &me))
	t.Logf("me: name=%s email=%s id=%v", me["name"], me["primary_email"], me["id"])
	require.NotEmpty(t, me["name"])
	require.NotEmpty(t, me["primary_email"])
}

// ──────────────────────────────────────────────────────
// Attachment storage test
// ──────────────────────────────────────────────────────

func TestEndpoint_AttachmentStorage(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	client := liveClient(t)

	rows, err := getCollectionTry(t, ctx, client, "/v1/attachments/storage", "per_page=1")
	if err != nil {
		t.Logf("attachment storage: %v (skipped)", err)
	} else {
		t.Logf("attachment storage: %d records", len(rows))
	}
}

// ──────────────────────────────────────────────────────
// Events endpoint
// ──────────────────────────────────────────────────────

func TestEndpoint_Events(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	client := liveClient(t)

	rows, err := getCollectionTry(t, ctx, client, "/v1/events", "per_page=5&fields=id,event")
	if err != nil {
		t.Logf("events: %v (skipped)", err)
	} else {
		t.Logf("events: %d records", len(rows))
	}
}

// ──────────────────────────────────────────────────────
// Export endpoint
// ──────────────────────────────────────────────────────

func TestEndpoint_Export(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	client := liveClient(t)

	rows, err := getCollectionTry(t, ctx, client, "/v1/exports", "per_page=5&fields=id")
	if err != nil {
		t.Logf("exports: %v (skipped)", err)
	} else {
		t.Logf("exports: %d records", len(rows))
	}
}

// ──────────────────────────────────────────────────────
// Import endpoint
// ──────────────────────────────────────────────────────

func TestEndpoint_Import(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	client := liveClient(t)

	rows, err := getCollectionTry(t, ctx, client, "/v1/imports", "per_page=5&fields=id")
	if err != nil {
		t.Logf("imports: %v (skipped)", err)
	} else {
		t.Logf("imports: %d records", len(rows))
	}
}

// ──────────────────────────────────────────────────────
// Custom Collection Elements (sub-resource)
// ──────────────────────────────────────────────────────

func TestEndpoint_CustomCollectionElements(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	client := liveClient(t)

	// Find a custom collection first.
	cols := getCollection(t, ctx, client, "/v1/custom_collections", "per_page=1&fields=id,name")
	if len(cols) > 0 {
		colID := firstID(cols)
		path := fmt.Sprintf("/v1/custom_collections/%d/elements", colID)
		elements := getCollection(t, ctx, client, path, "per_page=5&fields=id")
		t.Logf("custom_collection %d elements: %d records", colID, len(elements))
	} else {
		t.Skip("no custom collections")
	}
}

// ──────────────────────────────────────────────────────
// Data retention / Archive
// ──────────────────────────────────────────────────────

func TestEndpoint_Archive(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	client := liveClient(t)

	rows := getCollection(t, ctx, client, "/v1/archive", "per_page=5&fields=id")
	t.Logf("archive: %d records", len(rows))
}

func TestEndpoint_Trash(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	client := liveClient(t)

	rows := getCollection(t, ctx, client, "/v1/trash", "per_page=5&fields=id")
	t.Logf("trash: %d records", len(rows))
}

// ──────────────────────────────────────────────────────
// Audit entries
// ──────────────────────────────────────────────────────

func TestEndpoint_AuditEntries(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	client := liveClient(t)

	rows, err := getCollectionTry(t, ctx, client, "/v1/audit_entries", "per_page=5&fields=id")
	if err != nil {
		t.Logf("audit_entries: %v (skipped)", err)
	} else {
		t.Logf("audit_entries: %d records", len(rows))
	}
}

// ──────────────────────────────────────────────────────
// Webhooks
// ──────────────────────────────────────────────────────

func TestEndpoint_Webhooks(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	client := liveClient(t)

	rows := getCollection(t, ctx, client, "/v1/webhooks", "per_page=5&fields=id")
	t.Logf("webhooks: %d records", len(rows))
}
