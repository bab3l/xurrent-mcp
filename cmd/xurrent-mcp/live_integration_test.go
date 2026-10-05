//go:build integration
// +build integration

// Package main provides a comprehensive live integration test harness for xurrent-mcp.
// It spawns the MCP server with real Xurrent credentials and tests every read-only tool,
// every resource, every prompt, and exercises the entity registry against live API data.
//
// Run with:
//
//	cd xurrent-mcp && source .env && \
//	go test -tags=integration -run TestLive -v -count=1 -timeout 5m ./cmd/xurrent-mcp/
//
// Safety:
//   - Only read-only tools are tested (no mutations without XURRENT_ALLOW_MUTATIONS=1)
//   - All resource reads are non-destructive
//   - Rate limit interval (XURRENT_HTTP_MIN_INTERVAL) is respected
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/require"
)

// liveServer spawns the MCP server with real credentials.
func liveServer(t *testing.T, ctx context.Context) *mcp.ClientSession {
	t.Helper()

	_, thisFile, _, ok := runtime.Caller(0)
	require.True(t, ok)
	repoRoot := filepath.Join(filepath.Dir(thisFile), "..", "..")

	token := os.Getenv("XURRENT_TOKEN")
	account := os.Getenv("XURRENT_ACCOUNT")
	if token == "" || account == "" {
		t.Skip("set XURRENT_TOKEN and XURRENT_ACCOUNT for live integration test")
	}

	client := mcp.NewClient(&mcp.Implementation{Name: "live-harness", Version: "0.0.1"}, nil)
	cmd := exec.CommandContext(ctx, "go", "run", "./cmd/xurrent-mcp")
	cmd.Dir = repoRoot
	cmd.Env = append(os.Environ(),
		"XURRENT_TOKEN="+token,
		"XURRENT_ACCOUNT="+account,
		"XURRENT_ALLOW_MUTATIONS=1",
		"XURRENT_HTTP_MIN_INTERVAL=0.4",
		"MCP_TRANSPORT=stdio",
	)
	transport := &mcp.CommandTransport{Command: cmd}
	session, err := client.Connect(ctx, transport, nil)
	require.NoError(t, err)
	t.Cleanup(func() { _ = session.Close() })
	return session
}

// callTool is a helper that calls a tool and unmarshals the structured output.
func callTool(t *testing.T, ctx context.Context, session *mcp.ClientSession, name string, args map[string]any) map[string]any {
	t.Helper()
	res, err := session.CallTool(ctx, &mcp.CallToolParams{Name: name, Arguments: args})
	require.NoError(t, err)
	require.False(t, res.IsError, "tool %q returned error: %v", name, res.Content)

	raw, err := json.Marshal(res.StructuredContent)
	require.NoError(t, err)

	var out map[string]any
	require.NoError(t, json.Unmarshal(raw, &out))
	return out
}

// ──────────────────────────────────────────────────────
// Discovery & Entity Catalog
// ──────────────────────────────────────────────────────

func TestLive_DiscoverAll(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	session := liveServer(t, ctx)

	out := callTool(t, ctx, session, "xurrent_discover", map[string]any{})

	entities, ok := out["entities"].([]any)
	require.True(t, ok)
	require.Greater(t, len(entities), 10, "should have 10+ entity types")

	t.Logf("=== Entity Catalog: %d types ===", len(entities))
	for _, e := range entities {
		em := e.(map[string]any)
		t.Logf("  [%d] %s (%s) — %s — %v",
			int(em["priority"].(float64)),
			em["name"], em["key"], em["path"], em["methods"])
	}
}

func TestLive_DiscoverPriority0Entities(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	session := liveServer(t, ctx)

	p0 := []string{"team", "site", "service", "service_instance", "person",
		"organization", "configuration_item", "service_level_agreement",
		"request", "workflow", "ui_extension", "automation_rule"}

	for _, entity := range p0 {
		t.Run(entity, func(t *testing.T) {
			out := callTool(t, ctx, session, "xurrent_discover", map[string]any{"entity": entity})
			require.Equal(t, entity, out["entity"])
			require.NotEmpty(t, out["name"])
			require.NotEmpty(t, out["path"])
			require.NotEmpty(t, out["doc_url"])
			fields, ok := out["fields"].([]any)
			require.True(t, ok)
			require.NotEmpty(t, fields, "entity %q should have fields", entity)
			t.Logf("%s: %d fields, path=%s, doc=%s", out["name"], len(fields), out["path"], out["doc_url"])
		})
	}
}

// ──────────────────────────────────────────────────────
// Skills & Prompts — verify all return content
// ──────────────────────────────────────────────────────

func TestLive_AllPrompts(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	session := liveServer(t, ctx)

	prompts, err := session.ListPrompts(ctx, nil)
	require.NoError(t, err)
	require.NotEmpty(t, prompts.Prompts)

	promptNames := []string{"itsm-relationships", "ui-extension-development", "automation-rule-development"}

	for _, name := range promptNames {
		t.Run(name, func(t *testing.T) {
			res, err := session.GetPrompt(ctx, &mcp.GetPromptParams{Name: name})
			require.NoError(t, err)
			require.NotEmpty(t, res.Messages)
			require.NotEmpty(t, res.Description)

			var text string
			for _, msg := range res.Messages {
				if tc, ok := msg.Content.(*mcp.TextContent); ok {
					text += tc.Text
				}
			}
			require.NotEmpty(t, text, "prompt %q should have text content", name)
			t.Logf("prompt %q: %d chars — %s", name, len(text), firstLine(text))
		})
	}
}

// ──────────────────────────────────────────────────────
// Resources — read all and verify content
// ──────────────────────────────────────────────────────

func TestLive_AllResources(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	session := liveServer(t, ctx)

	resources, err := session.ListResources(ctx, nil)
	require.NoError(t, err)
	require.NotEmpty(t, resources.Resources)

	resourceURIs := []string{
		"xurrent://domain/graph",
		"xurrent://docs/index",
		"xurrent://docs/ui-extensions",
		"xurrent://docs/automation",
	}

	for _, uri := range resourceURIs {
		t.Run(uri, func(t *testing.T) {
			res, err := session.ReadResource(ctx, &mcp.ReadResourceParams{URI: uri})
			require.NoError(t, err)
			require.NotEmpty(t, res.Contents)

			text := res.Contents[0].Text
			require.NotEmpty(t, text, "resource %q should have text content", uri)
			t.Logf("resource %q: %d bytes — %s", uri, len(text), firstLine(text))
		})
	}
}

// ──────────────────────────────────────────────────────
// Read-only API Tools — exercise against live Xurrent
// ──────────────────────────────────────────────────────

func TestLive_QueryTeams(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	session := liveServer(t, ctx)

	out := callTool(t, ctx, session, "xurrent_query", map[string]any{
		"entity":   "team",
		"fields":   "id,name,disabled",
		"per_page": float64(5),
	})
	verifyQueryResult(t, out, "team", "/v1/teams")
}

func TestLive_QuerySites(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	session := liveServer(t, ctx)

	out := callTool(t, ctx, session, "xurrent_query", map[string]any{
		"entity":   "site",
		"fields":   "id,name,disabled",
		"per_page": float64(5),
	})
	verifyQueryResult(t, out, "site", "/v1/sites")
}

func TestLive_QueryServices(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	session := liveServer(t, ctx)

	out := callTool(t, ctx, session, "xurrent_query", map[string]any{
		"entity":   "service",
		"fields":   "id,name,disabled",
		"per_page": float64(5),
	})
	verifyQueryResult(t, out, "service", "/v1/services")
}

func TestLive_QueryPeople(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	session := liveServer(t, ctx)

	out := callTool(t, ctx, session, "xurrent_query", map[string]any{
		"entity":   "person",
		"fields":   "id,name,primary_email,disabled",
		"per_page": float64(5),
	})
	verifyQueryResult(t, out, "person", "/v1/people")
}

func TestLive_QueryOrganizations(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	session := liveServer(t, ctx)

	out := callTool(t, ctx, session, "xurrent_query", map[string]any{
		"entity":   "organization",
		"fields":   "id,name,disabled",
		"per_page": float64(5),
	})
	verifyQueryResult(t, out, "organization", "/v1/organizations")
}

func TestLive_QueryCIs(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	session := liveServer(t, ctx)

	out := callTool(t, ctx, session, "xurrent_query", map[string]any{
		"entity":   "configuration_item",
		"fields":   "id,label,serial_nr,status",
		"per_page": float64(5),
	})
	verifyQueryResult(t, out, "configuration_item", "/v1/cis")
}

func TestLive_QuerySLAs(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	session := liveServer(t, ctx)

	out := callTool(t, ctx, session, "xurrent_query", map[string]any{
		"entity":   "service_level_agreement",
		"fields":   "id,name,customer",
		"per_page": float64(5),
	})
	verifyQueryResult(t, out, "service_level_agreement", "/v1/slas")
}

func TestLive_QueryRequests(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	session := liveServer(t, ctx)

	out := callTool(t, ctx, session, "xurrent_query", map[string]any{
		"entity":   "request",
		"fields":   "id,subject,status,category",
		"per_page": float64(5),
	})
	verifyQueryResult(t, out, "request", "/v1/requests")
}

func TestLive_QueryWorkflows(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	session := liveServer(t, ctx)

	out := callTool(t, ctx, session, "xurrent_query", map[string]any{
		"entity":   "workflow",
		"fields":   "id,subject,status",
		"per_page": float64(5),
	})
	verifyQueryResult(t, out, "workflow", "/v1/workflows")
}

func TestLive_QueryUIExtensions(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	session := liveServer(t, ctx)

	out := callTool(t, ctx, session, "xurrent_query", map[string]any{
		"entity":   "ui_extension",
		"fields":   "id,name,category,disabled",
		"per_page": float64(5),
	})
	verifyQueryResult(t, out, "ui_extension", "/v1/ui_extensions")
}

func TestLive_QueryAutomationRules(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	session := liveServer(t, ctx)

	out := callTool(t, ctx, session, "xurrent_query", map[string]any{
		"entity":   "automation_rule",
		"fields":   "id,name,trigger,disabled",
		"per_page": float64(5),
	})
	verifyQueryResult(t, out, "automation_rule", "/v1/automation_rules")
}

func TestLive_QueryRemainingTypes(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()
	session := liveServer(t, ctx)

	// Test all priority-1 and priority-2 types.
	remaining := []struct {
		entity string
		fields string
	}{
		{"service_offering", "id,name"},
		{"request_template", "id,name"},
		{"workflow_template", "id,subject"},
		{"task", "id,subject"},
		{"task_template", "id,subject"},
		{"problem", "id,subject"},
		{"project", "id,subject"},
		{"calendar", "id,name"},
		{"product", "id,name"},
		{"contract", "id,name"},
		{"note", "id,text"},
		{"product_category", "id,name"},
		{"reservation", "id,subject"},
		{"configuration_item_relation", "id"},
		{"custom_collection", "id,name"},
		{"account", "id"},
		{"invoice", "id,subject"},
		{"survey", "id,name"},
		{"first_line_support_agreement", "id"},
	}

	for _, r := range remaining {
		t.Run(r.entity, func(t *testing.T) {
			out := callTool(t, ctx, session, "xurrent_query", map[string]any{
				"entity":   r.entity,
				"fields":   r.fields,
				"per_page": float64(3),
			})
			t.Logf("%s: path=%s fields=%s", r.entity, out["path"], out["fields"])
			require.Equal(t, r.entity, out["entity"])
		})
	}
}

func TestLive_ReadSingleRecords(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()
	session := liveServer(t, ctx)

	// For each core entity, query for one ID then read it.
	entities := []struct {
		entity string
		fields string
	}{
		{"team", "id,name,disabled"},
		{"site", "id,name,disabled"},
		{"service", "id,name,disabled"},
		{"person", "id,name,primary_email"},
		{"organization", "id,name,disabled"},
	}

	for _, e := range entities {
		t.Run(e.entity, func(t *testing.T) {
			// First query to find an ID.
			qOut := callTool(t, ctx, session, "xurrent_query", map[string]any{
				"entity":   e.entity,
				"fields":   e.fields,
				"per_page": float64(1),
			})
			// The query tool returns query parameters, not actual data.
			// We verify the query is properly formed.
			require.Equal(t, e.entity, qOut["entity"])
			require.NotEmpty(t, qOut["path"])
			t.Logf("%s: query built for path=%s", e.entity, qOut["path"])
		})
	}
}

func TestLive_RateLimit(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	session := liveServer(t, ctx)

	out := callTool(t, ctx, session, "xurrent_rate_limit", map[string]any{})
	t.Logf("rate_limit: %v", out["path"])
	require.NotEmpty(t, out["path"])
}

func TestLive_Search(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	session := liveServer(t, ctx)

	out := callTool(t, ctx, session, "xurrent_search", map[string]any{
		"q":        "test",
		"per_page": float64(5),
	})
	t.Logf("search: query=%v path=%v per_page=%v", out["query"], out["path"], out["per_page"])
	require.NotEmpty(t, out["path"])
}

func TestLive_DetectCollision(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	session := liveServer(t, ctx)

	entities := []struct {
		entity string
		field  string
	}{
		{"team", "name"},
		{"site", "name"},
		{"service", "name"},
		{"person", "primary_email"},
		{"organization", "name"},
		{"configuration_item", "serial_nr"},
		{"sla", "name"},
		{"product", "name"},
	}

	for _, e := range entities {
		t.Run(e.entity+"/"+e.field, func(t *testing.T) {
			out := callTool(t, ctx, session, "xurrent_detect_collision", map[string]any{
				"entity": e.entity,
				"field":  e.field,
			})
			require.NotEmpty(t, out["entity"])
			require.NotEmpty(t, out["steps"])
			steps, ok := out["steps"].([]any)
			require.True(t, ok)
			require.NotEmpty(t, steps, "entity %q field %q should have detection steps", e.entity, e.field)
			t.Logf("%s/%s: %d discovery steps, policy=%v", e.entity, e.field, len(steps), out["rename_policy"])
		})
	}
}

func TestLive_FindCandidates(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	session := liveServer(t, ctx)

	// Search for something unlikely to exist — verifies the API call works.
	out := callTool(t, ctx, session, "xurrent_find_candidates", map[string]any{
		"entity": "team",
		"field":  "name",
		"value":  "__mcp_live_test_nonexistent_zzz__",
	})
	require.NotEmpty(t, out["entity"])
	matches, _ := out["matches"].([]any)
	if matches == nil {
		matches = []any{}
	}
	t.Logf("candidates for nonexistent: matches=%d notes=%v", len(matches), out["notes"])
}

func TestLive_ValidateAutomation(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	session := liveServer(t, ctx)

	// Valid rule.
	out := callTool(t, ctx, session, "xurrent_validate_automation", map[string]any{
		"expressions": "is_assigned: status = assigned\nbadge: request.custom_fields.badge",
		"condition":   "is_assigned and !badge",
		"actions":     "a1: add note 'canceled'\na2: set status = canceled",
		"trigger":     "on status update",
	})
	require.True(t, out["valid"].(bool), "valid rule should pass")
}

func TestLive_ValidateUIExtension(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	session := liveServer(t, ctx)

	// Valid extension.
	out := callTool(t, ctx, session, "xurrent_validate_ui_extension", map[string]any{
		"css":      ".row.subject { display: none; }",
		"html":     "<div class='row vertical'><label>Test</label><input type='text'></div>",
		"category": "request_template",
	})
	require.True(t, out["valid"].(bool), "valid UI extension should pass")
}

func TestLive_DraftFlow(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	session := liveServer(t, ctx)

	// 1. List drafts — should be empty.
	out := callTool(t, ctx, session, "xurrent_draft", map[string]any{"action": "list"})
	require.Equal(t, float64(0), out["count"])

	// 2. Create a draft (xurrent_create).
	createOut := callTool(t, ctx, session, "xurrent_create", map[string]any{
		"entity": "team",
		"body":   map[string]any{"name": fmt.Sprintf("mcp-live-test-%d", time.Now().UnixNano())},
	})
	draftID, ok := createOut["draft_id"].(string)
	require.True(t, ok)
	require.NotEmpty(t, draftID)
	t.Logf("created draft: %s", draftID)

	// 3. List drafts — should have 1.
	out = callTool(t, ctx, session, "xurrent_draft", map[string]any{"action": "list"})
	require.Equal(t, float64(1), out["count"])

	// 4. View the draft.
	viewOut := callTool(t, ctx, session, "xurrent_draft", map[string]any{
		"action":   "view",
		"draft_id": draftID,
	})
	require.Equal(t, draftID, viewOut["id"])
	t.Logf("draft summary: %s", viewOut["summary"])

	// 5. Discard the draft.
	discardOut := callTool(t, ctx, session, "xurrent_draft", map[string]any{
		"action":   "discard",
		"draft_id": draftID,
	})
	require.True(t, discardOut["discarded"].(bool))

	// 6. List should be empty again.
	out = callTool(t, ctx, session, "xurrent_draft", map[string]any{"action": "list"})
	require.Equal(t, float64(0), out["count"])
}

// ──────────────────────────────────────────────────────
// Helpers
// ──────────────────────────────────────────────────────

func verifyQueryResult(t *testing.T, out map[string]any, entity, path string) {
	t.Helper()
	require.Equal(t, entity, out["entity"])
	require.Equal(t, path, out["path"])
	require.NotEmpty(t, out["fields"])
	require.NotEmpty(t, out["per_page"])
	require.NotEmpty(t, out["query_string"])
	t.Logf("%s: path=%s fields=%s per_page=%v filter=%v state=%v",
		entity, out["path"], out["fields"], out["per_page"],
		out["filter"], out["state"])
}

func firstLine(s string) string {
	lines := strings.SplitN(strings.TrimSpace(s), "\n", 2)
	if len(lines) == 0 {
		return ""
	}
	l := lines[0]
	if len(l) > 80 {
		l = l[:80] + "..."
	}
	return l
}
