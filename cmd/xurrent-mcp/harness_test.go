// Package main provides a local integration harness for the xurrent-mcp server.
// It compiles and runs the server as a subprocess, connects via stdio transport,
// and exercises the full MCP protocol: initialize, list tools, call tools,
// list resources, read resources, and list prompts.
//
// Run with:
//
//	go test -run TestHarness -v -count=1 ./cmd/xurrent-mcp/
package main

import (
	"context"
	"encoding/json"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/require"
)

// spawnServer builds and starts the xurrent-mcp server as a subprocess.
func spawnServer(t *testing.T, ctx context.Context) *mcp.ClientSession {
	t.Helper()

	_, thisFile, _, ok := runtime.Caller(0)
	require.True(t, ok)
	repoRoot := filepath.Join(filepath.Dir(thisFile), "..", "..")

	client := mcp.NewClient(&mcp.Implementation{Name: "harness", Version: "0.0.1"}, nil)
	cmd := exec.CommandContext(ctx, "go", "run", "./cmd/xurrent-mcp")
	cmd.Dir = repoRoot
	transport := &mcp.CommandTransport{Command: cmd}
	session, err := client.Connect(ctx, transport, nil)
	require.NoError(t, err)
	t.Cleanup(func() { _ = session.Close() })
	return session
}

func TestHarness_Initialize(t *testing.T) {
	if testing.Short() {
		t.Skip("spawns subprocess")
	}
	//t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	session := spawnServer(t, ctx)
	result := session.InitializeResult()
	require.NotNil(t, result)
	t.Logf("server: %s v%s", result.ServerInfo.Name, result.ServerInfo.Version)
	require.Equal(t, "xurrent-mcp", result.ServerInfo.Name)
}

func TestHarness_ListTools(t *testing.T) {
	if testing.Short() {
		t.Skip("spawns subprocess")
	}
	//t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	session := spawnServer(t, ctx)

	tools, err := session.ListTools(ctx, nil)
	require.NoError(t, err)
	require.NotNil(t, tools)
	require.NotEmpty(t, tools.Tools, "server should register tools")

	toolNames := make(map[string]bool)
	for _, tool := range tools.Tools {
		toolNames[tool.Name] = true
		t.Logf("  tool: %s — %s", tool.Name, tool.Description)
	}

	// Verify core tools are registered.
	expected := []string{
		"xurrent_discover",
		"xurrent_query",
		"xurrent_read",
		"xurrent_search",
		"xurrent_rate_limit",
		"xurrent_create",
		"xurrent_update",
		"xurrent_delete",
		"xurrent_draft",
		"xurrent_detect_collision",
		"xurrent_find_candidates",
		"xurrent_validate_automation",
		"xurrent_validate_ui_extension",
	}
	for _, name := range expected {
		require.True(t, toolNames[name], "tool %q should be registered", name)
	}
}

func TestHarness_CallDiscover(t *testing.T) {
	if testing.Short() {
		t.Skip("spawns subprocess")
	}
	//t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	session := spawnServer(t, ctx)

	res, err := session.CallTool(ctx, &mcp.CallToolParams{
		Name:      "xurrent_discover",
		Arguments: map[string]any{},
	})
	require.NoError(t, err)
	require.False(t, res.IsError, "xurrent_discover returned error")

	// StructuredContent should contain the entity list.
	raw, err := json.Marshal(res.StructuredContent)
	require.NoError(t, err)

	var out struct {
		Entities []map[string]any `json:"entities"`
		Count    int              `json:"count"`
	}
	require.NoError(t, json.Unmarshal(raw, &out))
	require.Greater(t, out.Count, 10, "should have at least 10 entities")
	t.Logf("discovered %d entities", out.Count)
}

func TestHarness_CallDiscover_SpecificEntity(t *testing.T) {
	if testing.Short() {
		t.Skip("spawns subprocess")
	}
	//t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	session := spawnServer(t, ctx)

	res, err := session.CallTool(ctx, &mcp.CallToolParams{
		Name: "xurrent_discover",
		Arguments: map[string]any{
			"entity": "team",
		},
	})
	require.NoError(t, err)
	require.False(t, res.IsError)

	raw, _ := json.Marshal(res.StructuredContent)
	var out struct {
		Entity string `json:"entity"`
		Name   string `json:"name"`
		Path   string `json:"path"`
	}
	require.NoError(t, json.Unmarshal(raw, &out))
	require.Equal(t, "team", out.Entity)
	require.Equal(t, "Team", out.Name)
	require.Equal(t, "/v1/teams", out.Path)
}

func TestHarness_CallDetectCollision(t *testing.T) {
	if testing.Short() {
		t.Skip("spawns subprocess")
	}
	//t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	session := spawnServer(t, ctx)

	res, err := session.CallTool(ctx, &mcp.CallToolParams{
		Name: "xurrent_detect_collision",
		Arguments: map[string]any{
			"entity": "team",
			"field":  "name",
		},
	})
	require.NoError(t, err)
	require.False(t, res.IsError)

	raw, _ := json.Marshal(res.StructuredContent)
	var out struct {
		Entity string `json:"entity"`
		Field  string `json:"field"`
		Steps  []struct {
			Path string `json:"path"`
		} `json:"steps"`
	}
	require.NoError(t, json.Unmarshal(raw, &out))
	require.Equal(t, "team", out.Entity)
	require.Equal(t, "name", out.Field)
	require.NotEmpty(t, out.Steps)
	require.Contains(t, out.Steps[0].Path, "/v1/teams")
}

func TestHarness_CallValidateAutomation(t *testing.T) {
	if testing.Short() {
		t.Skip("spawns subprocess")
	}
	//t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	session := spawnServer(t, ctx)

	res, err := session.CallTool(ctx, &mcp.CallToolParams{
		Name: "xurrent_validate_automation",
		Arguments: map[string]any{
			"expressions": "is_assigned: status = assigned",
			"condition":   "is_assigned",
			"actions":     "a1: add note 'test'",
			"trigger":     "on status update",
		},
	})
	require.NoError(t, err)
	require.False(t, res.IsError)

	raw, _ := json.Marshal(res.StructuredContent)
	var out struct {
		Valid  bool     `json:"valid"`
		Issues []string `json:"issues"`
	}
	require.NoError(t, json.Unmarshal(raw, &out))
	require.True(t, out.Valid, "valid automation should have no issues, got: %v", out.Issues)
}

func TestHarness_CallValidateAutomation_Errors(t *testing.T) {
	if testing.Short() {
		t.Skip("spawns subprocess")
	}
	//t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	session := spawnServer(t, ctx)

	res, err := session.CallTool(ctx, &mcp.CallToolParams{
		Name: "xurrent_validate_automation",
		Arguments: map[string]any{
			"condition": "status = assigned",
			"actions":   "", // empty actions
			"trigger":   "on status update",
		},
	})
	require.NoError(t, err)
	require.False(t, res.IsError)

	raw, _ := json.Marshal(res.StructuredContent)
	var out struct {
		Valid  bool     `json:"valid"`
		Issues []string `json:"issues"`
	}
	require.NoError(t, json.Unmarshal(raw, &out))
	require.False(t, out.Valid, "empty actions should produce validation issues")
	require.NotEmpty(t, out.Issues)
}

func TestHarness_CallValidateUIExtension(t *testing.T) {
	if testing.Short() {
		t.Skip("spawns subprocess")
	}
	//t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	session := spawnServer(t, ctx)

	res, err := session.CallTool(ctx, &mcp.CallToolParams{
		Name: "xurrent_validate_ui_extension",
		Arguments: map[string]any{
			"css":      ".row.subject { display: none; }",
			"html":     "<div class='row vertical'><label>Test</label></div>",
			"category": "request_template",
		},
	})
	require.NoError(t, err)
	require.False(t, res.IsError)

	raw, _ := json.Marshal(res.StructuredContent)
	var out struct {
		Valid  bool     `json:"valid"`
		Issues []string `json:"issues"`
	}
	require.NoError(t, json.Unmarshal(raw, &out))
	require.True(t, out.Valid, "valid UI extension should have no issues, got: %v", out.Issues)
}

func TestHarness_CallDraft(t *testing.T) {
	if testing.Short() {
		t.Skip("spawns subprocess")
	}
	//t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	session := spawnServer(t, ctx)

	// List drafts (should be empty).
	res, err := session.CallTool(ctx, &mcp.CallToolParams{
		Name: "xurrent_draft",
		Arguments: map[string]any{
			"action": "list",
		},
	})
	require.NoError(t, err)
	require.False(t, res.IsError)

	raw, _ := json.Marshal(res.StructuredContent)
	var out struct {
		Drafts []any `json:"drafts"`
		Count  int   `json:"count"`
	}
	require.NoError(t, json.Unmarshal(raw, &out))
	require.Equal(t, 0, out.Count, "drafts should be empty initially")
}

func TestHarness_ListResources(t *testing.T) {
	if testing.Short() {
		t.Skip("spawns subprocess")
	}
	//t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	session := spawnServer(t, ctx)

	resources, err := session.ListResources(ctx, nil)
	require.NoError(t, err)
	require.NotNil(t, resources)
	require.NotEmpty(t, resources.Resources, "server should register resources")

	uris := make(map[string]bool)
	for _, r := range resources.Resources {
		uris[r.URI] = true
		t.Logf("  resource: %s — %s", r.URI, r.Name)
	}

	expected := []string{
		"xurrent://domain/graph",
		"xurrent://docs/index",
		"xurrent://docs/ui-extensions",
		"xurrent://docs/automation",
	}
	for _, uri := range expected {
		require.True(t, uris[uri], "resource %q should be registered", uri)
	}
}

func TestHarness_ReadResource(t *testing.T) {
	if testing.Short() {
		t.Skip("spawns subprocess")
	}
	//t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	session := spawnServer(t, ctx)

	res, err := session.ReadResource(ctx, &mcp.ReadResourceParams{
		URI: "xurrent://domain/graph",
	})
	require.NoError(t, err)
	require.NotNil(t, res)
	require.NotEmpty(t, res.Contents, "domain graph resource should have content")
	require.Contains(t, res.Contents[0].Text, "organization", "should contain entity names")
}

func TestHarness_ListPrompts(t *testing.T) {
	if testing.Short() {
		t.Skip("spawns subprocess")
	}
	//t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	session := spawnServer(t, ctx)

	prompts, err := session.ListPrompts(ctx, nil)
	require.NoError(t, err)
	require.NotNil(t, prompts)
	require.NotEmpty(t, prompts.Prompts, "server should register prompts")

	names := make(map[string]bool)
	for _, p := range prompts.Prompts {
		names[p.Name] = true
		t.Logf("  prompt: %s — %s", p.Name, p.Description)
	}

	expected := []string{
		"itsm-relationships",
		"ui-extension-development",
		"automation-rule-development",
	}
	for _, name := range expected {
		require.True(t, names[name], "prompt %q should be registered", name)
	}
}

func TestHarness_GetPrompt(t *testing.T) {
	if testing.Short() {
		t.Skip("spawns subprocess")
	}
	//t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	session := spawnServer(t, ctx)

	res, err := session.GetPrompt(ctx, &mcp.GetPromptParams{
		Name: "itsm-relationships",
	})
	require.NoError(t, err)
	require.NotNil(t, res)
	require.NotEmpty(t, res.Messages)
	// Verify content references key entities.
	if tc, ok := res.Messages[0].Content.(*mcp.TextContent); ok {
		require.Contains(t, tc.Text, "Service Delivery Chain")
		require.Contains(t, tc.Text, "Request Flow")
		require.Contains(t, tc.Text, "CMDB")
	}
}
