//go:build integration
// +build integration

// Package main provides targeted live data fetch tests for key Xurrent entities.
// Verifies that the MCP server correctly discovers entity metadata and that
// the underlying go-xurrent client can fetch real records from the Xurrent API.
//
// Run with:
//
//	cd xurrent-mcp && source .env && \
//	go test -tags=integration -run TestFetch -v -count=1 -timeout 3m ./cmd/xurrent-mcp/
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
	"strings"
	"testing"
	"time"

	openapiclient "github.com/bab3l/go-xurrent"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/require"
)

// ──────────────────────────────────────────────────────
// MCP client helper
// ──────────────────────────────────────────────────────

func fetchMCPClient(t *testing.T, ctx context.Context) *mcp.ClientSession {
	t.Helper()

	_, thisFile, _, ok := runtime.Caller(0)
	require.True(t, ok)
	repoRoot := filepath.Join(filepath.Dir(thisFile), "..", "..")

	token := os.Getenv("XURRENT_TOKEN")
	account := os.Getenv("XURRENT_ACCOUNT")
	if token == "" || account == "" {
		t.Skip("set XURRENT_TOKEN and XURRENT_ACCOUNT")
	}

	client := mcp.NewClient(&mcp.Implementation{Name: "fetch-verify", Version: "0.0.1"}, nil)
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

func mcpCall(t *testing.T, ctx context.Context, session *mcp.ClientSession, name string, args map[string]any) map[string]any {
	t.Helper()
	res, err := session.CallTool(ctx, &mcp.CallToolParams{Name: name, Arguments: args})
	require.NoError(t, err)
	require.False(t, res.IsError, "tool %q returned error", name)

	raw, err := json.Marshal(res.StructuredContent)
	require.NoError(t, err)
	var out map[string]any
	require.NoError(t, json.Unmarshal(raw, &out))
	return out
}

// ──────────────────────────────────────────────────────
// go-xurrent direct API helper
// ──────────────────────────────────────────────────────

func apiClient(t *testing.T) *openapiclient.APIClient {
	t.Helper()
	token := os.Getenv("XURRENT_TOKEN")
	account := os.Getenv("XURRENT_ACCOUNT")
	require.NotEmpty(t, token)
	require.NotEmpty(t, account)

	cfg := openapiclient.NewConfiguration()
	cfg.AddDefaultHeader("Authorization", "Bearer "+token)
	cfg.AddDefaultHeader("X-4me-Account", account)
	return openapiclient.NewAPIClient(cfg)
}

func apiGet(t *testing.T, ctx context.Context, client *openapiclient.APIClient, path, query string) []map[string]any {
	t.Helper()
	q, _ := url.ParseQuery(query)
	body, resp, err := client.GetCollectionJSON(ctx, path, q)
	require.NoError(t, err)
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		t.Logf("GET %s: HTTP %d", path, resp.StatusCode)
		return nil
	}

	var rows []map[string]any
	if err := json.Unmarshal(body, &rows); err != nil {
		var single map[string]any
		require.NoError(t, json.Unmarshal(body, &single))
		rows = []map[string]any{single}
	}
	return rows
}

func apiID(m map[string]any) int64 {
	if v, ok := m["id"]; ok {
		switch n := v.(type) {
		case float64:
			return int64(n)
		}
	}
	return 0
}

func apiStr(m map[string]any, key string) string {
	if v, ok := m[key]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

// ──────────────────────────────────────────────────────
// Core Entity Fetch Tests
// ──────────────────────────────────────────────────────

func TestFetch_Person(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	session := fetchMCPClient(t, ctx)

	// 1. Discover entity metadata.
	out := mcpCall(t, ctx, session, "xurrent_discover", map[string]any{"entity": "person"})
	t.Logf("=== %s ===", out["name"])
	t.Logf("  path:     %s", out["path"])
	t.Logf("  doc:      %s", out["doc_url"])
	t.Logf("  unique:   %s", out["unique_key"])
	t.Logf("  fields:   %d total", len(out["fields"].([]any)))
	t.Logf("  defaults: %v", out["default_fields"])

	rels, _ := out["relationships"].([]any)
	for _, r := range rels {
		rm := r.(map[string]any)
		t.Logf("  → %s (%s) via %s: %s", rm["entity"], rm["cardinality"], rm["via"], rm["description"])
	}

	// 2. Build query via MCP tool.
	qOut := mcpCall(t, ctx, session, "xurrent_query", map[string]any{
		"entity":   "person",
		"fields":   "id,name,primary_email,disabled",
		"per_page": float64(10),
	})
	require.Equal(t, "person", qOut["entity"])
	require.GreaterOrEqual(t, int(qOut["count"].(float64)), 1, "expected records for person")
	t.Logf("  MCP query: %d records returned", int(qOut["count"].(float64)))

	// 3. Fetch real records via go-xurrent.
	client := apiClient(t)
	records := apiGet(t, ctx, client, "/v1/people", "per_page=10&fields=id,name,primary_email,disabled")
	require.GreaterOrEqual(t, len(records), 1, "expected at least 1 person")
	t.Logf("  records: %d", len(records))

	for i := 0; i < min(3, len(records)); i++ {
		t.Logf("  [%d] id=%d name=%s email=%s",
			i+1, apiID(records[i]), apiStr(records[i], "name"), apiStr(records[i], "primary_email"))
	}

	// 4. Read first person by ID.
	firstID := apiID(records[0])
	detail := apiGet(t, ctx, client,
		fmt.Sprintf("/v1/people/%d", firstID),
		"fields=id,name,primary_email,organization,site")
	t.Logf("  detail[%d]: %v", firstID, detail[0])
}

func TestFetch_ConfigurationItem(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	session := fetchMCPClient(t, ctx)

	out := mcpCall(t, ctx, session, "xurrent_discover", map[string]any{"entity": "configuration_item"})
	t.Logf("=== %s ===", out["name"])
	t.Logf("  path:     %s", out["path"])
	t.Logf("  doc:      %s", out["doc_url"])
	t.Logf("  unique:   %s", out["unique_key"])
	t.Logf("  defaults: %v", out["default_fields"])

	qOut := mcpCall(t, ctx, session, "xurrent_query", map[string]any{
		"entity":   "configuration_item",
		"fields":   "id,label,serial_nr,status",
		"per_page": float64(10),
	})
	require.Equal(t, "configuration_item", qOut["entity"])
	t.Logf("  MCP query: %d records returned", int(qOut["count"].(float64)))

	client := apiClient(t)
	records := apiGet(t, ctx, client, "/v1/cis", "per_page=10&fields=id,label,serial_nr,status")
	require.GreaterOrEqual(t, len(records), 1, "expected at least 1 CI")
	t.Logf("  records: %d", len(records))

	for i := 0; i < min(3, len(records)); i++ {
		t.Logf("  [%d] id=%d label=%s serial=%s status=%s",
			i+1, apiID(records[i]), apiStr(records[i], "label"),
			apiStr(records[i], "serial_nr"), apiStr(records[i], "status"))
	}

	// Read first CI by ID (including product reference).
	firstID := apiID(records[0])
	detail := apiGet(t, ctx, client,
		fmt.Sprintf("/v1/cis/%d", firstID),
		"fields=id,label,serial_nr,status,product,site")
	t.Logf("  detail[%d]: id=%d label=%s serial=%s status=%s product=%v site=%v",
		firstID,
		apiID(detail[0]),
		apiStr(detail[0], "label"),
		apiStr(detail[0], "serial_nr"),
		apiStr(detail[0], "status"),
		detail[0]["product"],
		detail[0]["site"])
}

func TestFetch_ServiceInstance(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	session := fetchMCPClient(t, ctx)

	out := mcpCall(t, ctx, session, "xurrent_discover", map[string]any{"entity": "service_instance"})
	t.Logf("=== %s ===", out["name"])
	t.Logf("  path:     %s", out["path"])
	t.Logf("  doc:      %s", out["doc_url"])
	t.Logf("  unique:   %s", out["unique_key"])
	t.Logf("  defaults: %v", out["default_fields"])

	rels, _ := out["relationships"].([]any)
	for _, r := range rels {
		rm := r.(map[string]any)
		t.Logf("  → %s (%s) via %s", rm["entity"], rm["cardinality"], rm["via"])
	}

	qOut := mcpCall(t, ctx, session, "xurrent_query", map[string]any{
		"entity":   "service_instance",
		"fields":   "id,name,status",
		"per_page": float64(10),
	})
	require.Equal(t, "service_instance", qOut["entity"])
	t.Logf("  MCP query: %d records returned", int(qOut["count"].(float64)))

	client := apiClient(t)
	records := apiGet(t, ctx, client, "/v1/service_instances", "per_page=10&fields=id,name,status")
	require.GreaterOrEqual(t, len(records), 1, "expected at least 1 service instance")
	t.Logf("  records: %d", len(records))

	for i := 0; i < min(3, len(records)); i++ {
		t.Logf("  [%d] id=%d name=%s status=%s",
			i+1, apiID(records[i]), apiStr(records[i], "name"), apiStr(records[i], "status"))
	}

	firstID := apiID(records[0])
	detail := apiGet(t, ctx, client,
		fmt.Sprintf("/v1/service_instances/%d", firstID),
		"fields=id,name,status,service,support_team")
	t.Logf("  detail[%d]: id=%d name=%s status=%s service=%v support_team=%v",
		firstID,
		apiID(detail[0]),
		apiStr(detail[0], "name"),
		apiStr(detail[0], "status"),
		detail[0]["service"],
		detail[0]["support_team"])
}

func TestFetch_SLA(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	session := fetchMCPClient(t, ctx)

	out := mcpCall(t, ctx, session, "xurrent_discover", map[string]any{"entity": "service_level_agreement"})
	t.Logf("=== %s ===", out["name"])
	t.Logf("  path:     %s", out["path"])
	t.Logf("  doc:      %s", out["doc_url"])
	t.Logf("  unique:   %s", out["unique_key"])
	t.Logf("  defaults: %v", out["default_fields"])

	rels, _ := out["relationships"].([]any)
	for _, r := range rels {
		rm := r.(map[string]any)
		t.Logf("  → %s (%s) via %s", rm["entity"], rm["cardinality"], rm["via"])
	}

	qOut := mcpCall(t, ctx, session, "xurrent_query", map[string]any{
		"entity":   "service_level_agreement",
		"fields":   "id,name,customer",
		"per_page": float64(10),
	})
	require.Equal(t, "service_level_agreement", qOut["entity"])
	t.Logf("  MCP query: %d records returned", int(qOut["count"].(float64)))

	client := apiClient(t)
	records := apiGet(t, ctx, client, "/v1/slas", "per_page=10&fields=id,name,customer")
	require.GreaterOrEqual(t, len(records), 1, "expected at least 1 SLA")
	t.Logf("  records: %d", len(records))

	for i := 0; i < min(3, len(records)); i++ {
		t.Logf("  [%d] id=%d name=%s customer=%v",
			i+1, apiID(records[i]), apiStr(records[i], "name"), records[i]["customer"])
	}

	firstID := apiID(records[0])
	detail := apiGet(t, ctx, client,
		fmt.Sprintf("/v1/slas/%d", firstID),
		"fields=id,name,customer,service_offering")
	t.Logf("  detail[%d]: id=%d name=%s customer=%v service_offering=%v",
		firstID,
		apiID(detail[0]),
		apiStr(detail[0], "name"),
		detail[0]["customer"],
		detail[0]["service_offering"])
}

// ──────────────────────────────────────────────────────
// Entity Relationship Chain: traverse the ITSM graph
// ──────────────────────────────────────────────────────

func TestFetch_ServiceChain(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	client := apiClient(t)
	t.Logf("=== Service Delivery Chain ===")

	// 1. Find a service.
	svcs := apiGet(t, ctx, client, "/v1/services", "per_page=5&fields=id,name")
	require.GreaterOrEqual(t, len(svcs), 1, "need at least 1 service")
	svcID := apiID(svcs[0])
	svcName := apiStr(svcs[0], "name")
	t.Logf("Service:           id=%d name=%s", svcID, svcName)

	// 2. Find offerings for that service.
	offerings := apiGet(t, ctx, client, "/v1/service_offerings", "per_page=5&fields=id,name,service")
	t.Logf("Offerings:         %d total", len(offerings))
	if len(offerings) > 0 {
		t.Logf("  first:           id=%d name=%s service_id=%v",
			apiID(offerings[0]), apiStr(offerings[0], "name"), offerings[0]["service"])
	}

	// 3. Find SLAs.
	slas := apiGet(t, ctx, client, "/v1/slas", "per_page=5&fields=id,name,customer")
	require.GreaterOrEqual(t, len(slas), 1, "need at least 1 SLA")
	slaID := apiID(slas[0])
	slaName := apiStr(slas[0], "name")
	t.Logf("SLA:               id=%d name=%s", slaID, slaName)

	// 4. Find service instances.
	sis := apiGet(t, ctx, client, "/v1/service_instances", "per_page=5&fields=id,name,status,service,support_team")
	t.Logf("Service Instances: %d total", len(sis))
	if len(sis) > 0 {
		si := sis[0]
		t.Logf("  first:           id=%d name=%s status=%s service=%v support_team=%v",
			apiID(si), apiStr(si, "name"), apiStr(si, "status"),
			si["service"], si["support_team"])
	}

	// 5. Find CIs.
	cis := apiGet(t, ctx, client, "/v1/cis", "per_page=5&fields=id,label,serial_nr,status")
	require.GreaterOrEqual(t, len(cis), 1, "need at least 1 CI")
	ciID := apiID(cis[0])
	ciLabel := apiStr(cis[0], "label")
	ciSerial := apiStr(cis[0], "serial_nr")
	t.Logf("CI:                id=%d label=%s serial=%s", ciID, ciLabel, ciSerial)

	// 6. Find people.
	people := apiGet(t, ctx, client, "/v1/people", "per_page=5&fields=id,name,primary_email")
	require.GreaterOrEqual(t, len(people), 1, "need at least 1 person")
	personID := apiID(people[0])
	personName := apiStr(people[0], "name")
	personEmail := apiStr(people[0], "primary_email")
	t.Logf("Person:            id=%d name=%s email=%s", personID, personName, personEmail)

	// 7. Verify we can read each by ID (single-record GET).
	t.Logf("=== Verify single-record reads ===")

	svcDetail := apiGet(t, ctx, client, fmt.Sprintf("/v1/services/%d", svcID), "fields=id,name")
	t.Logf("  GET /v1/services/%d → id=%d name=%s", svcID, apiID(svcDetail[0]), apiStr(svcDetail[0], "name"))

	slaDetail := apiGet(t, ctx, client, fmt.Sprintf("/v1/slas/%d", slaID), "fields=id,name")
	t.Logf("  GET /v1/slas/%d → id=%d name=%s", slaID, apiID(slaDetail[0]), apiStr(slaDetail[0], "name"))

	ciDetail := apiGet(t, ctx, client, fmt.Sprintf("/v1/cis/%d", ciID), "fields=id,label,serial_nr,status")
	t.Logf("  GET /v1/cis/%d → id=%d label=%s serial=%s status=%s",
		ciID, apiID(ciDetail[0]), apiStr(ciDetail[0], "label"),
		apiStr(ciDetail[0], "serial_nr"), apiStr(ciDetail[0], "status"))

	personDetail := apiGet(t, ctx, client, fmt.Sprintf("/v1/people/%d", personID), "fields=id,name,primary_email")
	t.Logf("  GET /v1/people/%d → id=%d name=%s email=%s",
		personID, apiID(personDetail[0]), apiStr(personDetail[0], "name"), apiStr(personDetail[0], "primary_email"))

	t.Logf("=== Chain complete: Service → SLA → Instance → CI → Person ===")
}

func joinFields(fields []string) string {
	return strings.Join(fields, ",")
}
