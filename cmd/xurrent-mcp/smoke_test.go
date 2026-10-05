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

func TestSmoke_MCP_detect_identifier_collision(t *testing.T) {
	if testing.Short() {
		t.Skip("spawns subprocess")
	}
	t.Parallel()

	_, thisFile, _, ok := runtime.Caller(0)
	require.True(t, ok)
	// cmd/xurrent-mcp -> repo root
	repoRoot := filepath.Join(filepath.Dir(thisFile), "..", "..")

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	client := mcp.NewClient(&mcp.Implementation{Name: "smoke-test", Version: "0.0.1"}, nil)
	cmd := exec.CommandContext(ctx, "go", "run", "./cmd/xurrent-mcp")
	cmd.Dir = repoRoot
	transport := &mcp.CommandTransport{Command: cmd}
	session, err := client.Connect(ctx, transport, nil)
	require.NoError(t, err)
	defer func() { _ = session.Close() }()

	res, err := session.CallTool(ctx, &mcp.CallToolParams{
		Name: "xurrent_detect_collision",
		Arguments: map[string]any{
			"entity": "team",
			"field":  "name",
		},
	})
	require.NoError(t, err)
	require.False(t, res.IsError, "tool returned error")

	// Structured output from ToolHandlerFor
	raw, err := json.Marshal(res.StructuredContent)
	require.NoError(t, err)
	t.Logf("structured: %s", string(raw))

	var out struct {
		Entity string `json:"entity"`
		Field  string `json:"field"`
		Steps  []struct {
			Path string `json:"path"`
		} `json:"steps"`
	}
	require.NoError(t, json.Unmarshal(raw, &out))
	if len(out.Steps) == 0 && len(res.Content) > 0 {
		for _, c := range res.Content {
			if tc, ok := c.(*mcp.TextContent); ok {
				t.Logf("fallback text content: %s", tc.Text)
				require.NoError(t, json.Unmarshal([]byte(tc.Text), &out))
				break
			}
		}
	}
	require.Equal(t, "team", out.Entity)
	require.Equal(t, "name", out.Field)
	require.NotEmpty(t, out.Steps)
	require.Contains(t, out.Steps[0].Path, "/v1/teams")
}
